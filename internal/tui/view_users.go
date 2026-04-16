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

type usersLoadedMsg struct {
	entries []service.UserView
	err     error
}

type inviteCreatedMsg struct {
	token string
	email string
	err   error
}

type usersView struct {
	app        *App
	list       *simpleList
	items      []service.UserView
	loading    bool
	inviteInfo string
}

func newUsersView(a *App) *usersView {
	return &usersView{
		app:     a,
		loading: true,
		list:    newSimpleList("Team", nil, "↑/↓ • r reload • q back", "no users"),
	}
}

func (v *usersView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *usersView) Init() tea.Cmd {
	if !v.app.isAdmin() {
		return func() tea.Msg { return msgs.ErrMsg{Err: service.ErrForbidden} }
	}
	v.loading = true
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		entries, err := deps.Users.List(context.Background(), auth)
		return usersLoadedMsg{entries: entries, err: err}
	}
}

func (v *usersView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case usersLoadedMsg:
		v.loading = false
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.items = m.entries
		items := make([]string, len(m.entries))
		for i, u := range m.entries {
			status := u.Role
			if u.RevokedAt != nil {
				status += " (revoked)"
			}
			items[i] = fmt.Sprintf("%-32s  %s", u.Email, status)
		}
		v.list.SetItems(items)
		return v, nil
	case inviteCreatedMsg:
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.inviteInfo = fmt.Sprintf("invite for %s → %s", m.email, m.token)
		return v, nil
	}
	return v, v.list.Update(msg)
}

func (v *usersView) View() string {
	if v.loading {
		return "loading users…"
	}
	out := v.list.View()
	if v.inviteInfo != "" {
		out += "\n" + styles.WarnText.Render(v.inviteInfo)
	}
	_ = keys.Invite
	return out
}
