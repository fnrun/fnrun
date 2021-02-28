package cli_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/cli"
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

func newSubprocessFn(t *testing.T) fn.Fn {
	t.Helper()

	commandStr := fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperSubprocess")
	env := []string{"GO_RUNNING_SUBPROCESS=1"}

	baseCmd, err := createBaseCmd(commandStr, env...)
	if err != nil {
		t.Fatalf("error creating cmd: %#v", err)
	}

	return cli.NewFromCmd(baseCmd)
}

func TestFn_Invoke_subprocessExitsUnsuccessfully(t *testing.T) {
	f := newSubprocessFn(t)
	ctx := context.Background()

	output, err := f.Invoke(ctx, "exit_error")
	if err == nil {
		t.Errorf("expected Invoke to return error, but it did not")
	}
	if output != nil {
		t.Errorf("Expected error to be nil but it was %+v", output)
	}

	t.Run("command restarts after exit", func(t *testing.T) {
		output, err := f.Invoke(ctx, "second time")
		if err != nil {
			t.Errorf("Invoke returned error: %+v", err)
		}

		want := "from subprocess: second time"
		got := output.(string)

		if got != want {
			t.Errorf("want: %q; got %q", want, got)
		}
	})
}

func TestFn_Invoke_subprocessDoesNotCrash(t *testing.T) {
	f := newSubprocessFn(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	output, err := f.Invoke(ctx, "first time")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	want := "from subprocess: first time"
	got := output.(string)
	if got != want {
		t.Errorf("first time: want %q; got %q", want, got)
	}

	output, err = f.Invoke(ctx, "second time")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	want = "from subprocess: second time"
	got = output.(string)
	if got != want {
		t.Errorf("first time: want %q; got %q", want, got)
	}
}

func TestFn_Invoke_hangingSubprocess(t *testing.T) {
	f := newSubprocessFn(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Invoke(ctx, "sleep")
	if err != context.DeadlineExceeded {
		t.Errorf("expected Invoke to return DeadlineExceeded but instead returned %+v", err)
	}
}

func TestNew_withoutConfigReturnsUnconfiguredError(t *testing.T) {
	f := cli.New()
	_, err := f.Invoke(context.Background(), "some input")
	if err != cli.ErrUnconfiguredCmd {
		t.Fatalf("expected Invoke to return ErrUnconfiguredCmd but returned: %+v", err)
	}
}

func TestNew_withConfiguration(t *testing.T) {
	configMap := map[string]interface{}{
		"command": fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperSubprocess"),
		"env":     []string{"GO_RUNNING_SUBPROCESS=1"},
	}
	f := cli.New()
	err := config.Configure(f, configMap)
	if err != nil {
		t.Fatalf("Configure returned err: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err != nil {
		t.Fatalf("Invoke returned err: %+v", err)
	}

	want := "from subprocess: some input"
	got := output.(string)

	if got != want {
		t.Errorf("Unexpected output from Invoke: want %q, got %q", want, got)
	}
}

func TestNew_withNilConfig(t *testing.T) {
	f := cli.New()
	err := config.Configure(f, nil)
	if err == nil {
		t.Errorf("expected Configure to return an error but was nil")
	}
}

// -----------------------------------------------------------------------------

func Test_HelperSubprocess(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "sleep":
			<-time.After(30 * time.Second)
			fmt.Println("from subprocess")
			break
		case "exit_error":
			fmt.Fprintln(os.Stderr, "from subprocess: exiting with error")
			os.Exit(1)
		default:
			fmt.Printf("from subprocess: %s\n", scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "received error: %#v", err)
		os.Exit(1)
	}
}
