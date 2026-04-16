package sshsrv

import (
	"log/slog"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/ssh"
)

// TUIDeps is the subset of services the TUI needs. It mirrors server.Deps
// but is defined here to avoid an import cycle.
type TUIDeps struct {
	Projects *service.Projects
	Users    *service.Users
	Audit    *service.Audit
	Secrets  *service.Secrets
	Vault    *service.Vault
	Invites  *service.Invites
}

// teaHandler is the Wish bubbletea.Handler. For each incoming SSH session it
// pulls the AuthContext stashed by the auth callback and constructs a fresh
// tui.App rooted at the main menu.
func teaHandler(deps TUIDeps) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		actx := AuthFromContext(sess.Context())
		if actx == nil {
			slog.Warn("ssh session with no auth context", "user", sess.User(), "remote", sess.RemoteAddr().String())
			return nil, nil
		}
		slog.Info("ssh session start",
			"user", actx.Email,
			"role", actx.Role,
			"remote", sess.RemoteAddr().String(),
		)
		app := tui.NewApp(tui.Deps{
			Projects: deps.Projects,
			Users:    deps.Users,
			Audit:    deps.Audit,
			Secrets:  deps.Secrets,
			Vault:    deps.Vault,
			Invites:  deps.Invites,
		}, actx)
		return app, nil
	}
}
