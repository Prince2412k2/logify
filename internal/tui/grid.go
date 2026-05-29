package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Box drawing chars (rounded — matches reference).
const (
	BTL = "╭"
	BTR = "╮"
	BBL = "╰"
	BBR = "╯"
	BH  = "─"
	BV  = "│"
	BHD = "┬"
	BHU = "┴"
)

// Layout constants. NAV_W is the inner width of the nav pane. TotalW/TotalH
// are picked from window size; LogsW is derived.
const (
	NavW       = 32
	OuterPad   = 2 // outer border + gutter on each side
	MinTotalW  = 100
	MinTotalH  = 24
	DefTotalW  = 144
	DefTotalH  = 38
)

// Layout caches a sized layout (recomputed on resize).
type Layout struct {
	TotalW int
	TotalH int
	LogsW  int
}

func NewLayout(w, h int) Layout {
	if w < MinTotalW {
		w = MinTotalW
	}
	if h < MinTotalH {
		h = MinTotalH
	}
	// Logs interior = total - (outer border*2 + outer-pad*2 + inner box borders*2 + nav inner + separator)
	// outer-pad already includes the outer border itself per our row builders, so:
	// total = 1(outer-l) + 1(gutter) + 1(nav-l) + navW + 1(sep) + logsW + 1(logs-r) + 1(gutter) + 1(outer-r)
	logsW := w - 2 /*outer*/ - 2 /*gutter*/ - 2 /*inner borders*/ - 1 /*separator*/ - NavW
	if logsW < 30 {
		logsW = 30
	}
	return Layout{TotalW: w, TotalH: h, LogsW: logsW}
}

// Segment is one styled run in a row. Width math uses len(rune(Text)).
type Segment struct {
	Text   string
	FG     string
	BG     string
	Bold   bool
	Strike bool
}

type Row []Segment

// runeLen returns the printable rune count.
func runeLen(s string) int { return utf8.RuneCountInString(s) }

// repeat is a clamped strings.Repeat — never panics on negative counts.
func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

func rpad(s string, n int) string {
	l := runeLen(s)
	if l >= n {
		return runesSlice(s, 0, n)
	}
	return s + repeat(" ", n-l)
}

func lpad(s string, n int) string {
	l := runeLen(s)
	if l >= n {
		return runesSlice(s, 0, n)
	}
	return repeat(" ", n-l) + s
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	l := runeLen(s)
	if l <= n {
		return s
	}
	if n == 1 {
		return runesSlice(s, 0, 1)
	}
	return runesSlice(s, 0, n-1) + "…"
}

func runesSlice(s string, a, b int) string {
	r := []rune(s)
	if a < 0 {
		a = 0
	}
	if b > len(r) {
		b = len(r)
	}
	return string(r[a:b])
}

// sanitizeRow scrubs characters that would break a single-row layout:
// CR, LF, TAB, and other ASCII C0 controls. Replaces TAB with 4 spaces;
// LF/CR collapse to a space (since the upstream should have split them
// already, this is defense-in-depth).
func sanitizeRow(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\t':
			b.WriteString("    ")
		case '\r', '\n':
			b.WriteByte(' ')
		default:
			if r < 0x20 || r == 0x7f {
				// other C0 / DEL — silently drop
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func segWidth(segs []Segment) int {
	w := 0
	for _, s := range segs {
		w += runeLen(s.Text)
	}
	return w
}

// fitRow pads or truncates segments to exactly width cells. Padding uses fillFG.
func fitRow(segs []Segment, width int, fillFG string) []Segment {
	used := segWidth(segs)
	if used < width {
		segs = append(segs, Segment{Text: repeat(" ", width-used), FG: fillFG})
		return segs
	}
	if used > width {
		over := used - width
		last := segs[len(segs)-1]
		lastLen := runeLen(last.Text)
		if lastLen >= over {
			last.Text = runesSlice(last.Text, 0, lastLen-over)
			segs[len(segs)-1] = last
		} else {
			// peel back segments until we've shed `over` cells
			for over > 0 && len(segs) > 0 {
				last := segs[len(segs)-1]
				lastLen := runeLen(last.Text)
				if lastLen <= over {
					over -= lastLen
					segs = segs[:len(segs)-1]
				} else {
					last.Text = runesSlice(last.Text, 0, lastLen-over)
					segs[len(segs)-1] = last
					over = 0
				}
			}
		}
	}
	return segs
}

// RenderRow renders a Row to an ANSI string with the given default background.
func RenderRow(r Row, defaultBG string) string {
	var b strings.Builder
	for _, s := range r {
		st := lipgloss.NewStyle()
		if s.FG != "" {
			st = st.Foreground(lipgloss.Color(s.FG))
		}
		bg := s.BG
		if bg == "" {
			bg = defaultBG
		}
		if bg != "" {
			st = st.Background(lipgloss.Color(bg))
		}
		if s.Bold {
			st = st.Bold(true)
		}
		if s.Strike {
			st = st.Strikethrough(true)
		}
		b.WriteString(st.Render(s.Text))
	}
	return b.String()
}

// JoinRows assembles a string from a list of Rows + default bg.
func JoinRows(rows []Row, defaultBG string) string {
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(RenderRow(r, defaultBG))
	}
	return b.String()
}
