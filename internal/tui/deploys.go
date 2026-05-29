package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/theme"
)

// DeploysState bundles the bits needed to render the Deploys tab.
type DeploysState struct {
	ResourceUUID string
	Type         string // "application" | "service" | ""
	Loading      bool
	Err          string
	Deployments  []api.Deployment
}

func deployStatusGlyph(status string) (string, func(t theme.Theme) string) {
	switch strings.ToLower(status) {
	case "finished":
		return "✓", func(t theme.Theme) string { return t.OK }
	case "in_progress", "running":
		return "⠿", func(t theme.Theme) string { return t.Accent }
	case "queued":
		return "◷", func(t theme.Theme) string { return t.Warn }
	case "failed":
		return "✕", func(t theme.Theme) string { return t.Error }
	case "cancelled-by-user", "cancelled":
		return "⊘", func(t theme.Theme) string { return t.Dim }
	}
	return "•", func(t theme.Theme) string { return t.Muted }
}

func humanizeAgo(epoch int64) string {
	if epoch <= 0 {
		return "—"
	}
	d := time.Since(time.Unix(epoch, 0))
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return time.Unix(epoch, 0).Format("Jan 2")
	}
}

func humanizeDuration(seconds int64) string {
	if seconds <= 0 {
		return "—"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	s := seconds % 60
	if m < 60 {
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%dh %02dm", h, m)
}

// Column widths for the Deploys table inside LogsW.
// Layout: STATUS  COMMIT  WHEN  DURATION  TRIGGER  MESSAGE
type deployCols struct {
	statusW   int
	commitW   int
	whenW     int
	durationW int
	triggerW  int
	messageW  int // takes remaining space
}

func computeDeployCols(logsW int) deployCols {
	// Fixed widths for the first columns. Message claims whatever's left,
	// columns drop in priority order if we run out of room.
	const (
		statusW   = 13
		commitW   = 8
		whenW     = 12
		durationW = 9
		triggerW  = 11
	)
	c := deployCols{statusW: statusW, commitW: commitW, whenW: whenW,
		durationW: durationW, triggerW: triggerW}
	gaps := 6 // leading space + 5 inter-column gaps
	fixed := c.statusW + c.commitW + c.whenW + c.durationW + c.triggerW + gaps

	if fixed+12 <= logsW {
		c.messageW = logsW - fixed
		return c
	}
	// Too narrow for message — drop trigger first.
	c.triggerW = 0
	gaps = 5
	fixed = c.statusW + c.commitW + c.whenW + c.durationW + gaps
	if fixed+8 <= logsW {
		c.messageW = logsW - fixed
		return c
	}
	// Drop duration.
	c.durationW = 0
	gaps = 4
	fixed = c.statusW + c.commitW + c.whenW + gaps
	if fixed <= logsW {
		c.messageW = max(0, logsW-fixed)
		return c
	}
	// Worst case: just status + commit.
	c.whenW = 0
	c.messageW = 0
	return c
}

func deploysHeader(th theme.Theme, lay Layout) []Segment {
	c := computeDeployCols(lay.LogsW)
	segs := []Segment{{Text: " ", FG: th.Text}}
	add := func(label string, w int) {
		if w == 0 {
			return
		}
		segs = append(segs, Segment{Text: rpad(label, w), FG: th.Muted, Bold: true})
		segs = append(segs, Segment{Text: " ", FG: th.Text})
	}
	add("STATUS", c.statusW)
	add("COMMIT", c.commitW)
	if c.whenW > 0 {
		add("STARTED", c.whenW)
	}
	if c.durationW > 0 {
		add("DURATION", c.durationW)
	}
	if c.triggerW > 0 {
		add("TRIGGER", c.triggerW)
	}
	if c.messageW > 0 {
		add("MESSAGE", c.messageW)
	}
	return fitRow(segs, lay.LogsW, th.Text)
}

func deploysSeparator(th theme.Theme, lay Layout) []Segment {
	return fitRow([]Segment{
		{Text: " ", FG: th.Text},
		{Text: strings.Repeat(BH, lay.LogsW-2), FG: th.Dim},
	}, lay.LogsW, th.Text)
}

func triggerColor(th theme.Theme, trigger string) string {
	switch trigger {
	case "webhook":
		return th.Info
	case "api":
		return th.BadgeProj
	case "rollback":
		return th.Warn
	case "restart":
		return th.Muted
	}
	return th.Muted
}

func deployRow(th theme.Theme, lay Layout, d api.Deployment, selected bool) []Segment {
	c := computeDeployCols(lay.LogsW)
	glyph, colorFn := deployStatusGlyph(d.Status)
	statusColor := colorFn(th)
	statusText := fmt.Sprintf("%s %s", glyph, strings.ToLower(d.Status))
	statusText = truncate(statusText, c.statusW)

	bg := ""
	fg := th.Text
	if selected {
		bg = th.Border
	}
	segs := []Segment{{Text: " ", FG: fg, BG: bg}}
	segs = append(segs, Segment{Text: rpad(statusText, c.statusW), FG: statusColor, Bold: true, BG: bg})
	segs = append(segs, Segment{Text: " ", FG: fg, BG: bg})
	commit := d.Commit
	if commit == "" {
		commit = "—"
	}
	segs = append(segs, Segment{Text: rpad(commit, c.commitW), FG: th.Accent, BG: bg})
	if c.whenW > 0 {
		segs = append(segs, Segment{Text: " ", FG: fg, BG: bg})
		segs = append(segs, Segment{Text: rpad(humanizeAgo(d.CreatedAt), c.whenW), FG: fg, BG: bg})
	}
	if c.durationW > 0 {
		segs = append(segs, Segment{Text: " ", FG: fg, BG: bg})
		segs = append(segs, Segment{Text: rpad(humanizeDuration(d.DurationSeconds), c.durationW), FG: th.Muted, BG: bg})
	}
	if c.triggerW > 0 {
		trig := d.Trigger
		if trig == "" {
			trig = "—"
		}
		segs = append(segs, Segment{Text: " ", FG: fg, BG: bg})
		segs = append(segs, Segment{Text: rpad(truncate(trig, c.triggerW), c.triggerW), FG: triggerColor(th, trig), BG: bg})
	}
	if c.messageW > 0 {
		msg := d.CommitMessage
		if msg == "" {
			msg = d.UUID // fall back to deployment id so the row isn't empty
		}
		segs = append(segs, Segment{Text: " ", FG: fg, BG: bg})
		segs = append(segs, Segment{Text: rpad(truncate(msg, c.messageW), c.messageW), FG: th.Dim, BG: bg})
	}
	return fitRow(segs, lay.LogsW, th.Text)
}

// deploysContent renders the full Deploys tab body for the right pane.
func deploysContent(th theme.Theme, lay Layout, st DeploysState, contentRows int) [][]Segment {
	out := [][]Segment{logsBlankRow(th, lay)}

	centerMsg := func(text string, fg string, bold bool) []Segment {
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
		out = append(out, centerMsg("No service selected.", th.Muted, false))
	case st.Type == "service":
		out = append(out, centerMsg("Services don't track deployments.", th.Muted, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, centerMsg("Use the Logs tab to watch runtime activity instead.", th.Dim, false))
	case st.Err != "":
		out = append(out, centerMsg("Could not load deployments", th.Error, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, centerMsg(truncate(st.Err, lay.LogsW-4), th.Muted, false))
	case st.Loading && len(st.Deployments) == 0:
		out = append(out, centerMsg("Loading deployments…", th.Muted, false))
	case len(st.Deployments) == 0:
		out = append(out, centerMsg("No deployments recorded yet.", th.Muted, false))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, centerMsg("Trigger a deploy in Coolify; refresh with r.", th.Dim, false))
	default:
		out = append(out, deploysHeader(th, lay))
		out = append(out, deploysSeparator(th, lay))
		max := contentRows - len(out) - 1
		for i, d := range st.Deployments {
			if i >= max {
				break
			}
			out = append(out, deployRow(th, lay, d, false))
		}
	}

	for len(out) < contentRows {
		out = append(out, logsBlankRow(th, lay))
	}
	return out[:contentRows]
}
