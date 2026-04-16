// Package keys defines key bindings used by the TUI.
package keys

// Key bindings expressed as the string forms returned by bubbletea v2
// KeyPressMsg.String(). Matched via switch statements in view Update funcs.
const (
	Quit   = "q"
	Esc    = "esc"
	Ctrl_C = "ctrl+c"
	Up     = "up"
	Down   = "down"
	K      = "k"
	J      = "j"
	Enter  = "enter"
	Help   = "?"
	Refr   = "r"

	Delete   = "d"
	Rollback = "b"
	History  = "h"
	Invite   = "i"
)
