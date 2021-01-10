package debug

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	m := New()
	dm := m.(*debugMiddleware)

	if dm.PrintEnabled != true {
		t.Error("expected default PrintEnabled to be true")
	}

	dm.ConfigureBool(false)

	if dm.PrintEnabled != false {
		t.Error("expected PrintEnabled to be configured with false")
	}
}

func readLines(buf *bytes.Buffer) ([]string, error) {
	lines := []string{}

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return lines, nil
			}

			return lines, err
		}

		trimmed := strings.Trim(line, "\n")

		lines = append(lines, trimmed)
	}
}

func TestInvoke(t *testing.T) {
	m := New()
	f := &sampleFn{hasErr: false}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	_, err := m.Invoke(context.Background(), "input", f)
	if err != nil {
		t.Errorf("Invoke returned error: %#v", err)
	}

	gotLines, err := readLines(&buf)
	if err != nil {
		t.Fatalf("received error reading lines from output: %#v", err)
	}

	wantLines := []string{
		`debugMiddleware: Handling input "input"`,
		`debugMiddleware: Received output "output"`,
	}

	if !reflect.DeepEqual(gotLines, wantLines) {
		t.Errorf("lines do not match: want %#v; got %#v", wantLines, gotLines)
	}
}

func TestInvoke_withError(t *testing.T) {
	m := New()
	f := &sampleFn{hasErr: true}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	_, err := m.Invoke(context.Background(), "input", f)
	if err == nil {
		t.Errorf("expected Invoke to return error")
	}

	gotLines, err := readLines(&buf)
	if err != nil {
		t.Fatalf("received error reading lines from output: %#v", err)
	}

	wantLines := []string{
		`debugMiddleware: Handling input "input"`,
		`debugMiddleware: Received output <nil>`,
		`debugMiddleware: Received error "has error"`,
	}

	if !reflect.DeepEqual(gotLines, wantLines) {
		t.Errorf("lines do not match:\nwant %#v\ngot  %#v", wantLines, gotLines)
	}
}

func TestInvoke_printDisabled(t *testing.T) {
	m := NewWithPrintEnabled(false)
	f := &sampleFn{hasErr: false}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	_, err := m.Invoke(context.Background(), "input", f)
	if err != nil {
		t.Errorf("Invoke returned error: %#v", err)
	}

	gotLines, err := readLines(&buf)
	if err != nil {
		t.Fatalf("received error reading lines from output: %#v", err)
	}

	wantLines := []string{}

	if !reflect.DeepEqual(gotLines, wantLines) {
		t.Errorf("lines do not match: want %#v; got %#v", wantLines, gotLines)
	}
}

// -----------------------------------------------------------------------------
// Sample function

type sampleFn struct {
	hasErr bool
}

func (s *sampleFn) Invoke(context.Context, interface{}) (interface{}, error) {
	if s.hasErr {
		return nil, errors.New("has error")
	}

	return "output", nil
}
