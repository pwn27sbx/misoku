package main

import "testing"

// tmux answers "on", "all" or "off"; only the first two let graphics through.
func TestPassthroughOn(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{{"on\n", true}, {"all\n", true}, {"ON", true}, {"off\n", false}, {"", false}, {"garbage", false}} {
		if got := passthroughOn(c.in); got != c.want {
			t.Errorf("passthroughOn(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
