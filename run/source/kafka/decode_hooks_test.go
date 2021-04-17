package kafka

import (
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/mitchellh/mapstructure"
)

func TestStringToBalanceStrategyHookFunc(t *testing.T) {
	tests := []struct {
		input    string
		expected sarama.BalanceStrategy
	}{
		{"sticky", sarama.BalanceStrategySticky},
		{"roundrobin", sarama.BalanceStrategyRoundRobin},
		{"range", sarama.BalanceStrategyRange},
	}

	for _, example := range tests {
		output := &struct {
			Other    string
			Strategy sarama.BalanceStrategy
		}{}

		decoderConfig := &mapstructure.DecoderConfig{
			Metadata:   nil,
			Result:     output,
			DecodeHook: stringToBalanceStrategyHookFunc(),
		}

		decoder, err := mapstructure.NewDecoder(decoderConfig)
		if err != nil {
			t.Fatalf("NewDecoder returned error: %+v", err)
		}

		err = decoder.Decode(map[string]interface{}{
			"strategy": example.input,
			"other":    "some other string",
		})
		if err != nil {
			t.Fatalf("Decode returned error: %+v", err)
		}

		got := output.Strategy
		want := example.expected

		if got != want {
			t.Errorf("Undesired output: want %v, got %v", want, got)
		}
	}
}

func TestStringToBalanceStrategyHookFunc_unrecognizedInput(t *testing.T) {
	output := &struct {
		Strategy sarama.BalanceStrategy
	}{}

	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     output,
		DecodeHook: stringToBalanceStrategyHookFunc(),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatalf("NewDecoder returned error: %+v", err)
	}

	err = decoder.Decode(map[string]interface{}{
		"strategy": "someothervalue",
	})
	if err == nil {
		t.Fatal("expected decoder.Decode to return an error but it did not")
	}

	got := err.Error()
	want := `unrecognized balance strategy: "someothervalue"`

	if !strings.Contains(got, want) {
		t.Errorf("unexpected error message: wanted to contain %q, got %q", want, got)
	}
}

func TestStringToKafkaVersionHookFunc(t *testing.T) {
	output := &struct {
		Other   string
		Version sarama.KafkaVersion
	}{}

	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     output,
		DecodeHook: stringToKafkaVersionHookFunc(),
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatalf("NewDecoder returned error: %+v", err)
	}

	err = decoder.Decode(map[string]interface{}{
		"version": "2.1.1",
		"other":   "some other value",
	})
	if err != nil {
		t.Fatalf("Decode returned error: %+v", err)
	}

	gotOther := output.Other
	wantOther := "some other value"
	if gotOther != wantOther {
		t.Errorf("incorrect Other value: want %q, got %q", wantOther, gotOther)
	}

	want := "2.1.1"
	got := output.Version.String()
	if want != got {
		t.Errorf("incorrect Version value: want %q, got %q", want, got)
	}
}
