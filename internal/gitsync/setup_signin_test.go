package gitsync_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// The sign-in is offered only where gh's access is what git uses: an HTTPS
// origin on github.com itself, with no credentials of its own in the URL.
func TestGitHubHTTPS(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		origin string
		want   bool
	}{
		{"https://github.com/ana/notes.git", true},
		{"https://github.com/ana/notes", true},
		{"HTTPS://GitHub.com/ana/notes.git", true},
		{"https://github.com:443/ana/notes.git", true},
		{" https://github.com/ana/notes.git\n", true},

		{"https://github.com.evil.example/ana/notes.git", false},
		{"https://evil.example/github.com/x", false},
		{"https://github.com@evil.example/ana/notes.git", false},
		{"https://evil.example#@github.com/ana/notes.git", false},
		{`https://github.com\@evil.example/ana/notes.git`, false},
		{"https://github.com./ana/notes.git", false},
		{"https://notgithub.com/ana/notes.git", false},
		{"https://www.github.com/ana/notes.git", false},
		{"https://github.com:8443/ana/notes.git", false},

		{"http://github.com/ana/notes.git", false},
		{"ssh://git@github.com/ana/notes.git", false},
		{"git@github.com:ana/notes.git", false},
		{"git://github.com/ana/notes.git", false},
		{"file:///github.com/ana/notes.git", false},
		{"/srv/github.com/notes.git", false},
		{"", false},

		// The URL's own credentials are what git uses: gh's access would not
		// change them.
		{"https://tok:s3cret@github.com/ana/notes.git", false},
		{"https://s3cret@github.com/ana/notes.git", false},
	} {
		if got := gitsync.GitHubHTTPS(c.origin); got != c.want {
			t.Errorf("GitHubHTTPS(%q) = %v, want %v", c.origin, got, c.want)
		}
	}
}
