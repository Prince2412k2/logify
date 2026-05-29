package tui

import (
	"fmt"
	"strings"

	"github.com/princepatel/logify/internal/theme"
)

type Tab struct {
	ID    string
	Label string
	Stub  bool
}

var Tabs = []Tab{
	{ID: "logs", Label: "Logs"},
	{ID: "build", Label: "Build", Stub: true},
	{ID: "config", Label: "Config", Stub: true},
	{ID: "env", Label: "Env", Stub: true},
	{ID: "deploys", Label: "Deploys", Stub: true},
}

func TabByID(id string) (int, Tab) {
	for i, t := range Tabs {
		if t.ID == id {
			return i, t
		}
	}
	return 0, Tabs[0]
}

func logsBlankRow(th theme.Theme, lay Layout) []Segment {
	return []Segment{{Text: repeat(" ", lay.LogsW), FG: th.Text}}
}

func logsTabsRow(th theme.Theme, lay Layout, activeTab string) []Segment {
	segs := []Segment{{Text: " ", FG: th.Text}}
	for i, t := range Tabs {
		if t.ID == activeTab {
			segs = append(segs, Segment{Text: "[ ", FG: th.Accent})
			segs = append(segs, Segment{Text: t.Label, FG: th.Accent, Bold: true})
			segs = append(segs, Segment{Text: " ]", FG: th.Accent})
		} else {
			segs = append(segs, Segment{Text: t.Label, FG: th.Muted})
			if t.Stub {
				segs = append(segs, Segment{Text: "·", FG: th.Dim})
			}
		}
		if i < len(Tabs)-1 {
			segs = append(segs, Segment{Text: "  ", FG: th.Text})
		}
	}
	return fitRow(segs, lay.LogsW, th.Text)
}

func logsTabsSeparator(th theme.Theme, lay Layout, activeTab string) []Segment {
	offset := 1
	type rng struct{ id string; start, length int }
	ranges := []rng{}
	for _, t := range Tabs {
		var l int
		if t.ID == activeTab {
			l = 2 + runeLen(t.Label) + 2 // "[ Logs ]"
		} else {
			l = runeLen(t.Label)
			if t.Stub {
				l++
			}
		}
		ranges = append(ranges, rng{t.ID, offset, l})
		offset += l + 2
	}
	var active rng
	for _, r := range ranges {
		if r.id == activeTab {
			active = r
		}
	}
	post := lay.LogsW - active.start - active.length
	if post < 0 {
		post = 0
	}
	return []Segment{
		{Text: repeat(BH, active.start), FG: th.Dim},
		{Text: repeat(BH, active.length), FG: th.Accent, Bold: true},
		{Text: repeat(BH, post), FG: th.Dim},
	}
}

func logsLevelFilterRow(th theme.Theme, lay Layout, enabled map[string]bool) []Segment {
	segs := []Segment{{Text: " ", FG: th.Text}}
	add := func(label string, on bool, color string) {
		if on {
			segs = append(segs, Segment{Text: "[", FG: color})
			segs = append(segs, Segment{Text: label, FG: color, Bold: true})
			segs = append(segs, Segment{Text: "]", FG: color})
		} else {
			segs = append(segs, Segment{Text: "[", FG: th.Dim})
			segs = append(segs, Segment{Text: label, FG: th.Dim, Strike: true})
			segs = append(segs, Segment{Text: "]", FG: th.Dim})
		}
		segs = append(segs, Segment{Text: " ", FG: th.Text})
	}
	add("err", enabled["err"], th.Error)
	add("warn", enabled["warn"], th.Warn)
	add("info", enabled["info"], th.Info)
	add("dbg", enabled["debug"], th.Debug)
	segs = append(segs, Segment{Text: " toggle with 1/2/3/4 · esc closes", FG: th.Dim})
	return fitRow(segs, lay.LogsW, th.Text)
}

// LogLine is the structured form of a streamed line.
type LogLine struct {
	TS    string
	Level string
	Msg   string
}

// ParseLine extracts level + timestamp + message from a raw log string.
// Recognises lines starting with HH:MM:SS, falls back to "now".
func ParseLine(raw, fallbackTS string) LogLine {
	s := sanitizeRow(strings.TrimRight(raw, "\r\n"))
	ts := fallbackTS
	if len(s) >= 8 && isTS(s[:8]) {
		ts = s[:8]
		s = strings.TrimLeft(s[8:], " \t")
	}
	level := "INFO"
	up := strings.ToUpper(s)
	switch {
	case strings.HasPrefix(up, "ERROR"), strings.HasPrefix(up, "ERR "), strings.Contains(up, "ERROR"), strings.Contains(up, "FATAL"):
		level = "ERROR"
	case strings.HasPrefix(up, "WARN"), strings.HasPrefix(up, "WARNING"):
		level = "WARN"
	case strings.HasPrefix(up, "DEBUG"), strings.HasPrefix(up, "DBG"):
		level = "DEBUG"
	case strings.HasPrefix(up, "INFO"):
		level = "INFO"
	}
	for _, p := range []string{"ERROR ", "WARN ", "WARNING ", "DEBUG ", "INFO "} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimPrefix(s, p)
			break
		}
	}
	return LogLine{TS: ts, Level: level, Msg: s}
}

func isTS(s string) bool {
	if len(s) != 8 {
		return false
	}
	return s[2] == ':' && s[5] == ':' && isDigit(s[0]) && isDigit(s[1]) && isDigit(s[3]) && isDigit(s[4]) && isDigit(s[6]) && isDigit(s[7])
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// logsLineRows renders one log entry as one (truncate mode) or many (wrap mode) rows.
func logsLineRows(th theme.Theme, lay Layout, line LogLine, query string, showCursor, wrap bool) [][]Segment {
	var rowBG string
	if line.Level == "ERROR" {
		rowBG = th.ErrRowBg
	} else if line.Level == "WARN" {
		rowBG = th.WarnRowBg
	}
	tsLen := 8
	levelLen := 5
	// Width available for the message column (one-row case).
	msgCol := lay.LogsW - 1 - tsLen - 2 - levelLen - 2 - 1
	if showCursor {
		msgCol--
	}
	if msgCol < 4 {
		msgCol = 4
	}

	levelColor := th.Text
	switch line.Level {
	case "ERROR":
		levelColor = th.Error
	case "WARN":
		levelColor = th.Warn
	case "INFO":
		levelColor = th.Info
	case "DEBUG":
		levelColor = th.Debug
	}

	// Build msg segments for a given chunk of the message string, with optional
	// search-highlight. `withCursor` appends the live-tail cursor at the end.
	mkMsgSegs := func(chunk string, withCursor bool) []Segment {
		var out []Segment
		if query != "" {
			needle := strings.ToLower(query)
			lower := strings.ToLower(chunk)
			i := 0
			for i < len(chunk) {
				found := strings.Index(lower[i:], needle)
				if found < 0 {
					out = append(out, Segment{Text: chunk[i:], FG: th.Text, BG: rowBG})
					break
				}
				if found > 0 {
					out = append(out, Segment{Text: chunk[i : i+found], FG: th.Text, BG: rowBG})
				}
				out = append(out, Segment{
					Text: chunk[i+found : i+found+len(needle)],
					FG:   th.MatchFg, BG: th.MatchBg, Bold: true,
				})
				i += found + len(needle)
			}
		} else {
			out = append(out, Segment{Text: chunk, FG: th.Text, BG: rowBG})
		}
		if withCursor {
			out = append(out, Segment{Text: "▌", FG: th.Accent, Bold: true, BG: rowBG})
		}
		return out
	}

	prefix := func() []Segment {
		return []Segment{
			{Text: " ", BG: rowBG},
			{Text: line.TS, FG: th.Dim, BG: rowBG},
			{Text: "  ", BG: rowBG},
			{Text: rpad(line.Level, levelLen), FG: levelColor, Bold: true, BG: rowBG},
			{Text: "  ", BG: rowBG},
		}
	}
	// Continuation rows replace the timestamp+level columns with spaces so the
	// wrapped text aligns visually with the start of the message column.
	contPrefix := func() []Segment {
		return []Segment{
			{Text: " " + repeat(" ", tsLen) + "  " + repeat(" ", levelLen) + "  ", BG: rowBG},
		}
	}

	if !wrap {
		msg := truncate(line.Msg, msgCol)
		segs := prefix()
		segs = append(segs, mkMsgSegs(msg, showCursor)...)
		return [][]Segment{fitRow(segs, lay.LogsW, th.Text)}
	}

	// Wrap mode: split message into chunks of msgCol cells. First chunk shares
	// a row with the timestamp/level prefix; subsequent chunks get a continuation prefix.
	runes := []rune(line.Msg)
	if len(runes) == 0 {
		segs := prefix()
		return [][]Segment{fitRow(segs, lay.LogsW, th.Text)}
	}
	var out [][]Segment
	for i := 0; i < len(runes); i += msgCol {
		end := i + msgCol
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		var segs []Segment
		if i == 0 {
			segs = prefix()
		} else {
			segs = contPrefix()
		}
		isLastChunk := end == len(runes)
		segs = append(segs, mkMsgSegs(chunk, isLastChunk && showCursor)...)
		out = append(out, fitRow(segs, lay.LogsW, th.Text))
	}
	return out
}

func stubTabContent(th theme.Theme, lay Layout, tabID string, contentRows int) [][]Segment {
	titles := map[string]string{"build": "Build Logs", "config": "Config", "env": "Environment", "deploys": "Deployments"}
	subs := map[string]string{
		"build":   "GET /api/deployments/{uuid}/build-log",
		"config":  "GET /api/services/{uuid}/config",
		"env":     "GET /api/services/{uuid}/env",
		"deploys": "GET /api/services/{uuid}/deployments",
	}
	title := titles[tabID]
	sub := subs[tabID]
	if title == "" {
		title = "Coming soon"
	}

	out := [][]Segment{
		logsBlankRow(th, lay), logsBlankRow(th, lay), logsBlankRow(th, lay),
	}

	csTxt := "Coming soon"
	// Box must fit both the title (in the top border) and the "Coming soon"
	// label on its centred line. Reserve 2 cells for the side borders and a
	// little breathing room.
	boxW := runeLen(title) + 6
	if min := runeLen(csTxt) + 4; boxW < min {
		boxW = min
	}
	if boxW > lay.LogsW-4 {
		boxW = lay.LogsW - 4
	}
	if boxW < 8 {
		boxW = 8
	}
	padLeft := (lay.LogsW - boxW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	titleFit := truncate(title, boxW-4)
	topDashes := boxW - 4 - runeLen(titleFit)
	if topDashes < 1 {
		topDashes = 1
	}
	topBox := []Segment{
		{Text: repeat(" ", padLeft), FG: th.Text},
		{Text: BTL + BH + " " + titleFit + " " + repeat(BH, topDashes) + BTR, FG: th.Border},
	}
	midBox := []Segment{
		{Text: repeat(" ", padLeft), FG: th.Text},
		{Text: BV + repeat(" ", boxW-2) + BV, FG: th.Border},
	}
	csFit := truncate(csTxt, boxW-2)
	csPad := (boxW - 2 - runeLen(csFit)) / 2
	if csPad < 0 {
		csPad = 0
	}
	csRight := boxW - 2 - csPad - runeLen(csFit)
	if csRight < 0 {
		csRight = 0
	}
	csBox := []Segment{
		{Text: repeat(" ", padLeft), FG: th.Text},
		{Text: BV, FG: th.Border},
		{Text: repeat(" ", csPad), FG: th.Text},
		{Text: csFit, FG: th.Muted, Bold: true},
		{Text: repeat(" ", csRight), FG: th.Text},
		{Text: BV, FG: th.Border},
	}
	botBox := []Segment{
		{Text: repeat(" ", padLeft), FG: th.Text},
		{Text: BBL + repeat(BH, boxW-2) + BBR, FG: th.Border},
	}

	out = append(out, fitRow(topBox, lay.LogsW, th.Text))
	out = append(out, fitRow(midBox, lay.LogsW, th.Text))
	out = append(out, fitRow(csBox, lay.LogsW, th.Text))
	out = append(out, fitRow(midBox, lay.LogsW, th.Text))
	out = append(out, fitRow(botBox, lay.LogsW, th.Text))
	out = append(out, logsBlankRow(th, lay))
	out = append(out, logsBlankRow(th, lay))

	desc := []string{
		"This view needs the gateway to expose an endpoint:",
		"",
		"   " + sub,
		"",
		"Not yet implemented in the backend — runtime logs are",
		"available on the Logs tab.",
		"",
		"See ../HANDOFF.md → \"Known Issues / Open Threads\".",
	}
	for _, ln := range desc {
		color := th.Muted
		if strings.HasPrefix(ln, "   GET") {
			color = th.Accent
		}
		out = append(out, fitRow([]Segment{
			{Text: "   ", FG: th.Text},
			{Text: ln, FG: color},
		}, lay.LogsW, th.Text))
	}
	for len(out) < contentRows {
		out = append(out, logsBlankRow(th, lay))
	}
	return out[:contentRows]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// LogsState bundles the bits needed to render the logs pane.
type LogsState struct {
	ActiveTab       string
	Logs            []LogLine
	Levels          map[string]bool
	ShowLevelFilter bool
	SearchOpen      bool
	SearchQuery     string
	Paused          bool
	QueuedLogs      int
	StreamDropped   bool
	Wrap            bool
	ScrollOffset    int // lines from the tail; 0 = follow newest
	Deploys         DeploysState
	Build           BuildState
	Config          ConfigState
	Env             EnvState
}

func logsContent(th theme.Theme, lay Layout, st LogsState, contentRows int) [][]Segment {
	rows := [][]Segment{
		logsTabsRow(th, lay, st.ActiveTab),
		logsTabsSeparator(th, lay, st.ActiveTab),
	}
	switch st.ActiveTab {
	case "deploys":
		return append(rows, deploysContent(th, lay, st.Deploys, contentRows-2)...)
	case "build":
		return append(rows, buildContent(th, lay, st.Build, contentRows-2, st.Wrap)...)
	case "config":
		return append(rows, configContent(th, lay, st.Config, contentRows-2)...)
	case "env":
		return append(rows, envContent(th, lay, st.Env, contentRows-2)...)
	case "logs":
		// fall through to log-stream rendering below
	default:
		return append(rows, stubTabContent(th, lay, st.ActiveTab, contentRows-2)...)
	}

	if st.ShowLevelFilter {
		rows = append(rows, logsLevelFilterRow(th, lay, st.Levels))
	}

	if st.ScrollOffset > 0 {
		banner := fmt.Sprintf(" ↑ scrolled %d from tail · G to follow ", st.ScrollOffset)
		rows = append(rows, fitRow([]Segment{
			{Text: banner, FG: th.AccentFg, BG: th.Accent, Bold: true},
		}, lay.LogsW, th.Text))
	}

	visible := make([]LogLine, 0, len(st.Logs))
	for _, l := range st.Logs {
		ok := true
		switch l.Level {
		case "ERROR":
			ok = st.Levels["err"]
		case "WARN":
			ok = st.Levels["warn"]
		case "INFO":
			ok = st.Levels["info"]
		case "DEBUG":
			ok = st.Levels["debug"]
		}
		if ok {
			visible = append(visible, l)
		}
	}

	reserved := len(rows)
	bottom := 0
	if st.SearchOpen {
		bottom = 3
	}
	if st.StreamDropped {
		bottom++
	}
	slots := contentRows - reserved - bottom
	if slots < 0 {
		slots = 0
	}

	q := ""
	if st.SearchOpen {
		q = st.SearchQuery
	}

	// Anchor is the newest *visible* line offset by ScrollOffset (in entries).
	// ScrollOffset is expressed in the unfiltered log buffer, so map it through
	// the filtered visible list by clamping.
	anchorEnd := len(visible) - st.ScrollOffset
	if anchorEnd > len(visible) {
		anchorEnd = len(visible)
	}
	if anchorEnd < 0 {
		anchorEnd = 0
	}

	// Walk backwards from anchorEnd, gathering entries until we fill `slots`.
	var rendered [][][]Segment
	used := 0
	for i := anchorEnd - 1; i >= 0; i-- {
		isAnchor := i == anchorEnd-1
		// Cursor only when following the tail (offset 0) AND we're at the newest.
		showCursor := isAnchor && st.ScrollOffset == 0 && !st.Paused && !st.SearchOpen
		entryRows := logsLineRows(th, lay, visible[i], q, showCursor, st.Wrap)
		rendered = append([][][]Segment{entryRows}, rendered...)
		used += len(entryRows)
		if used >= slots {
			over := used - slots
			if over > 0 && len(rendered[0]) > over {
				rendered[0] = rendered[0][over:]
				used = slots
			} else if over >= len(rendered[0]) {
				rendered = rendered[1:]
				used -= len(entryRows)
			}
			break
		}
	}

	// Top-pad if we under-filled.
	if used < slots {
		for i := 0; i < slots-used; i++ {
			rows = append(rows, logsBlankRow(th, lay))
		}
	}
	for _, entry := range rendered {
		rows = append(rows, entry...)
	}

	if st.Paused && len(rows) > 0 {
		pill := fmt.Sprintf("[ PAUSED · %d new lines ]", st.QueuedLogs)
		pad := lay.LogsW - runeLen(pill) - 2
		if pad < 0 {
			pad = 0
		}
		rows[len(rows)-1] = fitRow([]Segment{
			{Text: repeat(" ", pad), FG: th.Text},
			{Text: " " + pill + " ", FG: th.AccentFg, BG: th.Accent, Bold: true},
		}, lay.LogsW, th.Text)
	}

	if st.StreamDropped {
		rows = append(rows, fitRow([]Segment{
			{Text: " ▲ ", FG: th.Warn, Bold: true},
			{Text: "stream disconnected · auto-retry in 3s · ", FG: th.Muted},
			{Text: "r", FG: th.Accent, Bold: true},
			{Text: " retry now", FG: th.Muted},
		}, lay.LogsW, th.Text))
	}

	if st.SearchOpen {
		matchCount := 0
		needle := strings.ToLower(st.SearchQuery)
		if needle != "" {
			for _, l := range st.Logs {
				if strings.Contains(strings.ToLower(l.Msg), needle) {
					matchCount++
				}
			}
		}
		rows = append(rows, fitRow([]Segment{
			{Text: repeat(BH, 3) + " search ", FG: th.Accent, Bold: true},
			{Text: repeat(BH, lay.LogsW-3-8), FG: th.Accent},
		}, lay.LogsW, th.Accent))
		queryShow := st.SearchQuery
		fg := th.Text
		if queryShow == "" {
			queryShow = "type to search…"
			fg = th.Dim
		}
		rows = append(rows, fitRow([]Segment{
			{Text: " / ", FG: th.Accent, Bold: true},
			{Text: queryShow, FG: fg},
			{Text: "▌", FG: th.Accent, Bold: true},
		}, lay.LogsW, th.Text))
		rows = append(rows, fitRow([]Segment{
			{Text: repeat(BH, 3) + " ", FG: th.Accent},
			{Text: fmt.Sprintf("%d matches", matchCount), FG: th.Muted, Bold: true},
			{Text: " · ", FG: th.Dim},
			{Text: "n", FG: th.Accent, Bold: true}, {Text: " next  ", FG: th.Muted},
			{Text: "N", FG: th.Accent, Bold: true}, {Text: " prev  ", FG: th.Muted},
			{Text: "esc", FG: th.Accent, Bold: true}, {Text: " cancel ", FG: th.Muted},
			{Text: repeat(BH, lay.LogsW), FG: th.Accent},
		}, lay.LogsW, th.Accent))
	}

	for len(rows) < contentRows {
		rows = append(rows, logsBlankRow(th, lay))
	}
	return rows[:contentRows]
}
