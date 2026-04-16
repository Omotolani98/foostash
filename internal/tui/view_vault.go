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

type vaultLoadedMsg struct {
	entries []service.VaultView
	err     error
}

type vaultActionMsg struct {
	action string
	err    error
}

type vaultView struct {
	app     *App
	list    *simpleList
	items   []service.VaultView
	loading bool
}

func newVaultView(a *App) *vaultView {
	help := "↑/↓ • r reload • q back"
	if a.isAdmin() {
		help = "↑/↓ • b rollback(v-1) • d delete • r reload • q back"
	}
	return &vaultView{
		app:     a,
		loading: true,
		list:    newSimpleList("Vault (org-wide)", nil, help, "vault is empty"),
	}
}

func (v *vaultView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *vaultView) Init() tea.Cmd {
	v.loading = true
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		entries, err := deps.Vault.List(context.Background(), auth)
		return vaultLoadedMsg{entries: entries, err: err}
	}
}

func (v *vaultView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case vaultLoadedMsg:
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
	case vaultActionMsg:
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
			if !v.app.isAdmin() {
				return v, func() tea.Msg {
					return msgs.StatusMsg{Text: "admin only"}
				}
			}
			if i, _, ok := v.list.Selected(); ok && i < len(v.items) {
				return v, v.delete(v.items[i].Key)
			}
		case keys.Rollback:
			if !v.app.isAdmin() {
				return v, func() tea.Msg {
					return msgs.StatusMsg{Text: "admin only"}
				}
			}
			if i, _, ok := v.list.Selected(); ok && i < len(v.items) {
				s := v.items[i]
				if s.Version > 1 {
					return v, v.rollback(s.Key, s.Version-1)
				}
				return v, func() tea.Msg {
					return msgs.StatusMsg{Text: "no earlier version to roll back to"}
				}
			}
		}
	}
	return v, v.list.Update(msg)
}

func (v *vaultView) delete(key string) tea.Cmd {
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		err := deps.Vault.Delete(context.Background(), auth, key)
		return vaultActionMsg{action: "delete " + key, err: err}
	}
}

func (v *vaultView) rollback(key string, target int) tea.Cmd {
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		_, err := deps.Vault.Rollback(context.Background(), auth, key, target)
		return vaultActionMsg{action: fmt.Sprintf("rollback %s → v%d", key, target), err: err}
	}
}

func (v *vaultView) View() string {
	if v.loading {
		return "loading vault…"
	}
	return v.list.View() + "\n" + styles.WarnText.Render("values hidden — edit via 'foostash vault set' locally")
}
