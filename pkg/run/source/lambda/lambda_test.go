package lambda

import "testing"

func TestConfigureMap(t *testing.T) {
	f := New().(*lambdaSource)

	err := f.ConfigureMap(map[string]interface{}{"jsonDeserializeBody": false})
	if err != nil {
		t.Errorf("ConfigureMap returned err: %#v", err)
	}

	if f.config.JSONDeserializeBody != false {
		t.Errorf("did not set jsonDeserializeBody correctly")
	}
}
