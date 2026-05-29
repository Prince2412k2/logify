package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/theme"
)

// BuildState bundles the bits needed to render the Build tab body.
type BuildState struct {
	ResourceUUID string
	Type         string
	Loading      bool
	Err          string
	Log          api.BuildLog
}

func buildContent(th theme.Theme, lay Layout, st BuildState, contentRows int, wrap bool) [][]Segment {
	out := [][]Segment{logsBlankRow(th, lay)}
	center := func(text string, fg string, bold bool) []Segment {
		pad := (lay.LogsW - runeLen(text)) / 2
		if pad < 0 {
			pad = 0
		}
		return fitRow([]Segment{
			{Text: repeat(" ", pad), FG: th.Text},
			{Text: text, FG: fg, Bold: bold},
		}, lay.LogsW, th.Text)
	}

	switch {
	case st.ResourceUUID == "":
		out = append(out, center("No service selected.", th.Muted, false))
	case st.Type == "service":
		out = append(out, center("Services don't have build logs.", th.Muted, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, center("Build output applies to applications only.", th.Dim, false))
	case st.Err != "":
		out = append(out, center("Could not load build log", th.Error, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, center(truncate(st.Err, lay.LogsW-4), th.Muted, false))
	case st.Loading && len(st.Log.Lines) == 0:
		out = append(out, center("Loading build log…", th.Muted, false))
	case len(st.Log.Lines) == 0:
		out = append(out, center("No build recorded for this application yet.", th.Muted, false))
	default:
		// Summary line at top: status + commit + when + duration
		summary := buildSummaryRow(th, lay, st.Log)
		out = append(out, summary)
		out = append(out, fitRow([]Segment{
			{Text: " ", FG: th.Text},
			{Text: strings.Repeat(BH, lay.LogsW-2), FG: th.Dim},
		}, lay.LogsW, th.Text))

		// Render the tail of the log to fit the remaining rows.
		slots := contentRows - len(out)
		if slots < 1 {
			for len(out) < contentRows {
				out = append(out, logsBlankRow(th, lay))
			}
			return out[:contentRows]
		}
		lines := st.Log.Lines
		rendered := renderBuildLines(th, lay, lines, wrap, slots)
		out = append(out, rendered...)
	}

	for len(out) < contentRows {
		out = append(out, logsBlankRow(th, lay))
	}
	return out[:contentRows]
}

func buildSummaryRow(th theme.Theme, lay Layout, log api.BuildLog) []Segment {
	glyph, colorFn := deployStatusGlyph(log.Status)
	statusColor := colorFn(th)
	when := humanizeAgo(log.CreatedAt)
	dur := humanizeDuration(log.FinishedAt - log.CreatedAt)
	msg := log.CommitMessage
	commit := log.Commit
	if commit == "" {
		commit = "—"
	}
	segs := []Segment{
		{Text: " ", FG: th.Text},
		{Text: glyph + " " + strings.ToLower(log.Status), FG: statusColor, Bold: true},
		{Text: "   ", FG: th.Text},
		{Text: commit, FG: th.Accent},
		{Text: "   ", FG: th.Text},
		{Text: when, FG: th.Text},
		{Text: "   ", FG: th.Text},
		{Text: dur, FG: th.Muted},
	}
	if msg != "" {
		used := segWidth(segs)
		room := lay.LogsW - used - 4
		if room > 8 {
			segs = append(segs, Segment{Text: "   ", FG: th.Text})
			segs = append(segs, Segment{Text: truncate(msg, room), FG: th.Dim})
		}
	}
	return fitRow(segs, lay.LogsW, th.Text)
}

// renderBuildLines turns the build-log slice into wrapped/truncated rows,
// tail-anchored to slots count.
func renderBuildLines(th theme.Theme, lay Layout, lines []string, wrap bool, slots int) [][]Segment {
	if slots <= 0 {
		return nil
	}
	innerW := lay.LogsW - 2
	if innerW < 10 {
		innerW = 10
	}

	expanded := func(line string) [][]Segment {
		// Defensive: even if upstream forgets to split, never let an embedded
		// control char rip the cursor out of the pane.
		safe := sanitizeRow(line)
		if !wrap {
			return [][]Segment{
				fitRow([]Segment{
					{Text: " " + truncate(safe, innerW) + " ", FG: th.Text},
				}, lay.LogsW, th.Text),
			}
		}
		runes := []rune(safe)
		if len(runes) == 0 {
			return [][]Segment{logsBlankRow(th, lay)}
		}
		var out [][]Segment
		for i := 0; i < len(runes); i += innerW {
			end := i + innerW
			if end > len(runes) {
				end = len(runes)
			}
			out = append(out, fitRow([]Segment{
				{Text: " " + string(runes[i:end]) + " ", FG: th.Text},
			}, lay.LogsW, th.Text))
		}
		return out
	}

	// Walk lines from the end backwards collecting rows until we fill `slots`.
	var rendered [][][]Segment
	used := 0
	for i := len(lines) - 1; i >= 0; i-- {
		rows := expanded(lines[i])
		rendered = append([][][]Segment{rows}, rendered...)
		used += len(rows)
		if used >= slots {
			over := used - slots
			if over > 0 && len(rendered[0]) > over {
				rendered[0] = rendered[0][over:]
				used = slots
			}
			break
		}
	}
	var out [][]Segment
	for _, r := range rendered {
		out = append(out, r...)
	}
	// Pad top if we under-filled.
	for len(out) < slots {
		out = append([][]Segment{logsBlankRow(th, lay)}, out...)
	}
	return out[:slots]
}

// Silence "unused fmt" if no fmt usage sneaks in elsewhere.
var _ = fmt.Sprintf
var _ = time.Second
