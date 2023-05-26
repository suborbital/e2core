package exec

import (
	"bytes"
	"context"
	"net/netip"
	"os/exec"
	"time"

	"github.com/pkg/errors"

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
}

type exitMessage struct {
	target fqmn.FQMN
	err    error
	output []byte
}

func (p process) listenForExit(ec chan exitMessage) {
	em := exitMessage{
		target: p.target,
	}

	err := p.command.Wait()
	if err != nil {
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
