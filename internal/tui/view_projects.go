package tui

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	tea "charm.land/bubbletea/v2"
)

type projectsLoadedMsg struct {
	entries []service.ProjectListEntry
	err     error
}

type projectsView struct {
	app      *App
	list     *simpleList
	projects []service.ProjectListEntry
	loading  bool
}

func newProjectsView(a *App) *projectsView {
	v := &projectsView{
		app:     a,
		loading: true,
		list:    newSimpleList("Projects", nil, "↑/↓ • enter open envs • r reload • q back", "no projects"),
	}
	return v
}

func (v *projectsView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *projectsView) Init() tea.Cmd {
	v.loading = true
	return v.load()
}

func (v *projectsView) load() tea.Cmd {
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		entries, err := deps.Projects.List(context.Background(), auth)
		return projectsLoadedMsg{entries: entries, err: err}
	}
}

func (v *projectsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case projectsLoadedMsg:
		v.loading = false
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.projects = m.entries
		items := make([]string, len(m.entries))
		for i, p := range m.entries {
			items[i] = fmt.Sprintf("%-24s  %d env(s)", p.Slug, p.EnvCount)
		}
		v.list.SetItems(items)
		return v, nil
	case tea.KeyPressMsg:
		switch m.String() {
		case keys.Refr:
			return v, v.Init()
		case keys.Enter:
			if i, _, ok := v.list.Selected(); ok && i < len(v.projects) {
				slug := v.projects[i].Slug
				return v, func() tea.Msg { return msgs.NavMsg{To: msgs.ViewEnvs, Project: slug} }
			}
		}
	}
	return v, v.list.Update(msg)
}

func (v *projectsView) View() string {
	if v.loading {
		return "loading projects…"
	}
	return v.list.View()
}
