package tui

import (
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	tea "charm.land/bubbletea/v2"
)

type menuEntry struct {
	label  string
	target msgs.ViewID
	admin  bool
}

type menuView struct {
	app     *App
	list    *simpleList
	entries []menuEntry
}

func newMenu(a *App) *menuView {
	all := []menuEntry{
		{"Projects", msgs.ViewProjects, false},
		{"Vault", msgs.ViewVault, false},
		{"Team", msgs.ViewUsers, true},
		{"Audit", msgs.ViewAudit, true},
	}
	m := &menuView{app: a}
	for _, e := range all {
		if e.admin && !a.isAdmin() {
			continue
		}
		m.entries = append(m.entries, e)
	}
	items := make([]string, len(m.entries))
	for i, e := range m.entries {
		items[i] = e.label
	}
	m.list = newSimpleList("Main menu", items, "↑/↓ move • enter select • q quit", "nothing available")
	return m
}

func (m *menuView) Init() tea.Cmd { return nil }
func (m *menuView) Resize(w, h int) { m.list.Resize(w, h) }

func (m *menuView) Update(msg tea.Msg) (View, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == keys.Enter {
		if i, _, ok := m.list.Selected(); ok {
			entry := m.entries[i]
			return m, func() tea.Msg { return msgs.NavMsg{To: entry.target} }
		}
	}
	return m, m.list.Update(msg)
}

func (m *menuView) View() string { return m.list.View() }
