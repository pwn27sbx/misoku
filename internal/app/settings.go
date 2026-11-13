package app

import (
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"
	uikit "github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The settings menu (":settings") changes eight settings of the "ui"
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
	settingTransparent
	settingHeadingNumbers
	settingLinkSuggestions
	settingLanguage
)

// languageChoices are the values ui.language takes, in the order the
// Language list shows them: Auto (the key removed: the system decides),
// then each language.
var languageChoices = []string{"", string(i18n.EN), string(i18n.ES)}

// systemLanguage is the language Auto (system) means: the one read at
// startup, so choosing Auto runs no program inside Update. Tests replace
// it.
var systemLanguage = i18n.System

// lookPreset is a choice of heading icons and bullets. Misoku's own look
// has none: its keys are removed from the config. A preset is told apart
// by its place in lookPresets, never by its shown name.
type lookPreset struct {
	name    i18n.Key // "" for Misoku's own, a name in no language
	icons   []string
	bullets []string
}

var lookPresets = []lookPreset{
	{},
	{i18n.LookNumbers, []string{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "}, []string{"•", "◦", "▪", "▫"}},
	{i18n.LookArrows, []string{"▶ ", "▷ ", "▸ ", "▹ ", "› ", "· "}, []string{"▸", "▹", "›", "·"}},
	{i18n.LookPlain, []string{"# ", "## ", "### ", "#### ", "##### ", "###### "}, []string{"-", "*", "+"}},
}

// title is the preset's name as the menu shows it.
func (p lookPreset) title() string {
	if p.name == "" {
		return "Misoku"
	}
	return i18n.T(p.name)
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

// settingLabels are the rows' labels, in row order.
var settingLabels = []i18n.Key{
	i18n.SettingAccordion, i18n.SettingHideWidth, i18n.SettingLook, i18n.SettingBorder,
	i18n.SettingTransparent, i18n.SettingHeadingNumbers, i18n.SettingLinkSuggestions, i18n.SettingLanguage,
}

// openSettingsAt opens the settings menu with highlight highlighted, in
// the language in use.
func (m *Model) openSettingsAt(highlight int) {
	s := m.prefs()
	values := []string{
		onOff(s.ui.AccordionOn()),
		strconv.Itoa(s.hideWidth),
		m.lookName(),
		m.borderName(),
		onOff(s.ui.Transparent),
		onOff(s.ui.HeadingNumbersOn()),
		m.linkSuggestionsName(),
		languageName(s.ui.Language),
	}
	labels := make([]string, len(settingLabels))
	for i, key := range settingLabels {
		labels[i] = i18n.T(key)
	}
	m.menu(i18n.T(i18n.SettingsTitle), columns(labels, values), func(m *Model, i int) tea.Cmd { return m.chooseSetting(i) })
	m.menuBox.sel = highlight
	m.leaderPending = false
	m.tutorDefer("settings_open", "")
}

// columns lays out each label followed by its value, the values lined up
// one column past the widest label.
func columns(labels, values []string) []string {
	width := 0
	for _, l := range labels {
		width = max(width, ansi.StringWidth(l))
	}
	rows := make([]string, len(labels))
	for i, l := range labels {
		rows[i] = l + strings.Repeat(" ", width-ansi.StringWidth(l)+1) + values[i]
	}
	return rows
}

// changed says in the bar that row now holds value.
func (m *Model) changed(row int, value string) {
	m.setStatus(i18n.T(i18n.SettingChanged, i18n.T(settingLabels[row]), value))
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
		m.changed(row, onOff(on))
		return m.saveUI(map[string]any{config.UIAccordion: on})

	case settingHeadingNumbers:
		on := !s.ui.HeadingNumbersOn()
		s.ui.HeadingNumbers = &on
		m.applyLook()
		m.openSettingsAt(row)
		m.changed(row, onOff(on))
		return m.saveUI(map[string]any{config.UIHeadingNumbers: on})

	case settingLinkSuggestions:
		// Inline is the default: choosing it removes the key.
		if s.ui.LinkSuggestions == config.LinkSuggestionsFinder {
			s.ui.LinkSuggestions = config.LinkSuggestionsInline
			m.openSettingsAt(row)
			m.changed(row, m.linkSuggestionsName())
			return m.saveUI(map[string]any{config.UILinkSuggestions: nil})
		}
		s.ui.LinkSuggestions = config.LinkSuggestionsFinder
		m.openSettingsAt(row)
		m.changed(row, m.linkSuggestionsName())
		return m.saveUI(map[string]any{config.UILinkSuggestions: config.LinkSuggestionsFinder})

	case settingTransparent:
		on := !s.ui.Transparent
		s.ui.Transparent = on
		m.applyTheme(m.baseTheme, false)
		m.applyLook() // heading bands follow the option
		m.openSettingsAt(row)
		m.changed(row, onOff(on))
		if !on {
			// Off is the default: remove the key rather than write false.
			return m.saveUI(map[string]any{config.UITransparent: nil})
		}
		return m.saveUI(map[string]any{config.UITransparent: true})

	case settingHideWidth:
		cmd := m.prompt(i18n.T(i18n.SettingHideWidth), strconv.Itoa(s.hideWidth),
			i18n.T(i18n.SettingHideWidthHelp, config.DefaultExplorerHideWidth),
			func(m *Model, value string) tea.Cmd {
				defer m.openSettingsAt(settingHideWidth)
				n, err := strconv.Atoi(value)
				if err != nil || n < 1 || n > maxHideWidth {
					m.setError(i18n.T(i18n.SettingHideWidthBad, maxHideWidth))
					return nil
				}
				if n == m.prefs().hideWidth {
					return nil
				}
				m.prefs().ui.ExplorerHideWidth = n
				m.applyExplorerHideWidth(n)
				m.changed(settingHideWidth, strconv.Itoa(n))
				return m.saveUI(map[string]any{config.UIExplorerHideWidth: n})
			})
		m.promptBox.onCancel = back(row)
		return cmd

	case settingLook:
		names := make([]string, len(lookPresets))
		samples := make([]string, len(lookPresets))
		for i, p := range lookPresets {
			names[i], samples[i] = p.title(), lookSample(p)
		}
		m.menu(i18n.T(i18n.SettingLook), columns(names, samples), func(m *Model, i int) tea.Cmd {
			p := lookPresets[i]
			defer m.openSettingsAt(settingLook)
			if m.lookIndex() == i {
				return nil
			}
			ui := &m.prefs().ui
			ui.HeadingIcons, ui.Bullets = p.icons, p.bullets
			m.applyLook()
			m.changed(settingLook, p.title())
			if p.icons == nil {
				return m.saveUI(map[string]any{config.UIHeadingIcons: nil, config.UIBullets: nil})
			}
			return m.saveUI(map[string]any{config.UIHeadingIcons: p.icons, config.UIBullets: p.bullets})
		})
		m.menuBox.onCancel = back(row)
		if i := m.lookIndex(); i >= 0 {
			m.menuBox.sel = i
		}
		return nil

	case settingBorder:
		names := make([]string, len(borders))
		samples := make([]string, len(borders))
		for i, b := range borders {
			names[i], samples[i] = b.name, b.sample
		}
		m.menu(i18n.T(i18n.SettingBorder), columns(names, samples), func(m *Model, i int) tea.Cmd {
			name := borders[i].name
			defer m.openSettingsAt(settingBorder)
			if name == m.borderName() {
				return nil
			}
			m.prefs().ui.Border = name
			uikit.SetBorder(name)
			m.applyLook()
			m.changed(settingBorder, name)
			return m.saveUI(map[string]any{config.UIBorder: name})
		})
		m.menuBox.onCancel = back(row)
		if m.borderName() == "single" {
			m.menuBox.sel = 1
		}
		return nil

	case settingLanguage:
		items := make([]string, len(languageChoices))
		for i, c := range languageChoices {
			items[i] = languageName(c)
		}
		m.menu(i18n.T(i18n.SettingLanguage), items, func(m *Model, i int) tea.Cmd {
			saved := languageChoices[i]
			// Redrawn after the switch, so in the language just chosen.
			defer m.openSettingsAt(settingLanguage)
			if saved == m.prefs().ui.Language {
				return nil
			}
			m.prefs().ui.Language = saved
			l, ok := i18n.Parse(saved)
			if !ok {
				l = systemLanguage()
			}
			i18n.Set(l)
			m.changed(settingLanguage, languageName(saved))
			if saved == "" {
				// Auto is the default: remove the key rather than write it.
				return m.saveUI(map[string]any{config.UILanguage: nil})
			}
			return m.saveUI(map[string]any{config.UILanguage: saved})
		})
		m.menuBox.onCancel = back(row)
		if i := slices.Index(languageChoices, m.prefs().ui.Language); i >= 0 {
			m.menuBox.sel = i
		}
		return nil
	}
	return nil
}

// applyExplorerHideWidth sets the width below which the explorer hides,
// keeps it for the settings menu, and judges the current width again.
func (m *Model) applyExplorerHideWidth(n int) {
	m.prefs().hideWidth = n
	WithExplorerHideWidth(n)(m)
}

// lookIndex is the place in lookPresets of the heading icons and bullets
// in use, or -1 for lists of the user's own.
func (m *Model) lookIndex() int {
	ui := m.prefs().ui
	return slices.IndexFunc(lookPresets, func(p lookPreset) bool {
		return slices.Equal(ui.HeadingIcons, p.icons) && slices.Equal(ui.Bullets, p.bullets)
	})
}

// lookName names the heading icons and bullets in use: a preset's name,
// or "custom" for lists of the user's own.
func (m *Model) lookName() string {
	if i := m.lookIndex(); i >= 0 {
		return lookPresets[i].title()
	}
	return i18n.T(i18n.LookCustom)
}

func onOff(on bool) string {
	if on {
		return i18n.T(i18n.SettingOn)
	}
	return i18n.T(i18n.SettingOff)
}

// borderName names the dialog border in use.
func (m *Model) borderName() string {
	if b := m.prefs().ui.Border; b == "single" {
		return b
	}
	return "rounded" // unset, or a name ui.SetBorder refused
}

// languageName names a saved ui.language as the settings menu shows it: a
// language by its own name, anything else as Auto (system).
func languageName(saved string) string {
	if l, ok := i18n.Parse(saved); ok {
		return i18n.Name(l)
	}
	return i18n.T(i18n.LanguageAuto)
}

// linkSuggestionsName names what "[[" opens, as the settings menu shows it.
func (m *Model) linkSuggestionsName() string {
	if m.prefs().ui.LinkSuggestions == config.LinkSuggestionsFinder {
		return i18n.T(i18n.LinkSuggestionsFinder)
	}
	return i18n.T(i18n.LinkSuggestionsInline)
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
			return core.StatusMsg{Text: i18n.T(i18n.SettingSaveFailed, err.Error()), Error: true}
		}
		for key := range fresh {
			s.written[key] = seq
		}
		return settingsSavedMsg{}
	}
}
