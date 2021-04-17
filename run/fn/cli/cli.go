// Package cli provides an Fn that runs an external command. If the command is
// configured to be run as a script, a new instance of the script will be
// created for each invocation. Otherwise, the command is expected to be long-
// running and service many inputs, though it will only provided a single input
// to process at a time. This Fn provides process-level isolation. If a long-
// running command exits, it will be restarted.
//
// The cli fn will provide input over stdin and read output from stdout. Inputs
// and outputs should be expressed as single-line strings. Additionally, the Fn
// will read from stderr and log it to the runner's stdout stream.
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

	"github.com/fnrun/fnrun/fn"
	"github.com/mitchellh/mapstructure"
	"github.com/tessellator/executil"
)

func createBaseCmd(commandStr string, env ...string) (*exec.Cmd, error) {
	cmd, err := executil.ParseCmd(commandStr)
	if err != nil {
		return nil, err
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	return cmd, nil
}

func scanAndLogMessages(stream io.ReadCloser) error {
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}

	return scanner.Err()
}

// ErrUnconfiguredCmd indicates that the CLI Fn has not been configured with
// a command to run an external process.
var ErrUnconfiguredCmd = fmt.Errorf("cli: unconfigured command")

type cliFn struct {
	f fn.Fn
}

func (c *cliFn) RequiresConfig() bool {
	return true
}

func (c *cliFn) ConfigureString(commandStr string) error {
	cmd, err := createBaseCmd(commandStr)
	if err != nil {
		return err
	}

	c.f = newService(cmd)
	return nil
}

func (c *cliFn) ConfigureMap(configMap map[string]interface{}) error {
	cfg := struct {
		Command string   `mapstructure:"command"`
		Env     []string `mapstructure:"env"`
		Script  bool     `mapstructure:"script"`
	}{}
	err := mapstructure.Decode(configMap, &cfg)
	if err != nil {
		return err
	}

	baseCmd, err := createBaseCmd(cfg.Command, cfg.Env...)
	if err != nil {
		return err
	}

	if cfg.Script {
		c.f = newScript(baseCmd)
		return nil
	}

	c.f = newService(baseCmd)
	return nil
}

func (c *cliFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	if c.f == nil {
		return nil, ErrUnconfiguredCmd
	}

	return c.f.Invoke(ctx, input)
}

// New creates an unconfigured Fn. The result of this function must be
// configured with a command string, otherwise ErrUnconfiguredCmd will be
// returned from calls to Invoke.
func New() fn.Fn {
	return &cliFn{}
}
