package exec

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/systemspec/fqmn"
)

var localhost = netip.MustParseAddr(ipLocalhost)

type Config struct {
	ControlPlane string
}

type Spawn struct {
	controlPlane string
	client       *http.Client
	directory    map[string]process
	dieChan      chan exitMessage
	shutdownChan chan struct{}
	lock         *sync.Mutex
	logger       zerolog.Logger
}

func NewSpawn(c Config, l zerolog.Logger) Spawn {
	return Spawn{
		controlPlane: c.ControlPlane,
		client:       &http.Client{Timeout: clientTimeout},
		directory:    make(map[string]process),
		dieChan:      make(chan exitMessage),
		shutdownChan: make(chan struct{}),
		lock:         new(sync.Mutex),
		logger:       l.With().Str("component", "spawn").Logger(),
	}
}

func (s *Spawn) reapProcesses() {
	s.logger.Info().Msg("starting the process reaping thing")
	for {
		select {
		case em := <-s.dieChan:
			s.logger.Info().Msg("incoming message to the die channel")

			key := fmt.Sprintf(keyFormat, em.target.Tenant, em.target.Ref, em.target.Namespace, em.target.Name)

			s.logger.Warn().Str("key", key).Str("output", string(em.output)).Msg("process exited")

			delete(s.directory, key)
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *Spawn) Execute(ctx context.Context, target fqmn.FQMN, input []byte) ([]byte, error) {
	key := fmt.Sprintf(keyFormat, target.Tenant, target.Ref, target.Namespace, target.Name)

	ctx, span := tracing.Tracer.Start(ctx, "spawn.execMod", trace.WithAttributes(
		attribute.String("key", key),
	))
	defer span.End()

	var proc process
	var err error
	var found bool

	s.lock.Lock()
	proc, found = s.directory[key]
	s.lock.Unlock()
	if !found {
		span.AddEvent("key not found, launching new one")

		proc, err = s.launch(ctx, target)
		if err != nil {
			return nil, errors.Wrap(err, "s.launch")
		}

		s.directory[key] = proc
	} else {
		span.AddEvent("key found, using that one")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s/meta/sync", proc.addrPort.String()), bytes.NewReader(input))
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "s.client.Do")
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "io.Readall(resp.Body)")
	}

	return b, nil
}

func (s *Spawn) launch(ctx context.Context, target fqmn.FQMN) (process, error) {
	ctx, span := tracing.Tracer.Start(ctx, "span.launch")
	defer span.End()

	// choose a random addrPort above 10000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return process{}, errors.Wrap(err, "rand.Int")
	}

	port := uint16(randPort.Uint64() + 10000)

	// create a new uuid for the process
	procUUID := uuid.New().String()

	env := []string{
		fmt.Sprintf("SAT_HTTP_PORT=%d", port),
		fmt.Sprintf("SAT_TRACER_SERVICENAME=e2core_launch_bebby-%d", port),
		"SAT_CONTROL_PLANE=" + s.controlPlane,
		"SAT_TRACER_TYPE=collector",
		"SAT_TRACER_PROBABILITY=1",
		"SAT_TRACER_COLLECTOR_ENDPOINT=collector:4317",
		"E2CORE_UUID=" + strings.ToUpper(procUUID),
		"SAT_CONNECTIONS=",
	}

	// Create a context with a cancel with cause functionality. Instead of reaping the process by killing by process id,
	// we're going to call the cancel function for this process.
	ctx, cxl := context.WithCancelCause(ctx)

	fqmnArg, err := fqmn.FromParts(target.Tenant, target.Namespace, target.Name, target.Ref)
	if err != nil {
		return process{}, errors.Wrap(err, "fqmn.FromParts")
	}

	cmd := []string{
		"e2core",
		"mod",
		"start",
		fqmnArg,
	}

	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err = command.Start()
	if err != nil {
		return process{}, errors.Wrap(err, "command.Start()")
	}

	time.Sleep(2 * time.Second)

	span.AddEvent("launched process", trace.WithAttributes(
		attribute.Int("port", int(port)),
		attribute.Int("pid", command.Process.Pid),
		attribute.StringSlice("mod_command", cmd),
		attribute.StringSlice("env", env),
		attribute.String("procuuid", procUUID),
		attribute.String("command.String", command.String()),
	))

	p := process{
		addrPort: netip.AddrPortFrom(localhost, port),
		command:  command,
		target:   target,
		cxl:      cxl,
	}

	go p.listenForExit(s.dieChan)

	return p, nil
}
