// Package theme provides application-wide design system tokens, color palettes,
// and lipgloss style constructors.
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Theme encapsulates design system colors. The palette mirrors NvChad's
// base46 themes (onedark by default) so the UI reads like an NvChad session.
type Theme struct {
	Name  string
	Light bool

	// Base surfaces (darkest to lightest).
	DarkerBg  lipgloss.Color // tree / tabufline fill
	Bg        lipgloss.Color // editor background
	Bg2       lipgloss.Color // inactive tab, input fields
	OneBg     lipgloss.Color // cursorline, selection
	OneBg2    lipgloss.Color
	OneBg3    lipgloss.Color
	LightBg   lipgloss.Color // statusline file block
	StatusBg  lipgloss.Color // statusline fill
	Line      lipgloss.Color // separators, indent guides
	Grey      lipgloss.Color
	GreyFg    lipgloss.Color
	GreyFg2   lipgloss.Color
	LightGrey lipgloss.Color
	Fg        lipgloss.Color

	// Accents.
	Red        lipgloss.Color
	Pink       lipgloss.Color
	Green      lipgloss.Color
	Blue       lipgloss.Color
	NordBlue   lipgloss.Color
	Yellow     lipgloss.Color
	Purple     lipgloss.Color
	DarkPurple lipgloss.Color
	Teal       lipgloss.Color
	Orange     lipgloss.Color
	Cyan       lipgloss.Color
	Folder     lipgloss.Color

	// Semantic aliases kept for the Markdown compiler and older components.
	NormalBg    lipgloss.Color
	InsertBg    lipgloss.Color
	CommandBg   lipgloss.Color
	SidebarBg   lipgloss.Color
	DarkFg      lipgloss.Color
	MutedFg     lipgloss.Color
	TitleFg     lipgloss.Color
	SelectedFg  lipgloss.Color
	SelectedBg  lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color
}

// DefaultName is the theme used when none is configured.
const DefaultName = "onedark"

// palette is one NvChad base46 "base_30" colour table.
type palette struct {
	Light      bool   `json:"light"`
	Fg         string `json:"fg"`
	DarkerBg   string `json:"darker_bg"`
	Bg         string `json:"bg"`
	Bg2        string `json:"bg2"`
	OneBg      string `json:"one_bg"`
	OneBg2     string `json:"one_bg2"`
	OneBg3     string `json:"one_bg3"`
	Grey       string `json:"grey"`
	GreyFg     string `json:"grey_fg"`
	GreyFg2    string `json:"grey_fg2"`
	LightGrey  string `json:"light_grey"`
	Red        string `json:"red"`
	Pink       string `json:"pink"`
	Line       string `json:"line"`
	Green      string `json:"green"`
	NordBlue   string `json:"nord_blue"`
	Blue       string `json:"blue"`
	Yellow     string `json:"yellow"`
	Purple     string `json:"purple"`
	DarkPurple string `json:"dark_purple"`
	Teal       string `json:"teal"`
	Orange     string `json:"orange"`
	Cyan       string `json:"cyan"`
	StatusBg   string `json:"status_bg"`
	LightBg    string `json:"light_bg"`
	Folder     string `json:"folder"`
}

var (
	userThemesOnce sync.Once
	userThemes     map[string]palette
	userThemesPath = defaultUserThemesPath
)

func defaultUserThemesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "misoku", "themes")
}

// SetUserThemesPath points the theme registry at a user-supplied location:
// either a directory of *.json palettes or a single .json palette file.
// A leading "~" expands to the user's home directory, since config.json
// values never pass through a shell.
// Pass "" to restore the default OS config location. It clears the cached
// palettes, so call it before the first theme.Get/Names lookup (main does
// this right after loading the config file).
func SetUserThemesPath(path string) {
	if path == "" {
		path = defaultUserThemesPath()
	}
	path = expandHome(path)
	userThemesPath = func() string { return path }
	userThemesOnce = sync.Once{}
	userThemes = nil
}

// expandHome replaces a leading "~" with the user's home directory. Paths
// such as "~user/x" are left unchanged.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// UserThemesDir returns the path to user-defined themes in the OS user config directory
// ($XDG_CONFIG_HOME/misoku/themes on Linux, ~/Library/Application Support/misoku/themes
// on macOS, and %APPDATA%\misoku\themes on Windows), or the custom path set
// via SetUserThemesPath.
// Bundled themes take precedence over user themes on name collision.
func UserThemesDir() string {
	return userThemesPath()
}

func loadUserThemes() map[string]palette {
	userThemesOnce.Do(func() {
		userThemes = make(map[string]palette)
		path := userThemesPath()
		if path == "" {
			return
		}
		info, err := os.Stat(path)
		switch {
		case err != nil:
			return
		case !info.IsDir():
			if strings.HasSuffix(strings.ToLower(path), ".json") {
				addUserThemeFile(path)
			}
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			addUserThemeFile(filepath.Join(path, e.Name()))
		}
	})
	return userThemes
}

// UserThemesStamp fingerprints the user theme files (names, sizes and
// modification times), so a change on disk can be noticed cheaply.
func UserThemesStamp() string {
	path := userThemesPath()
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	stamp := func(fi os.FileInfo) string {
		return fmt.Sprintf("%s|%d|%d;", fi.Name(), fi.Size(), fi.ModTime().UnixNano())
	}
	if !info.IsDir() {
		return stamp(info)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, e := range entries {
		if fi, err := e.Info(); err == nil && !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			b.WriteString(stamp(fi))
		}
	}
	return b.String()
}

// ReloadUserThemes drops the cached user palettes; the next lookup reads
// them from disk again.
func ReloadUserThemes() {
	userThemesOnce = sync.Once{}
	userThemes = nil
}

// addUserThemeFile parses one palette JSON file into the userThemes map.
// Invalid files are skipped; bundled theme names win collisions.
func addUserThemeFile(path string) {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".json")
	if _, exists := palettes[name]; exists {
		return // bundled themes win on name collision
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var p palette
	if err := json.Unmarshal(data, &p); err == nil && p.Fg != "" && p.Bg != "" {
		userThemes[name] = p
	}
}

// Names lists every available theme (bundled and user-defined), sorted.
func Names() []string {
	namesMap := make(map[string]struct{}, len(palettes))
	for n := range palettes {
		namesMap[n] = struct{}{}
	}
	for n := range loadUserThemes() {
		namesMap[n] = struct{}{}
	}
	names := make([]string, 0, len(namesMap))
	for n := range namesMap {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// DefaultTheme returns the NvChad onedark theme.
func DefaultTheme() Theme {
	t, _ := Get(DefaultName)
	return t
}

// Get returns the named theme; ok is false for unknown names.
func Get(name string) (Theme, bool) {
	norm := strings.ToLower(strings.TrimSpace(name))
	p, ok := palettes[norm]
	if !ok {
		u := loadUserThemes()
		p, ok = u[norm]
		if !ok {
			return Theme{}, false
		}
	}
	c := func(hex string) lipgloss.Color { return lipgloss.Color(hex) }
	t := Theme{
		Name:  name,
		Light: p.Light,

		DarkerBg:  c(p.DarkerBg),
		Bg:        c(p.Bg),
		Bg2:       c(p.Bg2),
		OneBg:     c(p.OneBg),
		OneBg2:    c(p.OneBg2),
		OneBg3:    c(p.OneBg3),
		LightBg:   c(p.LightBg),
		StatusBg:  c(p.StatusBg),
		Line:      c(p.Line),
		Grey:      c(p.Grey),
		GreyFg:    c(p.GreyFg),
		GreyFg2:   c(p.GreyFg2),
		LightGrey: c(p.LightGrey),
		Fg:        c(p.Fg),

		Red:        c(p.Red),
		Pink:       c(p.Pink),
		Green:      c(p.Green),
		Blue:       c(p.Blue),
		NordBlue:   c(p.NordBlue),
		Yellow:     c(p.Yellow),
		Purple:     c(p.Purple),
		DarkPurple: c(p.DarkPurple),
		Teal:       c(p.Teal),
		Orange:     c(p.Orange),
		Cyan:       c(p.Cyan),
		Folder:     c(p.Folder),
	}

	t.NormalBg = t.Blue
	t.InsertBg = t.DarkPurple
	t.CommandBg = t.Green
	t.SidebarBg = t.Purple
	t.DarkFg = t.Bg
	t.MutedFg = t.GreyFg2
	t.TitleFg = t.Fg
	t.SelectedFg = t.Blue
	t.SelectedBg = t.OneBg
	t.Border = t.Line
	t.BorderFocus = t.Blue
	return t, true
}

// Greyed returns t with every text colour set to Grey and the backgrounds
// kept: how a pane without the focus is drawn.
func (t Theme) Greyed() Theme {
	for _, c := range []*lipgloss.Color{
		&t.Fg, &t.GreyFg, &t.GreyFg2, &t.LightGrey, &t.Red, &t.Pink,
		&t.Green, &t.Blue, &t.NordBlue, &t.Yellow, &t.Purple, &t.DarkPurple,
		&t.Teal, &t.Orange, &t.Cyan, &t.Folder, &t.TitleFg, &t.MutedFg, &t.SelectedFg,
	} {
		*c = t.Grey
	}
	return t
}
