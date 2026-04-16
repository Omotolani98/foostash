// Package tui implements the Foostash SSH-served terminal UI.
//
// Architecture: a single root App model owns a nav stack of view models. Each
// view model implements a small interface (Init/Update/View/Resize). The root
// forwards tea.Msgs to the top-of-stack view and handles global keys (quit,
// back) itself. Service calls happen inside tea.Cmd closures so the UI never
// blocks on Postgres.
package tui

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/msgs"
	"github.com/Omotolani98/foostash/internal/tui/styles"
	tea "charm.land/bubbletea/v2"
)

// Deps is the bundle of services the TUI views call into. Mirrors the
// fields the root model needs; kept narrow to simplify testing.
type Deps struct {
	Projects *service.Projects
	Users    *service.Users
	Audit    *service.Audit
	Secrets  *service.Secrets
	Vault    *service.Vault
	Invites  *service.Invites
}

// View is the interface each TUI pane implements. Parallel to tea.Model but
// with a Resize hook so the root can forward window dimensions on Nav.
type View interface {
	Init() tea.Cmd
	Update(tea.Msg) (View, tea.Cmd)
	View() string
	Resize(w, h int)
}

// App is the root tea.Model.
type App struct {
	deps   Deps
	auth   *service.AuthContext
	stack  []View
	width  int
	height int
	status string
	errMsg string
}

// NewApp builds the root model rooted at the main menu.
func NewApp(deps Deps, auth *service.AuthContext) *App {
	a := &App{deps: deps, auth: auth}
	a.push(newMenu(a))
	return a
}

func (a *App) push(v View) {
	a.stack = append(a.stack, v)
	v.Resize(a.width, a.height)
}

func (a *App) pop() {
	if len(a.stack) <= 1 {
		return
	}
	a.stack = a.stack[:len(a.stack)-1]
	a.top().Resize(a.width, a.height)
}

func (a *App) top() View { return a.stack[len(a.stack)-1] }

// Init implements tea.Model.
func (a *App) Init() tea.Cmd { return a.top().Init() }

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		for _, v := range a.stack {
			v.Resize(a.width, a.height)
		}
	case tea.KeyPressMsg:
		switch m.String() {
		case keys.Ctrl_C:
			return a, tea.Quit
		case keys.Quit:
			// Top-level menu: quit. Otherwise: treat as back.
			if len(a.stack) == 1 {
				return a, tea.Quit
			}
			a.pop()
			return a, a.top().Init()
		case keys.Esc:
			if len(a.stack) > 1 {
				a.pop()
				return a, a.top().Init()
			}
		}
	case msgs.NavMsg:
		a.push(a.buildView(m))
		return a, a.top().Init()
	case msgs.PopMsg:
		a.pop()
		return a, a.top().Init()
	case msgs.ErrMsg:
		a.errMsg = m.Err.Error()
		a.status = ""
		return a, nil
	case msgs.StatusMsg:
		a.status = m.Text
		a.errMsg = ""
		return a, nil
	}

	top := a.top()
	updated, cmd := top.Update(msg)
	a.stack[len(a.stack)-1] = updated
	return a, cmd
}

// View implements tea.Model.
func (a *App) View() tea.View {
	header := styles.Title.Render("foostash") + "  " +
		styles.MutedText.Render(fmt.Sprintf("%s (%s)", a.auth.Email, a.auth.Role))
	body := a.top().View()
	footer := styles.Help.Render("q back/quit • esc back • ? help")
	if a.errMsg != "" {
		footer = styles.ErrText.Render(a.errMsg) + "\n" + footer
	} else if a.status != "" {
		footer = styles.MutedText.Render(a.status) + "\n" + footer
	}
	return tea.NewView(header + "\n\n" + body + "\n" + footer)
}

// buildView materializes the View for a NavMsg target. Unknown ids fall back
// to the menu so a misrouted Nav never wedges the UI.
func (a *App) buildView(nav msgs.NavMsg) View {
	switch nav.To {
	case msgs.ViewMenu:
		return newMenu(a)
	case msgs.ViewProjects:
		return newProjectsView(a)
	case msgs.ViewEnvs:
		return newEnvsView(a, nav.Project)
	case msgs.ViewSecrets:
		return newSecretsView(a, nav.Project, nav.Env)
	case msgs.ViewVault:
		return newVaultView(a)
	case msgs.ViewUsers:
		return newUsersView(a)
	case msgs.ViewAudit:
		return newAuditView(a)
	default:
		return newMenu(a)
	}
}

func (a *App) isAdmin() bool {
	return a.auth != nil && a.auth.Role == "admin"
}
