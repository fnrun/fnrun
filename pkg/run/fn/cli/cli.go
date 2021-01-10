// Package cli provides an Fn that runs an external command. The Fn will run the
// command as an OS process and restart the command if it exits. The Fn will
// provide input over stdin and read output from stdout. Inputs and outputs
// should be expressed as single-line strings. Additionally, the Fn will read
// from stderr and log it.
//
// Although the Fn will restart a command if it crashes, this functionality
// should not be used to execute applications that exit successfully after
// processing a single input. Doing so will cause a race condition within the
// Fn as it takes nonzero time to detect that the application has exited
// successfully and can be in a bad state if a subsequent Invoke call is made
// before the process is cleaned up properly.
//
// Systems that use this as the base Fn will have developers write functions as
// CLI applications using any technology that can read and write standard
// standard streams.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/pkg/errors"
	"github.com/tessellator/executil"
)

type simple struct {
	alive         bool
	baseCmd       *exec.Cmd
	cmd           *exec.Cmd
	errorChannel  chan error
	outputChannel chan string
	stdin         io.WriteCloser
}

func (s *simple) start() error {
	if s.alive {
		return nil
	}

	cmd := executil.CloneCmd(s.baseCmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	s.cmd = cmd
	s.stdin = stdin
	s.alive = true

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("error reading message from cmd over stderr: %s\n", err)
		}
		if err := stderr.Close(); err != nil {
			log.Println(err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.outputChannel <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			s.errorChannel <- err
		}
	}()

	go func() {
		err := cmd.Wait()
		s.alive = false
		if err != nil {
			s.errorChannel <- err
		}
	}()

	return nil
}

func (s *simple) kill() error {
	if s.alive {
		return s.cmd.Process.Kill()
	}

	return nil
}

func (s *simple) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	if err := s.start(); err != nil {
		return nil, err
	}

	_, err := fmt.Fprintln(s.stdin, input)
	if err != nil {
		return nil, err
	}

	select {
	case response := <-s.outputChannel:
		return response, nil
	case <-ctx.Done():
		if err := s.kill(); err != nil {
			return nil, err
		}
		return nil, ctx.Err()
	case err = <-s.errorChannel:
		if kerr := s.kill(); kerr != nil {
			return nil, errors.Wrap(err, kerr.Error())
		}
		return nil, err
	}
}

// NewFromCmd builds an Fn based on baseCmd.
func NewFromCmd(baseCmd *exec.Cmd) fn.Fn {
	return &simple{
		alive:         false,
		baseCmd:       executil.CloneCmd(baseCmd),
		errorChannel:  make(chan error, 1),
		outputChannel: make(chan string, 1),
	}
}
