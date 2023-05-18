package exec

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/systemspec/fqmn"
)

const (
	clientTimeout = 30 * time.Second
	keyFormat     = "spawn://%s/%s/%s/%s"
	fqmnFormat    = "fqmn:/%s"
	ipLocalhost   = "127.0.0.1"
)

var localhost = netip.MustParseAddr(ipLocalhost)

type Config struct {
	ControlPlane string
}

type Spawn struct {
	controlPlane string
	client       *http.Client
	directory    map[string]process
	lock         *sync.Mutex
}

type process struct {
	addrPort netip.AddrPort
	pid      int
	wait     func() error
	cxl      context.CancelCauseFunc
}

func NewSpawn(c Config) Spawn {
	return Spawn{
		controlPlane: c.ControlPlane,
		client:       &http.Client{Timeout: clientTimeout},
		directory:    make(map[string]process),
		lock:         new(sync.Mutex),
	}
}

func (s *Spawn) Execute(ctx context.Context, target fqmn.FQMN, input []byte) ([]byte, error) {
	ctx, span := tracing.Tracer.Start(ctx, "spawn.execMod")
	defer span.End()

	key := fmt.Sprintf(keyFormat, target.Tenant, target.Ref, target.Namespace, target.Name)

	var proc process
	var err error
	var found bool

	s.lock.Lock()
	proc, found = s.directory[key]
	s.lock.Unlock()
	if !found {
		span.AddEvent("key not found, launching new one", trace.WithAttributes(
			attribute.String("key", key),
		))

		proc, err = s.launch(ctx, target)
		if err != nil {
			return nil, errors.Wrap(err, "s.launch")
		}

		s.directory[key] = proc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s/meta/syncmessage", proc.addrPort.String()), bytes.NewReader(input))
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "s.client.Do")
	}

	defer resp.Body.Close()

	return nil, nil
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

	command := exec.CommandContext(ctx, "e2core", "mod", "start", fmt.Sprintf(fqmnFormat, target.URLPath()))
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err = command.Start()
	if err != nil {
		return process{}, errors.Wrap(err, "command.Start()")
	}

	span.AddEvent("launched process", trace.WithAttributes(
		attribute.Int("port", int(port)),
		attribute.Int("pid", command.Process.Pid),
		attribute.String("procuuid", procUUID),
	))

	return process{
		addrPort: netip.AddrPortFrom(localhost, port),
		pid:      command.Process.Pid,
		wait:     command.Wait,
		cxl:      cxl,
	}, nil
}
