package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
	"github.com/tessellator/executil"
)

type service struct {
	alive         bool
	baseCmd       *exec.Cmd
	cmd           *exec.Cmd
	errorChannel  chan error
	outputChannel chan string
	stdin         io.WriteCloser
	locker        sync.RWMutex
}

func (s *service) setAlive(alive bool) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	if s.alive == alive {
		return nil
	}

	s.alive = alive

	if !alive {
		return s.cmd.Process.Kill()
	}

	return nil
}

func (s *service) getAlive() bool {
	s.locker.RLock()
	defer s.locker.RUnlock()

	return s.alive
}

func (s *service) start() error {
	if s.getAlive() {
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
	s.setAlive(true)

	go scanAndLogMessages(stderr)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.outputChannel <- scanner.Text()
		}
	}()

	go func() {
		err := cmd.Wait()
		s.setAlive(false)
		if err != nil {
			s.errorChannel <- err
		}
	}()

	return nil
}

func (s *service) kill() error {
	return s.setAlive(false)
}

func (s *service) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
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

func newService(baseCmd *exec.Cmd) *service {
	return &service{
		alive:         false,
		baseCmd:       executil.CloneCmd(baseCmd),
		errorChannel:  make(chan error, 1),
		outputChannel: make(chan string, 1),
	}
}
