package tui

import (
	"strings"

	"github.com/princepatel/logify/internal/theme"
)

// buildConfirmOverlay paints a small centred "are you sure?" modal.
func buildConfirmOverlay(th theme.Theme, lay Layout, base []Row, c confirmState) []Row {
	W := 60
	if W > lay.TotalW-8 {
		W = lay.TotalW - 8
	}
	if W < 30 {
		W = 30
	}

	border := func() []Segment {
		return []Segment{{Text: BV + strings.Repeat(" ", W-2) + BV, FG: th.Warn}}
	}
	pad := func(inner []Segment, leftCol int) []Segment {
		used := segWidth(inner)
		right := W - 4 - leftCol - used
		if right < 0 {
			right = 0
		}
		out := []Segment{{Text: BV + strings.Repeat(" ", leftCol+1), FG: th.Warn}}
		out = append(out, inner...)
		out = append(out, Segment{Text: strings.Repeat(" ", right) + " " + BV, FG: th.Warn})
		return out
	}

	title := " ⚠  confirm destructive action "
	dashes := W - 2 - runeLen(title)
	if dashes < 2 {
		dashes = 2
	}
	lines := [][]Segment{
		{
			{Text: BTL + title, FG: th.Warn, Bold: true},
			{Text: strings.Repeat(BH, dashes-1) + BTR, FG: th.Warn},
		},
		border(),
		pad([]Segment{{Text: c.Prompt, FG: th.Text, Bold: true}}, 2),
		border(),
		pad([]Segment{
			{Text: " y ", FG: th.AccentFg, BG: th.OK, Bold: true},
			{Text: "  confirm     ", FG: th.Muted},
			{Text: " n ", FG: th.AccentFg, BG: th.Error, Bold: true},
			{Text: "  cancel  ", FG: th.Muted},
			{Text: "(esc also cancels)", FG: th.Dim},
		}, 2),
		border(),
		{
			{Text: BBL + strings.Repeat(BH, W-2) + BBR, FG: th.Warn},
		},
	}

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
