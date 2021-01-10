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
	"os"
	"os/exec"
	"sync"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/tessellator/executil"
)

func createBaseCmd(commandStr string, env ...string) (*exec.Cmd, error) {
	cmd, err := executil.ParseCmd(commandStr)
	if err != nil {
		return nil, err
	}

	cmd.Env = os.Environ()
	for _, envVar := range env {
		cmd.Env = append(cmd.Env, envVar)
	}

	return cmd, nil
}

type simple struct {
	alive         bool
	baseCmd       *exec.Cmd
	cmd           *exec.Cmd
	errorChannel  chan error
	outputChannel chan string
	stdin         io.WriteCloser
	locker        sync.RWMutex
}

func (s *simple) setAlive(alive bool) error {
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

func (s *simple) getAlive() bool {
	s.locker.RLock()
	defer s.locker.RUnlock()

	return s.alive
}

// ErrUnconfiguredCmd indicates that the CLI Fn has not been configured with
// a command to run an external process.
var ErrUnconfiguredCmd = fmt.Errorf("cli: unconfigured command")

func (s *simple) start() error {
	if s.getAlive() {
		return nil
	}
	if s.baseCmd == nil {
		return ErrUnconfiguredCmd
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
		s.setAlive(false)
		if err != nil {
			s.errorChannel <- err
		}
	}()

	return nil
}

func (s *simple) kill() error {
	return s.setAlive(false)
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

func (s *simple) RequiresConfig() bool {
	return true
}

func (s *simple) ConfigureString(commandStr string) error {
	cmd, err := createBaseCmd(commandStr)
	if err != nil {
		return err
	}

	s.baseCmd = cmd
	return nil
}

func (s *simple) ConfigureMap(configMap map[string]interface{}) error {
	cfg := struct {
		Command string   `mapstructure:"command"`
		Env     []string `mapstructure:"env"`
	}{}
	err := mapstructure.Decode(configMap, &cfg)
	if err != nil {
		return err
	}

	baseCmd, err := createBaseCmd(cfg.Command, cfg.Env...)
	if err != nil {
		return err
	}

	s.baseCmd = baseCmd
	return nil
}

// New creates an unconfigured Fn. The result of this function must be
// configured with a command string, otherwise ErrUnconfiguredCmd will be
// returned from calls to Invoke.
func New() fn.Fn {
	return &simple{
		alive:         false,
		errorChannel:  make(chan error, 1),
		outputChannel: make(chan string, 1),
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
