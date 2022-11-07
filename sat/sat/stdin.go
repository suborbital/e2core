package sat

import (
	"bufio"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/request"
	"github.com/suborbital/vektor/vk"
)

// ExecFromStdin reads stdin, passes the data through the registered module, and writes the result to stdout.
func (s *Sat) ExecFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scanner.Scan")
	}

	input := scanner.Bytes()

	ctx := vk.NewCtx(s.log, nil, nil)

	// construct a fake HTTP request from the input
	req := &request.CoordinatedRequest{
		Method:      http.MethodPost,
		URL:         "/",
		ID:          ctx.RequestID(),
		Body:        input,
		Headers:     map[string]string{},
		RespHeaders: map[string]string{},
		Params:      map[string]string{},
		State:       map[string][]byte{},
	}

	result, err := s.exec.Do(s.jobName, req, ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	resp := result.(*request.CoordinatedResponse)

	fmt.Println(string(resp.Output))

	return nil
}
