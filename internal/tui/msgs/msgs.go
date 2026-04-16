// Package msgs defines cross-view tea.Msg types for the TUI.
package msgs

// ViewID identifies one of the top-level TUI views.
type ViewID int

const (
	ViewMenu ViewID = iota
	ViewProjects
	ViewEnvs
	ViewSecrets
	ViewVault
	ViewUsers
	ViewAudit
	ViewHistory
)

// NavMsg asks the root model to push the given view onto the nav stack.
type NavMsg struct {
	To ViewID
	// Context carried along for drill-downs (e.g. project slug for envs view).
	Project string
	Env     string
	Key     string
}

// PopMsg asks the root model to pop the current view.
type PopMsg struct{}

// ErrMsg carries a service-layer error up to the root for display.
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// StatusMsg is a one-shot status line message.
type StatusMsg struct{ Text string }
