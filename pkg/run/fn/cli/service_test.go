package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
)

func newSubprocessFn(t *testing.T) fn.Fn {
	t.Helper()

	commandStr := fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperSubprocess")
	env := []string{"GO_RUNNING_SUBPROCESS=1"}

	baseCmd, err := createBaseCmd(commandStr, env...)
	if err != nil {
		t.Fatalf("error creating cmd: %#v", err)
	}

	return newService(baseCmd)
}

func TestService_Invoke(t *testing.T) {
	f := newSubprocessFn(t)

	iterations := 1000
	for i := 0; i < iterations; i++ {
		output, err := f.Invoke(context.Background(), "some input")
		if err != nil {
			t.Fatalf("Invoke returned err on iteration %d: %+v", i, err)
		}

		want := "from subprocess: some input"
		got := output.(string)

		if got != want {
			t.Errorf("Unexpected output from Invoke on iteration %d: want %q, got %q", i, want, got)
		}
	}
}

func TestService_Invoke_subprocessExitsUnsuccessfully(t *testing.T) {
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

func TestService_Invoke_subprocessDoesNotCrash(t *testing.T) {
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

func TestService_Invoke_hangingSubprocess(t *testing.T) {
	f := newSubprocessFn(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Invoke(ctx, "sleep")
	if err != context.DeadlineExceeded {
		t.Errorf("expected Invoke to return DeadlineExceeded but instead returned %+v", err)
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
