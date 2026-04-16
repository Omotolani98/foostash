// Package styles holds the lipgloss palette used by the TUI.
package styles

import "charm.land/lipgloss/v2"

var (
	Primary = lipgloss.Color("#7D56F4")
	Accent  = lipgloss.Color("#F25D94")
	Muted   = lipgloss.Color("#626262")
	Fg      = lipgloss.Color("#FAFAFA")
	Warn    = lipgloss.Color("#FFB454")
	Err     = lipgloss.Color("#E06C75")

	Title = lipgloss.NewStyle().
		Foreground(Fg).
		Background(Primary).
		Bold(true).
		Padding(0, 1)

	Header = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Padding(0, 0, 1, 0)

	Item = lipgloss.NewStyle().
		Foreground(Fg).
		Padding(0, 2)

	ItemSelected = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true).
			Padding(0, 2)

	MutedText = lipgloss.NewStyle().Foreground(Muted)
	WarnText  = lipgloss.NewStyle().Foreground(Warn)
	ErrText   = lipgloss.NewStyle().Foreground(Err)

	Help = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(1, 0, 0, 0)

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(0, 1)
)
