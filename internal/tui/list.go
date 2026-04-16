package tui

import (
	"strings"

	"github.com/Omotolani98/foostash/internal/tui/keys"
	"github.com/Omotolani98/foostash/internal/tui/styles"
	tea "charm.land/bubbletea/v2"
)

// simpleList is a minimal keyboard-driven list widget shared by TUI views.
// Not meant to replace bubbles/list; just enough for the MVP views.
type simpleList struct {
	title   string
	items   []string
	index   int
	width   int
	height  int
	help    string
	empty   string
}

func newSimpleList(title string, items []string, help, empty string) *simpleList {
	return &simpleList{title: title, items: items, help: help, empty: empty}
}

func (l *simpleList) SetItems(items []string) {
	l.items = items
	if l.index >= len(items) {
		l.index = 0
	}
}

func (l *simpleList) Selected() (int, string, bool) {
	if len(l.items) == 0 {
		return 0, "", false
	}
	return l.index, l.items[l.index], true
}

func (l *simpleList) Resize(w, h int) { l.width, l.height = w, h }

func (l *simpleList) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case keys.Up, keys.K:
			if l.index > 0 {
				l.index--
			}
		case keys.Down, keys.J:
			if l.index < len(l.items)-1 {
				l.index++
			}
		}
	}
	return nil
}

func (l *simpleList) View() string {
	var b strings.Builder
	if l.title != "" {
		b.WriteString(styles.Header.Render(l.title))
		b.WriteString("\n")
	}
	if len(l.items) == 0 {
		b.WriteString(styles.MutedText.Render(l.empty))
		b.WriteString("\n")
	}
	for i, it := range l.items {
		if i == l.index {
			b.WriteString(styles.ItemSelected.Render("▸ " + it))
		} else {
			b.WriteString(styles.Item.Render("  " + it))
		}
		b.WriteString("\n")
	}
	if l.help != "" {
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render(l.help))
	}
	return b.String()
}
