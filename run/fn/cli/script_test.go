package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestScript_Invoke(t *testing.T) {
	commandStr := fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperScript")
	env := []string{"GO_RUNNING_SUBPROCESS=1"}

	baseCmd, err := createBaseCmd(commandStr, env...)
	if err != nil {
		t.Fatalf("createBaseCmd returned error: %+v", err)
	}

	s := newScript(baseCmd)

	iterations := 100
	for i := 0; i < iterations; i++ {
		output, err := s.Invoke(context.Background(), "some input")
		if err != nil {
			t.Fatalf("Invoke returned error on iteration %d: %+v", i, err)
		}

		outputStr := output.(string)

		want := "from subprocess: some input"
		got := strings.Split(outputStr, "\n")[0]

		if got != want {
			t.Errorf("Unexpected output from Invoke on iteration %d: want %q, got %q", i, want, got)
		}
	}
}

func TestScript_Invoke_crashingProcess(t *testing.T) {
	commandStr := fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperScript")
	env := []string{"GO_RUNNING_SUBPROCESS=1"}

	baseCmd, err := createBaseCmd(commandStr, env...)
	if err != nil {
		t.Fatalf("createBaseCmd returned error: %+v", err)
	}

	s := newScript(baseCmd)

	output, err := s.Invoke(context.Background(), "bad exit")
	if err == nil {
		t.Errorf("expected error from Invoke but did not receive one")
	}
	if output != nil {
		t.Errorf("expected nil output but received: %#v", output)
	}
}

func TestScript_Invoke_logsStderr(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	commandStr := fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperScript")
	env := []string{"GO_RUNNING_SUBPROCESS=1"}

	baseCmd, err := createBaseCmd(commandStr, env...)
	if err != nil {
		t.Fatalf("createBaseCmd returned error: %+v", err)
	}

	s := newScript(baseCmd)

	s.Invoke(context.Background(), "bad exit")

	want := "bad exit on command!\n"
	got := buf.String()

	if got != want {
		t.Errorf("did not capture log statement: want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------

func Test_HelperScript(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	ok := scanner.Scan()
	if !ok {
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "received error: %+v\n", err)
		}
		os.Exit(1)
	}

	input := scanner.Text()
	if input == "bad exit" {
		fmt.Fprintln(os.Stderr, "bad exit on command!")
		fmt.Println("from subprocess: bad exit!")
		os.Exit(1)
	}

	if input == "sleep" {
		<-time.After(100 * time.Millisecond)
	}

	fmt.Printf("from subprocess: %s\n", scanner.Text())
}
