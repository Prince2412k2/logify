// Package picker is an inline, gum-style selector. Renders to stderr without
// alt-screen so the host terminal's background and scrollback are untouched;
// stdout stays clean for downstream JSON.
package picker

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/princepatel/logify/internal/theme"
)

// Item is one row of the picker.
type Item struct {
	Label string
	Value any
}

// Result reports the outcome.
type Result struct {
	Picked bool
	Item   Item
}

type model struct {
	items    []Item
	cursor   int
	filter   string
	filtered []int
	prompt   string
	th       theme.Theme
	pick     bool
}

func newModel(items []Item, prompt string, th theme.Theme) *model {
	m := &model{items: items, prompt: prompt, th: th}
	m.refilter()
	return m
}

func (m *model) refilter() {
	q := strings.ToLower(m.filter)
	m.filtered = m.filtered[:0]
	for i, it := range m.items {
		if q == "" || strings.Contains(strings.ToLower(it.Label), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) == 0 {
			m.cursor = 0
		} else {
			m.cursor = len(m.filtered) - 1
		}
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.pick = true
				return m, tea.Quit
			}
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.refilter()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
				m.refilter()
			}
		}
	}
	return m, nil
}

func (m *model) View() string {
	th := m.th
	// Foreground-only styles → host terminal background shines through.
	fg := func(c string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	prompt := fg(th.Muted)
	cursor := fg(th.Accent).Bold(true)
	chosen := fg(th.Accent).Bold(true)
	idle := fg(th.Text)
	dim := fg(th.Dim)

	var b strings.Builder
	if m.prompt != "" {
		b.WriteString(prompt.Render(m.prompt))
		if m.filter != "" {
			b.WriteString(" ")
			b.WriteString(chosen.Render(m.filter))
			b.WriteString(cursor.Render("▌"))
		} else {
			b.WriteString(" ")
			b.WriteString(dim.Render("(type to filter, ↑↓ enter, esc cancel)"))
		}
		b.WriteString("\n")
	}
	for vi, idx := range m.filtered {
		it := m.items[idx]
		if vi == m.cursor {
			b.WriteString(cursor.Render("▸ "))
			b.WriteString(chosen.Render(it.Label))
		} else {
			b.WriteString(dim.Render("  "))
			b.WriteString(idle.Render(it.Label))
		}
		b.WriteString("\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString(dim.Render("  (no matches)\n"))
	}
	return b.String()
}

// Pick runs the inline picker. UI goes to stderr; stdout is left untouched so
// callers can emit JSON results afterward.
func Pick(items []Item, prompt string, th theme.Theme) (Result, error) {
	if len(items) == 0 {
		return Result{}, fmt.Errorf("no items to pick from")
	}
	m := newModel(items, prompt, th)
	p := tea.NewProgram(m,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)
	if _, err := p.Run(); err != nil {
		return Result{}, err
	}
	if !m.pick {
		return Result{Picked: false}, nil
	}
	idx := m.filtered[m.cursor]
	return Result{Picked: true, Item: m.items[idx]}, nil
}
