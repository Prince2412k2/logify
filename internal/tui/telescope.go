package tui

import (
	"fmt"
	"strings"

	"github.com/princepatel/logify/internal/theme"
)

// TelescopeState holds the picker overlay state.
type TelescopeState struct {
	Open    bool
	Filter  string
	Cursor  int      // index into Filtered
	Filtered []int   // indexes into the source rows slice (matched + sorted)
}

// Filter recomputes the matched index list from the current Filter string.
// Sources is the input universe (selectable service rows; non-headers).
func (s *TelescopeState) Recompute(sources []ServiceRow) {
	q := strings.ToLower(strings.TrimSpace(s.Filter))
	s.Filtered = s.Filtered[:0]
	for i, r := range sources {
		if q == "" || matchesService(r, q) {
			s.Filtered = append(s.Filtered, i)
		}
	}
	if s.Cursor >= len(s.Filtered) {
		if len(s.Filtered) == 0 {
			s.Cursor = 0
		} else {
			s.Cursor = len(s.Filtered) - 1
		}
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
}

func matchesService(r ServiceRow, q string) bool {
	return strings.Contains(strings.ToLower(r.Service), q) ||
		strings.Contains(strings.ToLower(r.Path), q) ||
		strings.Contains(strings.ToLower(r.Project), q)
}

// SelectedRow returns the currently-highlighted source row, if any.
func (s *TelescopeState) SelectedRow(sources []ServiceRow) (ServiceRow, bool) {
	if s.Cursor < 0 || s.Cursor >= len(s.Filtered) {
		return ServiceRow{}, false
	}
	idx := s.Filtered[s.Cursor]
	if idx < 0 || idx >= len(sources) {
		return ServiceRow{}, false
	}
	return sources[idx], true
}

// buildTelescopeOverlay paints the modal onto an existing row set.
func buildTelescopeOverlay(th theme.Theme, lay Layout, base []Row, st TelescopeState, sources []ServiceRow) []Row {
	// Mobile: full-screen panel. Desktop: centred modal ~70% width.
	W := lay.TotalW - 8
	if lay.Mode != ModeMobile {
		W = lay.TotalW * 7 / 10
		if W < 50 {
			W = 50
		}
		if W > 90 {
			W = 90
		}
	}
	listMax := lay.TotalH - 12
	if listMax < 6 {
		listMax = 6
	}
	if lay.Mode == ModeMobile {
		listMax = lay.TotalH - 10
	}

	lines := [][]Segment{
		// Header
		{
			{Text: BTL + BH + " pick a service ", FG: th.Accent, Bold: true},
			{Text: strings.Repeat(BH, W-19) + BTR, FG: th.Accent},
		},
		{{Text: BV + strings.Repeat(" ", W-2) + BV, FG: th.Accent}},
		// Filter input row
		filterRow(th, st.Filter, W),
		// Separator under filter
		{
			{Text: BV + " ", FG: th.Accent},
			{Text: strings.Repeat(BH, W-4), FG: th.Dim},
			{Text: " " + BV, FG: th.Accent},
		},
	}

	// List rows
	shown := len(st.Filtered)
	if shown > listMax {
		shown = listMax
	}
	for i := 0; i < shown; i++ {
		srcIdx := st.Filtered[i]
		row := sources[srcIdx]
		lines = append(lines, telescopeRow(th, W, row, i == st.Cursor))
	}
	for i := shown; i < listMax; i++ {
		lines = append(lines, []Segment{{Text: BV + strings.Repeat(" ", W-2) + BV, FG: th.Accent}})
	}

	// Footer
	lines = append(lines,
		[]Segment{
			{Text: BV + " ", FG: th.Accent},
			{Text: strings.Repeat(BH, W-4), FG: th.Dim},
			{Text: " " + BV, FG: th.Accent},
		},
		footerRow(th, W, len(st.Filtered), len(sources)),
		[]Segment{{Text: BBL + strings.Repeat(BH, W-2) + BBR, FG: th.Accent}},
	)

	// Paint over base.
	left := (lay.TotalW - W) / 2
	top := (lay.TotalH - len(lines)) / 2
	if top < 0 {
		top = 0
	}
	out := make([]Row, len(base))
	copy(out, base)
	for i, ln := range lines {
		if top+i < 0 || top+i >= len(out) {
			continue
		}
		out[top+i] = paintOverlay(out[top+i], ln, left, W, th)
	}
	return out
}

func filterRow(th theme.Theme, value string, w int) []Segment {
	caret := "▸"
	innerW := w - 6
	display := value
	if value == "" {
		display = ""
	}
	cursorCh := "▌"
	if len(display) > innerW-1 {
		display = display[len(display)-(innerW-1):]
	}
	pad := innerW - len(display) - 1
	if pad < 0 {
		pad = 0
	}
	return []Segment{
		{Text: BV + " ", FG: th.Accent},
		{Text: caret + " ", FG: th.Accent, Bold: true},
		{Text: display, FG: th.Text},
		{Text: cursorCh, FG: th.Accent, Bold: true},
		{Text: strings.Repeat(" ", pad), FG: th.Text},
		{Text: " " + BV, FG: th.Accent},
	}
}

func telescopeRow(th theme.Theme, w int, row ServiceRow, selected bool) []Segment {
	dotColor := statusColor(th, row.Status)
	bg := ""
	fg := th.Text
	cursor := "  "
	if selected {
		bg = th.Accent
		fg = th.AccentFg
		dotColor = th.AccentFg
		cursor = "▸ "
	}
	// inner usable width = w - 4 (borders + padding)
	innerW := w - 4
	name := row.Service
	suffix := "  " + row.Project
	if row.Stage != "" && row.Stage != row.Project {
		suffix = "  " + row.Project + "/" + row.Stage
	}
	combined := name + suffix
	if runeLen(combined) > innerW-6 {
		// truncate suffix first
		over := runeLen(combined) - (innerW - 6)
		if runeLen(suffix) > over {
			suffix = runesSlice(suffix, 0, runeLen(suffix)-over-1) + "…"
		} else {
			suffix = ""
			rem := innerW - 6 - runeLen(name)
			if rem < 0 {
				name = runesSlice(name, 0, innerW-6-1) + "…"
				suffix = ""
			}
		}
	}
	right := innerW - 4 - runeLen(name) - runeLen(suffix)
	if right < 1 {
		right = 1
	}
	return []Segment{
		{Text: BV + " ", FG: th.Accent},
		{Text: cursor, FG: fg, BG: bg, Bold: selected},
		{Text: "●", FG: dotColor, BG: bg},
		{Text: " ", FG: fg, BG: bg},
		{Text: name, FG: fg, BG: bg, Bold: true},
		{Text: suffix, FG: th.Muted, BG: bg},
		{Text: strings.Repeat(" ", right), FG: fg, BG: bg},
		{Text: " " + BV, FG: th.Accent},
	}
}

func footerRow(th theme.Theme, w int, matched, total int) []Segment {
	left := fmt.Sprintf("  %d of %d services", matched, total)
	right := "↑↓ move · enter open · esc cancel "
	pad := w - 2 - runeLen(left) - runeLen(right)
	if pad < 1 {
		pad = 1
	}
	return []Segment{
		{Text: BV, FG: th.Accent},
		{Text: left, FG: th.Muted},
		{Text: strings.Repeat(" ", pad), FG: th.Text},
		{Text: right, FG: th.Dim},
		{Text: BV, FG: th.Accent},
	}
}
