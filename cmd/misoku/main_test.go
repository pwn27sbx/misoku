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

func TestParseTmuxOuter(t *testing.T) {
	cases := []struct {
		name       string
		clientTerm string
		globalEnv  string
		wantTerm   string
		wantProg   string
		wantHerdr  string
	}{
		{
			name:       "ghostty client and global env",
			clientTerm: "xterm-ghostty\n",
			globalEnv:  "TERM_PROGRAM=ghostty\n-DISPLAY\nPATH=/usr/bin\n",
			wantTerm:   "xterm-ghostty",
			wantProg:   "ghostty",
		},
		{
			name:       "wezterm global env without client_termname",
			clientTerm: "",
			globalEnv:  "TERM_PROGRAM=WezTerm\n",
			wantProg:   "WezTerm",
		},
		{
			name:       "unset TERM_PROGRAM ignored",
			clientTerm: "xterm-kitty\n",
			globalEnv:  "-TERM_PROGRAM\nHERDR_ENV=1\n",
			wantTerm:   "xterm-kitty",
			wantHerdr:  "1",
		},
		{
			name: "empty outputs",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := parseTmuxOuter(tc.clientTerm, tc.globalEnv)
			if got := env("TERM"); got != tc.wantTerm {
				t.Errorf("TERM = %q, want %q", got, tc.wantTerm)
			}
			if got := env("TERM_PROGRAM"); got != tc.wantProg {
				t.Errorf("TERM_PROGRAM = %q, want %q", got, tc.wantProg)
			}
			if got := env("HERDR_ENV"); got != tc.wantHerdr {
				t.Errorf("HERDR_ENV = %q, want %q", got, tc.wantHerdr)
			}
		})
	}
}
