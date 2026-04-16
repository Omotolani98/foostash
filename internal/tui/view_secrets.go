package tui

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	"github.com/Omotolani98/foostash/internal/tui/styles"
	tea "charm.land/bubbletea/v2"
)

type secretsLoadedMsg struct {
	entries []service.SecretView
	err     error
}

type secretActionMsg struct {
	action string
	err    error
}

type secretsView struct {
	app     *App
	list    *simpleList
	items   []service.SecretView
	project string
	env     string
	loading bool
}

func newSecretsView(a *App, project, env string) *secretsView {
	return &secretsView{
		app:     a,
		project: project,
		env:     env,
		loading: true,
		list: newSimpleList(
			fmt.Sprintf("Secrets — %s/%s", project, env),
			nil,
			"↑/↓ • b rollback(v-1) • d delete • r reload • q back",
			"no secrets",
		),
	}
}

func (v *secretsView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *secretsView) Init() tea.Cmd {
	v.loading = true
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		entries, err := deps.Secrets.List(context.Background(), auth, v.project, v.env)
		return secretsLoadedMsg{entries: entries, err: err}
	}
}

func (v *secretsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case secretsLoadedMsg:
		v.loading = false
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.items = m.entries
		items := make([]string, len(m.entries))
		for i, s := range m.entries {
			items[i] = fmt.Sprintf("%-28s  v%d  %s",
				s.Key, s.Version, s.UpdatedAt.Format("2006-01-02 15:04"))
		}
		v.list.SetItems(items)
		return v, nil
	case secretActionMsg:
		if m.err != nil {
			return v, tea.Batch(
				func() tea.Msg { return msgs.ErrMsg{Err: m.err} },
				v.Init(),
			)
		}
		return v, tea.Batch(
			func() tea.Msg { return msgs.StatusMsg{Text: m.action + " ok"} },
			v.Init(),
		)
	case tea.KeyPressMsg:
		switch m.String() {
		case keys.Refr:
			return v, v.Init()
		case keys.Delete:
			if i, _, ok := v.list.Selected(); ok && i < len(v.items) {
				key := v.items[i].Key
				return v, v.delete(key)
			}
		case keys.Rollback:
			if i, _, ok := v.list.Selected(); ok && i < len(v.items) {
				sec := v.items[i]
				if sec.Version > 1 {
					return v, v.rollback(sec.Key, sec.Version-1)
				}
				return v, func() tea.Msg {
					return msgs.StatusMsg{Text: "no earlier version to roll back to"}
				}
			}
		}
	}
	return v, v.list.Update(msg)
}

func (v *secretsView) delete(key string) tea.Cmd {
	deps := v.app.deps
	auth := v.app.auth
	project, env := v.project, v.env
	return func() tea.Msg {
		err := deps.Secrets.Delete(context.Background(), auth, project, env, key)
		return secretActionMsg{action: "delete " + key, err: err}
	}
}

func (v *secretsView) rollback(key string, target int) tea.Cmd {
	deps := v.app.deps
	auth := v.app.auth
	project, env := v.project, v.env
	return func() tea.Msg {
		_, err := deps.Secrets.Rollback(context.Background(), auth, project, env, key, target)
		return secretActionMsg{action: fmt.Sprintf("rollback %s → v%d", key, target), err: err}
	}
}

func (v *secretsView) View() string {
	if v.loading {
		return "loading secrets…"
	}
	return v.list.View() + "\n" + styles.WarnText.Render("values hidden — edit via 'foostash set' locally")
}
