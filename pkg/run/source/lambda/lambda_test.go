package lambda

import "testing"

func TestConfigureMap(t *testing.T) {
	f := New().(*lambdaSource)

	err := f.ConfigureMap(map[string]interface{}{"jsonDeserializeEvent": false})
	if err != nil {
		t.Errorf("ConfigureMap returned err: %#v", err)
	}

	if f.config.JSONDeserializeEvent != false {
		t.Errorf("did not set jsonDeserializeEvent correctly")
	}
}
