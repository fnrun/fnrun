// Package tap provides a middleware that broadcasts inputs, outputs, and errors
// to an external program. The middleware sends messages in a fire-and-forget
// manner; it does not expect a response from the external program.
package tap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/mitchellh/mapstructure"
	"github.com/tessellator/executil"
)

// ErrUnconfiguredCmd indicates that the middleware has not been configured with
// a command to run an external process.
var ErrUnconfiguredCmd = fmt.Errorf("tap: unconfigured command")

func createBaseCmd(commandStr string, env ...string) (*exec.Cmd, error) {
	cmd, err := executil.ParseCmd(commandStr)
	if err != nil {
		return nil, err
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	return cmd, nil
}

type tapMiddleware struct {
	alive     bool
	baseCmd   *exec.Cmd
	cmd       *exec.Cmd
	locker    sync.RWMutex
	stdin     io.WriteCloser
	tapInput  bool
	tapOutput bool
	tapError  bool
}

func (m *tapMiddleware) RequiresConfig() bool {
	return true
}

func (m *tapMiddleware) ConfigureString(config string) error {
	cmd, err := createBaseCmd(config)
	if err != nil {
		return err
	}

	m.baseCmd = cmd

	return nil
}

func (m *tapMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	cfg := struct {
		Command   string   `mapstructure:"command"`
		Env       []string `mapstructure:"env"`
		TapInput  bool     `mapstructure:"tapInput,omitempty"`
		TapOutput bool     `mapstructure:"tapOutput,omitempty"`
		TapError  bool     `mapstructure:"tapError,omitempty"`
	}{
		TapInput:  true,
		TapOutput: true,
		TapError:  true,
	}

	err := mapstructure.Decode(configMap, &cfg)
	if err != nil {
		return err
	}

	baseCmd, err := createBaseCmd(cfg.Command, cfg.Env...)
	if err != nil {
		return err
	}

	m.baseCmd = baseCmd
	m.tapInput = cfg.TapInput
	m.tapOutput = cfg.TapOutput
	m.tapError = cfg.TapError

	return nil
}

func (m *tapMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	if m.baseCmd == nil {
		return nil, ErrUnconfiguredCmd
	}

	err := m.start()
	if err != nil {
		return nil, err
	}

	m.locker.RLock()
	defer m.locker.RUnlock()

	// ignore errors in the following because tapping is best effort

	if m.tapInput {
		_, _ = fmt.Fprintln(m.stdin, input)
	}

	output, err := f.Invoke(ctx, input)

	if m.tapOutput {
		_, _ = fmt.Fprintln(m.stdin, output)
	}

	if err != nil && m.tapError {
		_, _ = fmt.Fprintf(m.stdin, "%+v\n", err)
	}

	return output, err
}

func (m *tapMiddleware) start() error {
	if m.getAlive() {
		return nil
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	// to prevent a race of two threads contending for a lock to start
	if m.alive {
		return nil
	}

	cmd := executil.CloneCmd(m.baseCmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	m.cmd = cmd
	m.stdin = stdin
	m.alive = true

	go func() {
		_ = cmd.Wait()
		m.locker.Lock()
		m.alive = false
		m.locker.Unlock()
	}()

	return nil
}

func (m *tapMiddleware) getAlive() bool {
	m.locker.RLock()
	defer m.locker.RUnlock()

	return m.alive
}

func New() run.Middleware {
	return &tapMiddleware{
		tapInput:  true,
		tapOutput: true,
		tapError:  true,
	}
}
