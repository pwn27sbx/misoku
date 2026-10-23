package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	uikit "github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// The settings menu (":settings") changes four settings of the "ui"
// section of config.json. Save model: a value picked with Enter (or a
// digit, or a click) applies at once and is saved at once; Esc only goes
// back (from a list or the width prompt to the menu, from the menu out)
// and never saves. A value equal to the one in use saves nothing. The
// save runs in a tea.Cmd, writes only the key that changed (config.SetUI)
// and reports a failure in the status bar.

// settingsState holds the settings in use and orders their saves. It is
// shared by every copy of the Model.
type settingsState struct {
	ui        config.UI // the "ui" section in use
	hideWidth int       // the explorer's hide width in use

	// seq numbers the saves in the order they were asked for (Update
	// only). Saves run one at a time (mu); written is, per key, the save
	// that wrote it last, so an older save that runs late does not undo a
	// newer one.
	seq     uint64
	mu      sync.Mutex
	written map[string]uint64
}

func (m *Model) prefs() *settingsState {
	if m.settings == nil {
		m.settings = &settingsState{hideWidth: config.DefaultExplorerHideWidth, written: map[string]uint64{}}
	}
	return m.settings
}

// Rows of the settings menu.
const (
	settingAccordion = iota
	settingHideWidth
	settingLook
	settingBorder
)

// lookPreset is a choice of heading icons and bullets. Misoku's own look
// has none: its keys are removed from the config.
type lookPreset struct {
	name    string
	icons   []string
	bullets []string
}

var lookPresets = []lookPreset{
	{name: "Misoku"},
	{"Numbers", []string{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "}, []string{"•", "◦", "▪", "▫"}},
	{"Arrows", []string{"▶ ", "▷ ", "▸ ", "▹ ", "› ", "· "}, []string{"▸", "▹", "›", "·"}},
	{"Plain", []string{"# ", "## ", "### ", "#### ", "##### ", "###### "}, []string{"-", "*", "+"}},
}

// borders are the names config ui.border accepts, with a sample.
var borders = []struct{ name, sample string }{{"rounded", "╭─╮"}, {"single", "┌─┐"}}

// maxHideWidth bounds the explorer hide width the menu accepts.
const maxHideWidth = 1000

// OpenSettings opens the settings menu. Other screens (the home screen,
// the command palette) open it with this or with a core.SettingsMsg.
func (m *Model) OpenSettings() tea.Cmd {
	m.openSettingsAt(settingAccordion)
	return nil
}

// openSettingsAt opens the settings menu with row highlighted.
func (m *Model) openSettingsAt(row int) {
	s := m.prefs()
	row4 := func(label, value string) string { return fmt.Sprintf("%-20s %s", label, value) }
	m.menu("Settings", []string{
		row4("Accordion folders", onOff(s.ui.AccordionOn())),
		row4("Explorer hide width", strconv.Itoa(s.hideWidth)),
		row4("Icons & bullets", m.lookName()),
		row4("Dialog border", m.borderName()),
	}, func(m *Model, i int) tea.Cmd { return m.chooseSetting(i) })
	m.menuBox.sel = row
	m.leaderPending = false
}

// back returns to the settings menu on row, for a list or prompt closed
// with Esc.
func back(row int) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		m.openSettingsAt(row)
		return nil
	}
}

func (m *Model) chooseSetting(row int) tea.Cmd {
	s := m.prefs()
	switch row {
	case settingAccordion:
		on := !s.ui.AccordionOn()
		s.ui.Accordion = &on
		m.sidebar.SetAccordion(on)
		m.openSettingsAt(row)
		m.setStatus("Accordion folders: " + onOff(on))
		return m.saveUI(map[string]any{config.UIAccordion: on})

	case settingHideWidth:
		cmd := m.prompt("Explorer hide width", strconv.Itoa(s.hideWidth),
			fmt.Sprintf("The explorer hides below this many columns (default %d).", config.DefaultExplorerHideWidth),
			func(m *Model, value string) tea.Cmd {
				defer m.openSettingsAt(settingHideWidth)
				n, err := strconv.Atoi(value)
				if err != nil || n < 1 || n > maxHideWidth {
					m.setError(fmt.Sprintf("Explorer hide width must be a whole number from 1 to %d", maxHideWidth))
					return nil
				}
				if n == m.prefs().hideWidth {
					return nil
				}
				m.prefs().ui.ExplorerHideWidth = n
				m.applyExplorerHideWidth(n)
				m.setStatus(fmt.Sprintf("Explorer hide width: %d", n))
				return m.saveUI(map[string]any{config.UIExplorerHideWidth: n})
			})
		m.promptBox.onCancel = back(row)
		return cmd

	case settingLook:
		items := make([]string, len(lookPresets))
		for i, p := range lookPresets {
			items[i] = fmt.Sprintf("%-8s %s", p.name, lookSample(p))
		}
		m.menu("Icons & bullets", items, func(m *Model, i int) tea.Cmd {
			p := lookPresets[i]
			defer m.openSettingsAt(settingLook)
			if m.lookName() == p.name {
				return nil
			}
			ui := &m.prefs().ui
			ui.HeadingIcons, ui.Bullets = p.icons, p.bullets
			m.applyLook()
			m.setStatus("Icons & bullets: " + p.name)
			if p.icons == nil {
				return m.saveUI(map[string]any{config.UIHeadingIcons: nil, config.UIBullets: nil})
			}
			return m.saveUI(map[string]any{config.UIHeadingIcons: p.icons, config.UIBullets: p.bullets})
		})
		m.menuBox.onCancel = back(row)
		if i := slices.IndexFunc(lookPresets, func(p lookPreset) bool { return p.name == m.lookName() }); i >= 0 {
			m.menuBox.sel = i
		}
		return nil

	case settingBorder:
		items := make([]string, len(borders))
		for i, b := range borders {
			items[i] = fmt.Sprintf("%-8s %s", b.name, b.sample)
		}
		m.menu("Dialog border", items, func(m *Model, i int) tea.Cmd {
			name := borders[i].name
			defer m.openSettingsAt(settingBorder)
			if name == m.borderName() {
				return nil
			}
			m.prefs().ui.Border = name
			uikit.SetBorder(name)
			m.applyLook()
			m.setStatus("Dialog border: " + name)
			return m.saveUI(map[string]any{config.UIBorder: name})
		})
		m.menuBox.onCancel = back(row)
		if m.borderName() == "single" {
			m.menuBox.sel = 1
		}
		return nil
	}
	return nil
}

// applyExplorerHideWidth sets the width below which the explorer hides.
// For now it only keeps the value; the explorer's auto-hide reads it once
// it exists.
func (m *Model) applyExplorerHideWidth(n int) {
	m.prefs().hideWidth = n
}

// lookName names the heading icons and bullets in use: a preset's name,
// or "custom" for lists of the user's own.
func (m *Model) lookName() string {
	ui := m.prefs().ui
	for _, p := range lookPresets {
		if slices.Equal(ui.HeadingIcons, p.icons) && slices.Equal(ui.Bullets, p.bullets) {
			return p.name
		}
	}
	return "custom"
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// borderName names the dialog border in use.
func (m *Model) borderName() string {
	if b := m.prefs().ui.Border; b == "single" {
		return b
	}
	return "rounded" // unset, or a name ui.SetBorder refused
}

// lookSample shows a preset: its first three heading icons and bullets.
func lookSample(p lookPreset) string {
	icons, bullets := p.icons, p.bullets
	if icons == nil {
		d := preview.DefaultLook()
		icons, bullets = d.HeadingIcons[:], d.Bullets
	}
	var b strings.Builder
	for _, s := range icons[:3] {
		b.WriteString(strings.TrimSpace(s) + " ")
	}
	b.WriteString(" ")
	b.WriteString(strings.Join(bullets[:3], " "))
	return b.String()
}

// saveUI saves values in the "ui" section of the config file, in the
// background (config.SetUI: only these keys change, nothing is written
// when they already hold these values). A failure is shown in the status
// bar. Without a config file nothing is saved.
func (m *Model) saveUI(values map[string]any) tea.Cmd {
	if m.configPath == "" {
		return nil
	}
	s := m.prefs()
	s.seq++
	seq, path := s.seq, m.configPath
	return func() tea.Msg {
		s.mu.Lock()
		defer s.mu.Unlock()
		fresh := map[string]any{}
		for key, value := range values {
			if s.written[key] < seq {
				fresh[key] = value
			}
		}
		if len(fresh) == 0 {
			return nil // newer values of these keys were saved already
		}
		if _, err := config.SetUI(path, fresh); err != nil {
			return core.StatusMsg{Text: "Could not save settings: " + err.Error(), Error: true}
		}
		for key := range fresh {
			s.written[key] = seq
		}
		return nil
	}
}
