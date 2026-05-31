package version

import "testing"

func TestGet(t *testing.T) {
	expected := "0.1.0-dev"
	if result := Get(); result != expected {
		t.Errorf("Get() = %q, want %q", result, expected)
	}
}
