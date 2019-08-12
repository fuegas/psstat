package utils

import (
	"testing"
)

func TestEscape(t *testing.T) {
	res := Escape("tmux: server")
	if res != "tmux:\\ server" {
		t.Errorf("Escape was incorrect, got: %s, want: %s.", res, "tmux:\\ server")
	}
}
