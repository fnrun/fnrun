package config

import (
	"reflect"
	"testing"
	"time"
)

func TestConfigure_stringValue(t *testing.T) {
	e := &everythingConfigurable{}
	err := Configure(e, "some string")
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithString
	want := "some string"

	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestConfigure_intValue(t *testing.T) {
	e := &everythingConfigurable{}
	err := Configure(e, 123)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithInt
	want := 123

	if got != want {
		t.Errorf("want %d, got %d", want, got)
	}
}

func TestConfigure_floatValue(t *testing.T) {
	e := &everythingConfigurable{}
	err := Configure(e, 1.234)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithFloat
	want := 1.234

	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestConfigure_boolValue(t *testing.T) {
	e := &everythingConfigurable{}
	err := Configure(e, true)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithBool
	want := true

	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestConfigure_mapValue(t *testing.T) {
	e := &everythingConfigurable{}
	want := map[string]interface{}{"key": 123}
	err := Configure(e, want)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithMap

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestConfigure_arrayValue(t *testing.T) {
	e := &everythingConfigurable{}
	want := []interface{}{"key", 123}
	err := Configure(e, want)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithArray

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestConfigure_genericValue(t *testing.T) {
	e := &everythingConfigurable{}
	want := 30 * time.Second
	err := Configure(e, want)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	got := e.ConfiguredWithGeneric

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestConfigure_nilValue(t *testing.T) {
	e := &everythingConfigurable{}
	err := Configure(e, nil)
	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}

	if !e.ConfiguredWithNil {
		t.Errorf("expected object to be configured with nil value but was not")
	}
}

func TestConfigure_requiresConfig(t *testing.T) {
	s := &stringConfigurable{Required: true}
	err := Configure(s, nil)

	if err == nil {
		t.Errorf("Configure expected to return an error but did not")
	}
}

func TestConfigure_doesNotRequireConfig(t *testing.T) {
	s := &stringConfigurable{Required: false}
	err := Configure(s, nil)

	if err != nil {
		t.Errorf("Configure returned error: %s", err)
	}
}

func TestGetSinglePair_noEntries(t *testing.T) {
	_, _, err := GetSinglePair(map[string]interface{}{})
	if err == nil {
		t.Error("expected GetSinglePair to return an error but it did not")
	}
}

func TestGetSinglePair_manyEntries(t *testing.T) {
	_, _, err := GetSinglePair(map[string]interface{}{
		"first key":  "first value",
		"second key": "second value",
	})
	if err == nil {
		t.Error("expected GetSinglePair to return an error but it did not")
	}
}

func TestGetSinglePair_singleEntry(t *testing.T) {
	wantKey := "first key"
	wantVal := "first value"

	gotKey, gotVal, err := GetSinglePair(map[string]interface{}{
		wantKey: wantVal,
	})
	if err != nil {
		t.Errorf("GetSinglePair returned err: %+v", err)
	}

	if gotKey != wantKey {
		t.Errorf("keys did not match: want %q, got %q", wantKey, gotKey)
	}

	if gotVal != wantVal {
		t.Errorf("values did not match: want %q, got %q", wantKey, gotKey)
	}
}

// -----------------------------------------------------------------------------
// Test types

type everythingConfigurable struct {
	ConfiguredWithNil     bool
	ConfiguredWithString  string
	ConfiguredWithInt     int
	ConfiguredWithFloat   float64
	ConfiguredWithBool    bool
	ConfiguredWithMap     map[string]interface{}
	ConfiguredWithArray   []interface{}
	ConfiguredWithGeneric interface{}
}

func (e *everythingConfigurable) RequiresConfig() bool {
	return true
}

func (e *everythingConfigurable) ConfigureString(value string) error {
	e.ConfiguredWithString = value
	return nil
}

func (e *everythingConfigurable) ConfigureInteger(value int) error {
	e.ConfiguredWithInt = value
	return nil
}

func (e *everythingConfigurable) ConfigureFloat(value float64) error {
	e.ConfiguredWithFloat = value
	return nil
}

func (e *everythingConfigurable) ConfigureBool(value bool) error {
	e.ConfiguredWithBool = value
	return nil
}

func (e *everythingConfigurable) ConfigureMap(value map[string]interface{}) error {
	e.ConfiguredWithMap = value
	return nil
}

func (e *everythingConfigurable) ConfigureArray(value []interface{}) error {
	e.ConfiguredWithArray = value
	return nil
}

func (e *everythingConfigurable) ConfigureGeneric(value interface{}) error {
	e.ConfiguredWithGeneric = value
	return nil
}

func (e *everythingConfigurable) Configure() error {
	e.ConfiguredWithNil = true
	return nil
}

// -------------------------------------

type stringConfigurable struct {
	Required bool
	Value    string
}

func (o *stringConfigurable) RequiresConfig() bool {
	return o.Required
}

func (o *stringConfigurable) ConfigureString(value string) error {
	o.Value = value
	return nil
}
