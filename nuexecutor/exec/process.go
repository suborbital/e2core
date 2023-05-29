package exec

import (
	"bytes"
	"context"
	"net/netip"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/systemspec/fqmn"
)

const (
	clientTimeout = 30 * time.Second
	keyFormat     = "spawn://%s/%s/%s/%s"
	fqmnFormat    = "fqmn:/%s"
	ipLocalhost   = "127.0.0.1"
)

type process struct {
	addrPort netip.AddrPort
	command  *exec.Cmd
	target   fqmn.FQMN
	cxl      context.CancelCauseFunc
	logger   zerolog.Logger
}

type exitMessage struct {
	target fqmn.FQMN
	err    error
	output []byte
}

func (p process) listenForExit(ec chan exitMessage) {
	p.logger.Info().Msg("listening for exit")

	em := exitMessage{
		target: p.target,
	}

	p.logger.Info().Msg("waiting on command exit")
	err := p.command.Wait()
	if err != nil {
		p.logger.Err(err).Msg("p.command.wait returned error")
		em.err = errors.Wrap(err, "command.Wait returned an error")
	}

	op := bytes.Buffer{}

	output, err := p.command.CombinedOutput()
	if err != nil {
		op.WriteString(errors.Wrap(err, "p.command.CombinedOutput").Error())
	}
	op.Write(output)

	em.output = op.Bytes()

	ec <- em
}