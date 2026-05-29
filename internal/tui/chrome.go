package tui

import (
	"fmt"

	"github.com/princepatel/logify/internal/theme"
)

// topBorderRow builds the rounded top frame with brand + breadcrumb on the left
// and connection-status pill on the right.
func topBorderRow(th theme.Theme, lay Layout, breadcrumb, conn string, tail int, isAdmin bool) Row {
	left := []Segment{
		{Text: BTL + BH + " ", FG: th.Border},
		{Text: "logify", FG: th.Accent, Bold: true},
	}
	leftLen := runeLen(BTL+BH+" ") + runeLen("logify")
	if breadcrumb != "" {
		left = append(left,
			Segment{Text: " ", FG: th.Border},
			Segment{Text: "▸", FG: th.Dim},
			Segment{Text: " ", FG: th.Border},
			Segment{Text: breadcrumb, FG: th.Text},
		)
		leftLen += 3 + runeLen(breadcrumb)
	}
	if isAdmin {
		chip := " ◆ admin "
		left = append(left,
			Segment{Text: " ", FG: th.Border},
			Segment{Text: chip, FG: th.AccentFg, BG: th.Accent, Bold: true},
		)
		leftLen += 1 + runeLen(chip)
	}
	left = append(left, Segment{Text: " ", FG: th.Border})
	leftLen++

	var dot, label, extra string
	switch conn {
	case "connecting":
		dot, label = th.Warn, "connecting…"
	case "disconnected":
		dot, label = th.Error, "disconnected"
	default:
		dot, label = th.OK, "connected"
		if tail > 0 {
			extra = fmt.Sprintf(" · tail %d", tail)
		}
	}
	rightStr := fmt.Sprintf(" ● %s%s ", label, extra)
	rightLen := runeLen(rightStr) + 2 // + ─ + ╮

	right := []Segment{
		{Text: " ", FG: th.Border},
		{Text: "●", FG: dot},
		{Text: " " + label, FG: th.Muted},
		{Text: extra, FG: th.Dim},
		{Text: " ", FG: th.Border},
		{Text: BH + BTR, FG: th.Border},
	}

	dashes := lay.TotalW - leftLen - rightLen
	if dashes < 0 {
		dashes = 0
	}
	mid := []Segment{{Text: repeat(BH, dashes), FG: th.Border}}
	out := append([]Segment{}, left...)
	out = append(out, mid...)
	out = append(out, right...)
	return out
}

func bottomBorderRow(th theme.Theme, lay Layout) Row {
	return Row{{Text: BBL + repeat(BH, lay.TotalW-2) + BBR, FG: th.Border}}
}

// edgeRow wraps an interior of exactly TotalW-2 cells in left/right border chars.
func edgeRow(th theme.Theme, lay Layout, interior []Segment) Row {
	interior = fitRow(interior, lay.TotalW-2, th.Text)
	out := append([]Segment{{Text: BV, FG: th.Border}}, interior...)
	out = append(out, Segment{Text: BV, FG: th.Border})
	return out
}

func spacerRow(th theme.Theme, lay Layout) Row {
	return edgeRow(th, lay, []Segment{{Text: repeat(" ", lay.TotalW-2)}})
}

func innerBoxTopRow(th theme.Theme, lay Layout, focused string) Row {
	colNav := th.Border
	if focused == "nav" {
		colNav = th.BorderFoc
	}
	colLog := th.Border
	if focused == "logs" {
		colLog = th.BorderFoc
	}
	interior := []Segment{
		{Text: " ", FG: th.Border},
		{Text: BTL, FG: colNav},
		{Text: repeat(BH, NavW), FG: colNav},
		{Text: BHD, FG: th.Border},
		{Text: repeat(BH, lay.LogsW), FG: colLog},
		{Text: BTR, FG: colLog},
		{Text: " ", FG: th.Border},
	}
	return edgeRow(th, lay, interior)
}

func innerBoxBottomRow(th theme.Theme, lay Layout, focused string) Row {
	colNav := th.Border
	if focused == "nav" {
		colNav = th.BorderFoc
	}
	colLog := th.Border
	if focused == "logs" {
		colLog = th.BorderFoc
	}
	interior := []Segment{
		{Text: " ", FG: th.Border},
		{Text: BBL, FG: colNav},
		{Text: repeat(BH, NavW), FG: colNav},
		{Text: BHU, FG: th.Border},
		{Text: repeat(BH, lay.LogsW), FG: colLog},
		{Text: BBR, FG: colLog},
		{Text: " ", FG: th.Border},
	}
	return edgeRow(th, lay, interior)
}

// innerSplitRow joins one nav row and one logs row through the vertical seam.
func innerSplitRow(th theme.Theme, lay Layout, focused string, nav, logs []Segment) Row {
	colNav := th.Border
	if focused == "nav" {
		colNav = th.BorderFoc
	}
	colLog := th.Border
	if focused == "logs" {
		colLog = th.BorderFoc
	}
	nav = fitRow(nav, NavW, th.Text)
	logs = fitRow(logs, lay.LogsW, th.Text)
	interior := []Segment{{Text: " ", FG: th.Border}, {Text: BV, FG: colNav}}
	interior = append(interior, nav...)
	interior = append(interior, Segment{Text: BV, FG: th.Border})
	interior = append(interior, logs...)
	interior = append(interior, Segment{Text: BV, FG: colLog})
	interior = append(interior, Segment{Text: " ", FG: th.Border})
	return edgeRow(th, lay, interior)
}

// helpStripInterior is the bottom keybinds row. Width = TotalW-2.
// When notice is non-empty, it replaces the keybind hints with a centred toast.
func helpStripInterior(th theme.Theme, lay Layout, notice string) []Segment {
	if notice != "" {
		used := runeLen(notice) + 4
		pad := (lay.TotalW - 2 - used) / 2
		if pad < 0 {
			pad = 0
		}
		return fitRow([]Segment{
			{Text: repeat(" ", pad), FG: th.Text},
			{Text: " ✓ ", FG: th.AccentFg, BG: th.OK, Bold: true},
			{Text: " " + notice + " ", FG: th.Text, Bold: true},
		}, lay.TotalW-2, th.Text)
	}
	segs := []Segment{{Text: " ", FG: th.Text}}
	add := func(key, label string) {
		segs = append(segs, Segment{Text: key, FG: th.Accent, Bold: true})
		segs = append(segs, Segment{Text: " " + label, FG: th.Muted})
		segs = append(segs, Segment{Text: "  ·  ", FG: th.Dim})
	}
	last := func(key, label string) {
		segs = append(segs, Segment{Text: key, FG: th.Accent, Bold: true})
		segs = append(segs, Segment{Text: " " + label, FG: th.Muted})
	}
	switch lay.Mode {
	case ModeMobile:
		add("o", "open")
		add("/", "find")
		add("1-5", "tab")
		last("q", "quit")
	case ModeCompact:
		add("o", "open")
		add("←→", "tabs")
		add("/", "search")
		add("␣", "pause")
		add("y", "yank")
		add("?", "help")
		last("q", "quit")
	default:
		add("o", "open")
		add("←→", "tabs")
		add("/", "search")
		add("␣", "pause")
		add("y", "yank")
		add("w", "wrap")
		add("z", "zoom")
		add("f", "filter")
		add("t", "theme")
		add("?", "help")
		last("q", "quit")
	}
	return fitRow(segs, lay.TotalW-2, th.Text)
}
