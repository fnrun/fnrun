package cli

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/tessellator/executil"
)

type script struct {
	baseCmd *exec.Cmd
}

func (s *script) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	cmd := executil.CloneCmd(s.baseCmd)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	go scanAndLogMessages(stderr)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintln(stdin, input)
	if err != nil {
		return nil, err
	}

	outputChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		outputBytes, err := cmd.Output()
		if err != nil {
			errorChan <- err
			return
		}

		outputChan <- string(outputBytes)
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()

	case err := <-errorChan:
		cmd.Process.Kill()
		return nil, err

	case result := <-outputChan:
		return result, nil
	}
}

func newScript(baseCmd *exec.Cmd) *script {
	return &script{
		baseCmd: baseCmd,
	}
}
