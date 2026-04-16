package tui

import (
	"context"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	tea "charm.land/bubbletea/v2"
)

type envsLoadedMsg struct {
	entries []service.EnvView
	err     error
}

type envsView struct {
	app     *App
	list    *simpleList
	envs    []service.EnvView
	project string
	loading bool
}

func newEnvsView(a *App, project string) *envsView {
	return &envsView{
		app:     a,
		project: project,
		loading: true,
		list:    newSimpleList("Envs — "+project, nil, "↑/↓ • enter open secrets • r reload • q back", "no envs"),
	}
}

func (v *envsView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *envsView) Init() tea.Cmd {
	v.loading = true
	deps := v.app.deps
	auth := v.app.auth
	project := v.project
	return func() tea.Msg {
		envs, err := deps.Projects.Envs().List(context.Background(), auth, project)
		return envsLoadedMsg{entries: envs, err: err}
	}
}

func (v *envsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case envsLoadedMsg:
		v.loading = false
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.envs = m.entries
		items := make([]string, len(m.entries))
		for i, e := range m.entries {
			items[i] = e.Slug
		}
		v.list.SetItems(items)
		return v, nil
	case tea.KeyPressMsg:
		switch m.String() {
		case keys.Refr:
			return v, v.Init()
		case keys.Enter:
			if i, _, ok := v.list.Selected(); ok && i < len(v.envs) {
				env := v.envs[i].Slug
				return v, func() tea.Msg {
					return msgs.NavMsg{To: msgs.ViewSecrets, Project: v.project, Env: env}
				}
			}
		}
	}
	return v, v.list.Update(msg)
}

func (v *envsView) View() string {
	if v.loading {
		return "loading envs…"
	}
	return v.list.View()
}
