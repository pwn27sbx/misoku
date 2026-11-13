package app

import tea "github.com/charmbracelet/bubbletea"

// ReloadLearn is the command that reads the home screen's Learn entries
// again after a language change.
func ReloadLearn(tm tea.Model) tea.Cmd {
	m := tm.(Model)
	return m.reloadLearn()
}

// LearnLabels are the home screen's Learn entries as they stand.
func LearnLabels(tm tea.Model) []string {
	m := tm.(Model)
	var out []string
	for _, e := range m.learn.entries() {
		out = append(out, e.Label)
	}
	return out
}
