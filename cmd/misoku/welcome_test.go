package main

// The welcome (P2): when misoku greets instead of opening a folder, and
// what each of its choices does. These tests change HOME, so the ones
// that run the welcome are not parallel.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// systemExample is a system folder of this OS, and one inside it.
func systemExample(t *testing.T) (string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return root, filepath.Join(root, "System32")
	}
	return "/etc", "/usr/local/bin"
}

// rootOf is the root of the file system dir is on.
func rootOf(dir string) string {
	return filepath.VolumeName(dir) + string(filepath.Separator)
}

// withNote is a new folder with a note at its first level.
func withNote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "hello.md", "# hello\n")
	return dir
}

func TestWelcomeDue(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeFile(t, home, "todo.md", "a note in the home folder\n")
	empty := t.TempDir()
	notes := withNote(t)
	deep := t.TempDir() // notes only below the first level
	writeFile(t, deep, "sub/deep.md", "deep\n")
	listed := t.TempDir()
	other := t.TempDir() // files, but no note
	writeFile(t, other, "readme.txt", "not a note\n")
	mdDir := t.TempDir() // a folder named like a note
	writeFile(t, mdDir, "old.md/inside.txt", "x\n")
	system, _ := systemExample(t)
	cfg := config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{listed: {}}}}
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cwd := func(dir string) workspaceChoice { return workspaceChoice{Dir: dir, Source: fromCwd} }
	for _, tc := range []struct {
		name                     string
		in                       welcomeCheck
		want                     bool
		noTerminal, noConfigPath bool
	}{
		{"an empty terminal folder", welcomeCheck{Choice: cwd(empty)}, true, false, false},
		{"notes only below the first level", welcomeCheck{Choice: cwd(deep)}, true, false, false},
		{"files that are not notes", welcomeCheck{Choice: cwd(other)}, true, false, false},
		{"a folder named like a note", welcomeCheck{Choice: cwd(mdDir)}, true, false, false},
		{"the remembered folder is missing", welcomeCheck{Choice: workspaceChoice{Dir: filepath.Join(empty, "gone"), Source: notesDirMissing}}, true, false, false},
		{"the home folder, even with notes", welcomeCheck{Choice: cwd(home)}, true, false, false},
		{"the root", welcomeCheck{Choice: cwd(rootOf(empty))}, true, false, false},
		{"a system folder", welcomeCheck{Choice: cwd(system)}, true, false, false},
		{"--dir", welcomeCheck{Choice: workspaceChoice{Dir: empty, Source: fromFlag}}, false, false, false},
		{"a folder argument", welcomeCheck{Choice: workspaceChoice{Dir: empty, Source: fromArg}}, false, false, false},
		{"$MISOKU_WORKSPACE", welcomeCheck{Choice: workspaceChoice{Dir: empty, Source: fromEnv}}, false, false, false},
		{"a remembered folder", welcomeCheck{Choice: workspaceChoice{Dir: empty, Source: fromNotesDir}}, false, false, false},
		{"a vault with notes", welcomeCheck{Choice: cwd(notes)}, false, false, false},
		{"a vault listed in sync.vaults", welcomeCheck{Choice: cwd(listed)}, false, false, false},
		{"an unreadable config", welcomeCheck{Choice: cwd(empty), ConfigErr: errors.New("bad json")}, false, false, false},
		{"no config file location", welcomeCheck{Choice: cwd(empty)}, false, false, true},
		{"a notes_dir that cannot be used", welcomeCheck{Choice: workspaceChoice{Dir: empty, Source: fromCwd, Warning: "Ignoring notes_dir"}}, false, false, false},
		{"no terminal", welcomeCheck{Choice: cwd(empty)}, false, true, false},
	} {
		in := tc.in
		in.Terminal = !tc.noTerminal
		if !tc.noConfigPath {
			in.ConfigPath = cfgPath
		}
		in.Config, in.Home = cfg, home
		if got := welcomeDue(in); got != tc.want {
			t.Errorf("%s: welcomeDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// "Use this folder" is offered only for a folder with notes that is not
// the home folder, the root or a system folder.
func TestUseThisFolderOnlyForASafeFolderWithNotes(t *testing.T) {
	t.Parallel()
	home := withNote(t)
	notes := withNote(t)
	listed := t.TempDir()
	cfg := config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{listed: {}}}}
	system, inside := systemExample(t)
	for _, tc := range []struct {
		name, dir string
		want      bool
	}{
		{"a folder with notes", notes, true},
		{"a folder listed in sync.vaults", listed, true},
		{"a folder without notes", t.TempDir(), false},
		{"the home folder", home, false},
		{"the root", rootOf(notes), false},
		{"a system folder", system, false},
		{"a folder inside a system folder", inside, false},
	} {
		if got := canUseFolder(cfg, tc.dir, home); got != tc.want {
			t.Errorf("%s (%s): canUseFolder = %v, want %v", tc.name, tc.dir, got, tc.want)
		}
	}
}

// Bring my notes proposes ~/Notes, or the next free ~/Notes-N when that
// folder already has something in it, so two sets of notes never mix.
func TestFreeNotesFolder(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if got := freeFolder(home, "Notes"); got != "~/Notes" {
		t.Errorf("no folder yet: %q, want ~/Notes", got)
	}
	if err := os.Mkdir(filepath.Join(home, "Notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := freeFolder(home, "Notes"); got != "~/Notes" {
		t.Errorf("an empty folder: %q, want ~/Notes", got)
	}
	writeFile(t, home, "Notes/a.md", "a\n")
	if got := freeFolder(home, "Notes"); got != "~/Notes-2" {
		t.Errorf("~/Notes has notes: %q, want ~/Notes-2", got)
	}
	writeFile(t, home, "Notes-2/b.md", "b\n")
	if got := freeFolder(home, "Notes"); got != "~/Notes-3" {
		t.Errorf("~/Notes and ~/Notes-2 have notes: %q, want ~/Notes-3", got)
	}
}

// greeting runs the welcome in a home of its own (HOME and USERPROFILE),
// with a config file of its own.
type greeting struct {
	home, cfg, missing, cwd string
}

func newGreeting(t *testing.T) *greeting {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return &greeting{home: home, cfg: filepath.Join(t.TempDir(), "config.json")}
}

// run runs the welcome with input as what the user types, and returns
// the folder to open, the exit code and what it printed.
func (g *greeting) run(t *testing.T, input string) (string, int, string) {
	t.Helper()
	var out bytes.Buffer
	dir, code := runWelcome(context.Background(), welcomeOptions{
		Stdin: strings.NewReader(input), Out: &out, Err: &out,
		ConfigPath: g.cfg, Home: g.home, Missing: g.missing, Cwd: g.cwd,
	})
	return dir, code, out.String()
}

// notesDir is notes_dir as written in the config, "" when unset.
func (g *greeting) notesDir(t *testing.T) string {
	t.Helper()
	c, err := config.Load(g.cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c.NotesDir
}

func (g *greeting) nothingSaved(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(g.cfg); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the config was written (%v)", err)
	}
	if _, err := os.Stat(filepath.Join(g.home, "Notes")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("~/Notes was created (%v)", err)
	}
}

func TestWelcomeShowsItsChoices(t *testing.T) {
	g := newGreeting(t)
	_, _, out := g.run(t, "")
	wantOutput(t, out, "Welcome to Misoku", "1) Start my notes", "2) Bring my notes from another computer")
	if strings.Contains(out, "Use this folder") || strings.Contains(out, "I can't find") {
		t.Errorf("no folder offered and none missing:\n%s", out)
	}
}

func TestStartMyNotesCreatesRemembersAndOpensTheDefaultFolder(t *testing.T) {
	g := newGreeting(t)
	dir, code, out := g.run(t, "1\n\n")
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q to open:\n%s", dir, code, want, out)
	}
	wantOutput(t, out, "Folder for your notes [~/Notes]")
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Errorf("~/Notes was not created (%v)", err)
	}
	if got := g.notesDir(t); got != "~/Notes" {
		t.Errorf("notes_dir = %q, want ~/Notes", got)
	}
}

func TestStartMyNotesTakesAnotherFolder(t *testing.T) {
	g := newGreeting(t)
	if err := os.Mkdir(filepath.Join(g.home, "Docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(t.TempDir(), "Elsewhere")
	for _, tc := range []struct{ typed, open, saved string }{
		{"~/Docs/Mine", filepath.Join(g.home, "Docs", "Mine"), "~/Docs/Mine"},
		{abs, abs, abs},
		{abs + string(filepath.Separator) + ".", abs, abs}, // saved clean
	} {
		dir, code, out := g.run(t, "1\n"+tc.typed+"\n")
		if dir != tc.open || code != 0 {
			t.Fatalf("%s: got %q exit %d, want %q:\n%s", tc.typed, dir, code, tc.open, out)
		}
		if info, err := os.Stat(tc.open); err != nil || !info.IsDir() {
			t.Errorf("%s was not created (%v)", tc.open, err)
		}
		if got := g.notesDir(t); got != tc.saved {
			t.Errorf("notes_dir = %q, want %q", got, tc.saved)
		}
	}
}

// A folder that is not one of its own (home, root), a relative one, or
// one whose parent is missing is asked for again; nothing is created.
func TestStartMyNotesAsksAgainForAFolderItCannotUse(t *testing.T) {
	g := newGreeting(t)
	missingParent := filepath.Join(g.home, "no", "such", "Notes")
	_, code, out := g.run(t, "1\n~\nNotes\n"+missingParent+"\n")
	if code != 0 {
		t.Errorf("exit %d, want 0 (cancelled at the end of the input)", code)
	}
	wantOutput(t, out, "choose a folder of its own", "Type a full path, such as ~/Notes", filepath.Join(g.home, "no", "such"))
	if _, err := os.Stat(filepath.Join(g.home, "no")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the missing parent was created (%v)", err)
	}
	g.nothingSaved(t)
}

func TestCancellingTheWelcomeSavesNothing(t *testing.T) {
	for name, input := range map[string]string{
		"Enter at the choices":         "\n",
		"end of input at the choices":  "",
		"end of input at the folder":   "1\n",
		"end of input at Bring folder": "2\n",
	} {
		t.Run(name, func(t *testing.T) {
			g := newGreeting(t)
			dir, code, out := g.run(t, input)
			if dir != "" || code != 0 {
				t.Errorf("got %q exit %d, want nothing opened, exit 0:\n%s", dir, code, out)
			}
			wantOutput(t, out, "Nothing was changed")
			g.nothingSaved(t)
		})
	}
}

func TestAWrongChoiceIsAskedAgain(t *testing.T) {
	g := newGreeting(t)
	_, _, out := g.run(t, "3\nnine\n")
	if n := strings.Count(out, "Number (Enter cancels)"); n != 3 {
		t.Errorf("asked %d times, want 3:\n%s", n, out)
	}
	wantOutput(t, out, "Type 1 or 2")
	g.nothingSaved(t)
}

// A remembered folder that is gone is named, and never created again.
func TestMissingNotesFolderIsNamedAndNotCreated(t *testing.T) {
	g := newGreeting(t)
	g.missing = filepath.Join(t.TempDir(), "Disk", "Notes")
	_, _, out := g.run(t, "\n")
	wantOutput(t, out, "I can't find your notes at "+g.missing)
	if _, err := os.Stat(filepath.Dir(g.missing)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the missing folder was created (%v)", err)
	}
	g.nothingSaved(t)
}

func TestUseThisFolderRemembersAndOpensIt(t *testing.T) {
	g := newGreeting(t)
	g.cwd = withNote(t)
	dir, code, out := g.run(t, "3\n")
	if dir != g.cwd || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, g.cwd, out)
	}
	wantOutput(t, out, "3) Use this folder: "+g.cwd)
	if got := g.notesDir(t); got != g.cwd {
		t.Errorf("notes_dir = %q, want %q", got, g.cwd)
	}
}

// Bring my notes proposes ~/Notes-2 when ~/Notes already has notes.
func TestBringMyNotesProposesAFreeFolder(t *testing.T) {
	g := newGreeting(t)
	writeFile(t, g.home, "Notes/mine.md", "mine\n")
	_, _, out := g.run(t, "2\n")
	wantOutput(t, out, "Folder for your notes [~/Notes-2]")
}

// What the welcome names and offers, from the folder misoku would open.
func TestWelcomeForNamesTheMissingFolderAndOffersASafeOne(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	notes := withNote(t)
	gone := filepath.Join(t.TempDir(), "gone")
	cfg := config.Config{}
	for _, tc := range []struct {
		name          string
		choice        workspaceChoice
		getwd         func() (string, error)
		missing, offr string
	}{
		{"terminal folder with notes", workspaceChoice{Dir: notes, Source: fromCwd}, nil, "", notes},
		{"terminal folder is home", workspaceChoice{Dir: home, Source: fromCwd}, nil, "", ""},
		{"missing, terminal folder with notes", workspaceChoice{Dir: gone, Source: notesDirMissing}, cwdIs(notes), gone, notes},
		{"missing, empty terminal folder", workspaceChoice{Dir: gone, Source: notesDirMissing}, cwdIs(t.TempDir()), gone, ""},
		{"missing, no terminal folder", workspaceChoice{Dir: gone, Source: notesDirMissing}, func() (string, error) { return "", errors.New("gone") }, gone, ""},
	} {
		o := welcomeFor(tc.choice, cfg, home, tc.getwd)
		if o.Missing != tc.missing || o.Cwd != tc.offr || o.Home != home {
			t.Errorf("%s: missing %q offer %q, want %q and %q", tc.name, o.Missing, o.Cwd, tc.missing, tc.offr)
		}
	}
}

// An unreadable config is still reported and skipped, as before the
// welcome: misoku goes on to the folder it was given.
func TestUnreadableConfigIsStillIgnored(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	path := configIn(home)
	writeFile(t, filepath.Dir(path), filepath.Base(path), "{not json")
	target := touch(t, filepath.Join(t.TempDir(), "notes"))
	code, out := runProbe(t, home, t.TempDir(), []string{"MISOKU_WORKSPACE=" + target})
	if code != 1 || !strings.Contains(out, "Ignoring unreadable config") || !strings.Contains(out, "Could not open workspace") {
		t.Errorf("exit %d, want 1 after ignoring the config:\n%s", code, out)
	}
}
