package tui

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	tea "charm.land/bubbletea/v2"
)

type auditLoadedMsg struct {
	entries []service.AuditEntryView
	err     error
}

type auditView struct {
	app     *App
	list    *simpleList
	items   []service.AuditEntryView
	loading bool
}

func newAuditView(a *App) *auditView {
	return &auditView{
		app:     a,
		loading: true,
		list:    newSimpleList("Audit", nil, "↑/↓ • r reload • q back", "no entries"),
	}
}

func (v *auditView) Resize(w, h int) { v.list.Resize(w, h) }

func (v *auditView) Init() tea.Cmd {
	if !v.app.isAdmin() {
		return func() tea.Msg { return msgs.ErrMsg{Err: service.ErrForbidden} }
	}
	v.loading = true
	deps := v.app.deps
	auth := v.app.auth
	return func() tea.Msg {
		entries, err := deps.Audit.Query(context.Background(), auth, service.AuditQuery{Limit: 100})
		return auditLoadedMsg{entries: entries, err: err}
	}
}

func (v *auditView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch m := msg.(type) {
	case auditLoadedMsg:
		v.loading = false
		if m.err != nil {
			return v, func() tea.Msg { return msgs.ErrMsg{Err: m.err} }
		}
		v.items = m.entries
		items := make([]string, len(m.entries))
		for i, e := range m.entries {
			resource := ""
			if e.ResourceType != nil {
				resource = *e.ResourceType
			}
			items[i] = fmt.Sprintf("%s  %-20s  %-12s  status=%d",
				e.CreatedAt.Format("2006-01-02 15:04:05"),
				e.Action,
				resource,
				e.Status,
			)
		}
		v.list.SetItems(items)
		return v, nil
	case tea.KeyPressMsg:
		if m.String() == keys.Refr {
			return v, v.Init()
		}
	}
	return v, v.list.Update(msg)
}

func (v *auditView) View() string {
	if v.loading {
		return "loading audit…"
	}
	return v.list.View()
}

var _ = msgs.PopMsg{}
