package tui

import (
	"strings"

	"github.com/princepatel/logify/internal/theme"
)

// buildMain assembles the main view (v2: logs-only, no sidebar).
func buildMain(th theme.Theme, lay Layout, st screenState) []Row {
	var rows []Row
	rows = append(rows, topBorderRow(th, lay, st.SelectedPath, st.Connection, st.tailIfLogs()))
	rows = append(rows, spacerRow(th, lay))

	// One inner box around the logs pane (always "focused" since there's no
	// other pane to give focus to).
	colLog := th.BorderFoc
	innerW := lay.TotalW - 4
	rows = append(rows, edgeRow(th, lay, []Segment{
		{Text: " ", FG: th.Border},
		{Text: BTL + strings.Repeat(BH, innerW-2) + BTR, FG: colLog},
		{Text: " ", FG: th.Border},
	}))

	contentRows := lay.TotalH - 7
	if contentRows < 4 {
		contentRows = 4
	}
	logs := logsContent(th, lay, st.LogsState, contentRows)
	for i := 0; i < contentRows; i++ {
		row := []Segment{
			{Text: " ", FG: th.Border},
			{Text: BV, FG: colLog},
		}
		row = append(row, logs[i]...)
		row = append(row, Segment{Text: BV, FG: colLog})
		row = append(row, Segment{Text: " ", FG: th.Border})
		rows = append(rows, edgeRow(th, lay, row))
	}
	rows = append(rows, edgeRow(th, lay, []Segment{
		{Text: " ", FG: th.Border},
		{Text: BBL + strings.Repeat(BH, innerW-2) + BBR, FG: colLog},
		{Text: " ", FG: th.Border},
	}))
	rows = append(rows, spacerRow(th, lay))
	rows = append(rows, edgeRow(th, lay, helpStripInterior(th, lay, st.Notice)))
	rows = append(rows, bottomBorderRow(th, lay))
	return rows
}

// buildMainFullscreen renders the logs pane across the full width.
func buildMainFullscreen(th theme.Theme, lay Layout, st screenState) []Row {
	var rows []Row
	rows = append(rows, topBorderRow(th, lay, st.SelectedPath+"  (zoom)", st.Connection, st.tailIfLogs()))
	rows = append(rows, spacerRow(th, lay))

	// Inner top border for the logs pane (full width).
	innerW := lay.TotalW - 4 // outer + gutters
	colLog := th.BorderFoc
	rows = append(rows, edgeRow(th, lay, []Segment{
		{Text: " ", FG: th.Border},
		{Text: BTL + repeat(BH, innerW-2) + BTR, FG: colLog},
		{Text: " ", FG: th.Border},
	}))

	contentRows := lay.TotalH - 7
	if contentRows < 4 {
		contentRows = 4
	}
	fsLay := lay
	fsLay.LogsW = innerW - 2
	logs := logsContent(th, fsLay, st.LogsState, contentRows)
	for i := 0; i < contentRows; i++ {
		row := []Segment{
			{Text: " ", FG: th.Border},
			{Text: BV, FG: colLog},
		}
		row = append(row, logs[i]...)
		row = append(row, Segment{Text: BV, FG: colLog})
		row = append(row, Segment{Text: " ", FG: th.Border})
		rows = append(rows, edgeRow(th, lay, row))
	}

	// Inner bottom border.
	rows = append(rows, edgeRow(th, lay, []Segment{
		{Text: " ", FG: th.Border},
		{Text: BBL + repeat(BH, innerW-2) + BBR, FG: colLog},
		{Text: " ", FG: th.Border},
	}))
	rows = append(rows, spacerRow(th, lay))
	rows = append(rows, edgeRow(th, lay, helpStripInterior(th, lay, st.Notice)))
	rows = append(rows, bottomBorderRow(th, lay))
	return rows
}

func (s screenState) tailIfLogs() int {
	if s.LogsState.ActiveTab == "logs" {
		return s.TailSize
	}
	return 0
}

// centeredBoxScreen builds a generic centred-message view (connecting, errors).
func centeredBoxScreen(th theme.Theme, lay Layout, lines [][]Segment, conn string) []Row {
	var rows []Row
	rows = append(rows, topBorderRow(th, lay, "", conn, 0))
	rows = append(rows, spacerRow(th, lay))
	fill := lay.TotalH - 4
	top := (fill - len(lines)) / 2
	for i := 0; i < fill; i++ {
		li := i - top
		if li >= 0 && li < len(lines) {
			used := segWidth(lines[li])
			pad := (lay.TotalW - 2 - used) / 2
			interior := []Segment{{Text: repeat(" ", pad), FG: th.Text}}
			interior = append(interior, lines[li]...)
			interior = append(interior, Segment{Text: repeat(" ", lay.TotalW-2-pad-used), FG: th.Text})
			rows = append(rows, edgeRow(th, lay, interior))
		} else {
			rows = append(rows, spacerRow(th, lay))
		}
	}
	rows = append(rows, edgeRow(th, lay, helpStripInterior(th, lay, "")) )
	rows = append(rows, bottomBorderRow(th, lay))
	return rows
}

func buildConnecting(th theme.Theme, lay Layout, target string, frame int) []Row {
	// 10-cell knight-rider bar: a 4-cell filled window rotates through 10 slots.
	const total = 10
	const window = 4
	pos := frame % total
	bar := make([]byte, 0, total*4)
	barSeg := []Segment{}
	for i := 0; i < total; i++ {
		on := false
		for w := 0; w < window; w++ {
			if (pos+w)%total == i {
				on = true
				break
			}
		}
		if on {
			barSeg = append(barSeg, Segment{Text: "▰", FG: th.Accent, Bold: true})
		} else {
			barSeg = append(barSeg, Segment{Text: "▱", FG: th.Dim})
		}
		_ = bar
	}
	barSeg = append(barSeg, Segment{Text: "  ", FG: th.Text})
	barSeg = append(barSeg, Segment{Text: "fetching projects", FG: th.Dim})

	// Dot pulse on the "connecting to" line uses the same frame counter.
	dotColors := []string{th.Warn, th.Accent, th.Warn, th.Dim}
	dotCol := dotColors[frame%len(dotColors)]
	ellipsis := []string{" …", " ‥", "  ", " ·"}[frame%4]

	return centeredBoxScreen(th, lay, [][]Segment{
		{{Text: "●", FG: dotCol, Bold: true}, {Text: " connecting to ", FG: th.Muted}, {Text: target, FG: th.Text, Bold: true}, {Text: ellipsis, FG: th.Muted}},
		{},
		barSeg,
	}, "connecting")
}

// wrapBordered surrounds an `innerW`-wide row of segments with the card's
// left/right `│` borders + the requested left indent.
func wrapBordered(row []Segment, leftCol int, accent string, W int) []Segment {
	out := []Segment{{Text: BV + repeat(" ", leftCol+1), FG: accent}}
	out = append(out, row...)
	out = append(out, Segment{Text: " " + BV, FG: accent})
	return out
}

// errorCard is the unified panel for lifecycle error screens.
// kind: "error" | "warn" | "info" — controls icon + accent color.
type errorCard struct {
	Kind     string
	Title    string
	Subtitle string             // one-line context under the title (optional)
	Body     [][]Segment        // detail rows (each <= W-6 cells)
	Hints    []string           // bullet hints under "What to try"
	Actions  []errorCardAction  // bottom action keys
	Conn     string             // header pill: "connecting" | "disconnected"
	Frame    int                // for animated icon
}

type errorCardAction struct {
	Key, Label string
}

func buildErrorCard(th theme.Theme, lay Layout, c errorCard) []Row {
	W := 76
	if W > lay.TotalW-8 {
		W = lay.TotalW - 8
	}
	if W < 40 {
		W = 40
	}

	// Accent color for the card border + icon, derived from kind.
	accent := th.Error
	icon := "✕"
	switch c.Kind {
	case "warn":
		accent = th.Warn
		icon = "▲"
	case "info":
		accent = th.Info
		icon = "◆"
	}
	// Subtle pulse on the icon: alternate accent ↔ dim every ~360 ms.
	if (c.Frame/3)%2 == 1 {
		// dim phase — softened by using muted instead of accent
		_ = icon
	}

	// Card body line builders ------------------------------------------------

	border := func() []Segment {
		return []Segment{{Text: BV + repeat(" ", W-2) + BV, FG: accent}}
	}
	// padWrap returns one or more rows for `inner`, wrapping by characters if
	// the content exceeds the interior width. Styles are preserved across wraps.
	padWrap := func(inner []Segment, leftCol int, hangIndent int) [][]Segment {
		innerW := W - 4 - leftCol
		if innerW < 1 {
			innerW = 1
		}
		if len(inner) == 0 {
			return [][]Segment{wrapBordered([]Segment{{Text: repeat(" ", innerW), FG: th.Text}}, leftCol, accent, W)}
		}
		// Flatten to cells while remembering style.
		type cell struct{ ch string; s Segment }
		var cells []cell
		for _, s := range inner {
			for _, r := range s.Text {
				cells = append(cells, cell{string(r), s})
			}
		}
		var out [][]Segment
		hang := repeat(" ", hangIndent)
		for i := 0; i < len(cells); {
			rowCells := cells[i:min(i+innerW, len(cells))]
			// On wrap continuation, prepend hanging indent (visual only).
			row := []Segment{}
			if i > 0 && hangIndent > 0 {
				row = append(row, Segment{Text: hang, FG: th.Text})
			}
			var cur *Segment
			for _, c := range rowCells {
				if cur != nil && cur.FG == c.s.FG && cur.BG == c.s.BG && cur.Bold == c.s.Bold && cur.Strike == c.s.Strike {
					cur.Text += c.ch
					row[len(row)-1] = *cur
				} else {
					row = append(row, Segment{Text: c.ch, FG: c.s.FG, BG: c.s.BG, Bold: c.s.Bold, Strike: c.s.Strike})
					cur = &row[len(row)-1]
				}
			}
			row = fitRow(row, innerW, th.Text)
			out = append(out, wrapBordered(row, leftCol, accent, W))
			i += len(rowCells)
		}
		return out
	}

	var lines [][]Segment

	// Top border with title strip
	titleStr := " " + icon + "  " + c.Title + " "
	titleLen := runeLen(titleStr)
	dashes := W - 2 - titleLen
	if dashes < 2 {
		dashes = 2
	}
	lines = append(lines, []Segment{
		{Text: BTL + titleStr, FG: accent, Bold: true},
		{Text: repeat(BH, dashes-1) + BTR, FG: accent},
	})

	lines = append(lines, border())

	// Subtitle
	if c.Subtitle != "" {
		lines = append(lines, padWrap([]Segment{
			{Text: c.Subtitle, FG: th.Muted},
		}, 2, 2)...)
		lines = append(lines, border())
	}

	// Body rows
	for _, b := range c.Body {
		lines = append(lines, padWrap(b, 2, 4)...)
	}

	if len(c.Hints) > 0 {
		lines = append(lines, border())
		lines = append(lines, padWrap([]Segment{
			{Text: "What to try", FG: th.Text, Bold: true},
		}, 2, 2)...)
		for _, h := range c.Hints {
			lines = append(lines, padWrap([]Segment{
				{Text: "• ", FG: accent, Bold: true},
				{Text: h, FG: th.Muted},
			}, 4, 6)...)
		}
	}

	// Actions footer
	if len(c.Actions) > 0 {
		lines = append(lines, border())
		seg := []Segment{}
		for i, a := range c.Actions {
			if i > 0 {
				seg = append(seg, Segment{Text: "   ", FG: th.Text})
			}
			seg = append(seg, Segment{Text: " " + a.Key + " ", FG: th.AccentFg, BG: accent, Bold: true})
			seg = append(seg, Segment{Text: "  " + a.Label, FG: th.Muted})
		}
		lines = append(lines, padWrap(seg, 2, 2)...)
	}

	lines = append(lines, border())
	lines = append(lines, []Segment{{Text: BBL + repeat(BH, W-2) + BBR, FG: accent}})

	// Compose into full-screen rows --------------------------------------------

	conn := c.Conn
	if conn == "" {
		conn = "disconnected"
	}

	var rows []Row
	rows = append(rows, topBorderRow(th, lay, "", conn, 0))
	rows = append(rows, spacerRow(th, lay))

	fill := lay.TotalH - 4
	top := (fill - len(lines)) / 2
	if top < 0 {
		top = 0
	}
	left := (lay.TotalW - W) / 2
	if left < 1 {
		left = 1
	}

	for i := 0; i < fill; i++ {
		li := i - top
		if li >= 0 && li < len(lines) {
			segs := lines[li]
			used := segWidth(segs)
			interior := []Segment{{Text: repeat(" ", left-1), FG: th.Text}}
			interior = append(interior, segs...)
			tail := lay.TotalW - 2 - (left - 1) - used
			if tail < 0 {
				tail = 0
			}
			interior = append(interior, Segment{Text: repeat(" ", tail), FG: th.Text})
			rows = append(rows, edgeRow(th, lay, interior))
		} else {
			rows = append(rows, spacerRow(th, lay))
		}
	}
	rows = append(rows, edgeRow(th, lay, helpStripInterior(th, lay, "")))
	rows = append(rows, bottomBorderRow(th, lay))
	return rows
}

// friendlyNetError translates raw Go net errors into a one-liner most humans
// can act on. Returns (headline, suggestedHints).
func friendlyNetError(raw string) (string, []string) {
	r := raw
	switch {
	case raw == "":
		return "Connection failed", []string{
			"Check the gateway URL in your config.",
		}
	case containsAny(r, "connection refused"):
		return "Gateway is not accepting connections", []string{
			"Is the gateway container running?  `docker compose ps` on the host.",
			"Check the port: default is :8089 inside compose, :8080 inside the container.",
		}
	case containsAny(r, "no such host", "name resolution"):
		return "DNS lookup failed", []string{
			"The hostname in your gateway URL didn't resolve.",
			"Try the IP directly to isolate DNS vs reachability.",
		}
	case containsAny(r, "i/o timeout", "deadline exceeded"):
		return "Connection timed out", []string{
			"The host is reachable but the gateway isn't responding.",
			"Check firewall rules / reverse-proxy logs on the gateway host.",
		}
	case containsAny(r, "tls", "certificate", "x509"):
		return "TLS handshake failed", []string{
			"Certificate didn't validate — check the gateway's cert and your system trust store.",
		}
	case containsAny(r, "network is unreachable", "host is down"):
		return "Network is unreachable", []string{
			"Are you on a VPN? The gateway may only be exposed on the internal network.",
		}
	}
	return "Could not reach the gateway", []string{
		"Verify the gateway URL and try again.",
	}
}

func containsAny(s string, needles ...string) bool {
	low := lowerASCII(s)
	for _, n := range needles {
		if indexOf(low, lowerASCII(n)) >= 0 {
			return true
		}
	}
	return false
}

func lowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
func indexOf(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func buildErrorUnreachable(th theme.Theme, lay Layout, target, reason string, frame int) []Row {
	headline, hints := friendlyNetError(reason)
	body := [][]Segment{
		{{Text: "GET ", FG: th.Dim}, {Text: target, FG: th.Text, Bold: true}},
		{},
	}
	if reason != "" {
		body = append(body, []Segment{
			{Text: reason, FG: th.Error},
		})
	}
	return buildErrorCard(th, lay, errorCard{
		Kind:     "error",
		Title:    "Cannot reach gateway",
		Subtitle: headline,
		Body:     body,
		Hints:    hints,
		Actions: []errorCardAction{
			{Key: "r", Label: "retry"},
			{Key: "q", Label: "quit"},
		},
		Conn:  "disconnected",
		Frame: frame,
	})
}

func buildErrorAuth(th theme.Theme, lay Layout, cfgPath string, frame int) []Row {
	return buildErrorCard(th, lay, errorCard{
		Kind:     "error",
		Title:    "Authentication failed",
		Subtitle: "The gateway rejected your API key.",
		Body: [][]Segment{
			{{Text: "Config:  ", FG: th.Dim}, {Text: cfgPath, FG: th.Text, Bold: true}},
		},
		Hints: []string{
			"Double-check the key — copy/paste errors are the usual culprit.",
			"Ask an admin to confirm the key still exists in the gateway DB.",
			"Re-run with --token <key> to override the saved value.",
		},
		Actions: []errorCardAction{
			{Key: "q", Label: "quit"},
		},
		Conn:  "disconnected",
		Frame: frame,
	})
}

func buildErrorEmpty(th theme.Theme, lay Layout, frame int) []Row {
	return buildErrorCard(th, lay, errorCard{
		Kind:     "warn",
		Title:    "No services accessible",
		Subtitle: "Your API key authenticates, but isn't allowed for any container.",
		Body: [][]Segment{
			{{Text: "Nothing to stream until access is granted.", FG: th.Muted}},
		},
		Hints: []string{
			"Ask an admin to assign at least one container to your key.",
			"If you just got access, press r to refresh the project list.",
		},
		Actions: []errorCardAction{
			{Key: "r", Label: "refresh"},
			{Key: "q", Label: "quit"},
		},
		Conn:  "connected",
		Frame: frame,
	})
}

// buildFirstRun renders the first-run config form.
func buildFirstRun(th theme.Theme, lay Layout, urlVal, tokenVal string, focused int, cfgPath string) []Row {
	W := 70
	if W > lay.TotalW-4 {
		W = lay.TotalW - 4
	}
	left := (lay.TotalW - W) / 2

	mkField := func(label, value string, isFocused, masked bool) []Segment {
		labelCol := 14
		inputCol := W - labelCol - 7
		display := value
		if masked && value != "" {
			display = repeat("•", min(len(value), inputCol-4))
		}
		barColor := th.Border
		if isFocused {
			barColor = th.Accent
		}
		cursor := " "
		if isFocused {
			cursor = "▌"
		}
		return []Segment{
			{Text: BV + "  ", FG: th.Accent},
			{Text: rpad(label, labelCol), FG: th.Muted, Bold: true},
			{Text: BV, FG: barColor},
			{Text: " ", FG: th.Text},
			{Text: rpad(truncate(display, inputCol-2), inputCol-2), FG: th.Text, Bold: isFocused},
			{Text: cursor, FG: th.Accent, Bold: true},
			{Text: " " + BV, FG: th.Accent},
		}
	}

	mkButtons := func(focusBtn int) []Segment {
		segs := []Segment{{Text: BV + "   ", FG: th.Accent}}
		btns := []string{"Save & Continue", "Quit"}
		for i, b := range btns {
			if i == focusBtn {
				segs = append(segs, Segment{Text: " " + b + " ", FG: th.AccentFg, BG: th.Accent, Bold: true})
			} else {
				segs = append(segs, Segment{Text: " [ " + b + " ] ", FG: th.Muted})
			}
			segs = append(segs, Segment{Text: "  "})
		}
		used := segWidth(segs)
		segs = append(segs, Segment{Text: repeat(" ", W-used-1), FG: th.Text})
		segs = append(segs, Segment{Text: BV, FG: th.Accent})
		return segs
	}

	urlField := mkField("Gateway URL", urlVal, focused == 0, false)
	tokField := mkField("API Key", tokenVal, focused == 1, true)
	btnLine := mkButtons(focused - 2)

	lines := [][]Segment{
		{{Text: BTL + BH + " logify · first run ", FG: th.Accent, Bold: true}, {Text: repeat(BH, W-22) + BTR, FG: th.Accent}},
		{{Text: BV + repeat(" ", W-2) + BV, FG: th.Accent}},
		urlField,
		tokField,
		{{Text: BV + repeat(" ", W-2) + BV, FG: th.Accent}},
		btnLine,
		{{Text: BV + repeat(" ", W-2) + BV, FG: th.Accent}},
		{
			{Text: BV + "   ", FG: th.Accent},
			{Text: "Saved to ", FG: th.Dim},
			{Text: cfgPath, FG: th.Muted},
			{Text: repeat(" ", max(0, W-5-runeLen("Saved to ")-runeLen(cfgPath))), FG: th.Text},
			{Text: " " + BV, FG: th.Accent},
		},
		{{Text: BBL + repeat(BH, W-2) + BBR, FG: th.Accent}},
	}

	var rows []Row
	rows = append(rows, topBorderRow(th, lay, "first run", "connecting", 0))
	rows = append(rows, spacerRow(th, lay))
	fill := lay.TotalH - 4
	top := (fill - len(lines)) / 2
	for i := 0; i < fill; i++ {
		li := i - top
		if li >= 0 && li < len(lines) {
			segs := lines[li]
			used := segWidth(segs)
			padL := left - 1
			if padL < 0 {
				padL = 0
			}
			interior := []Segment{{Text: repeat(" ", padL), FG: th.Text}}
			interior = append(interior, segs...)
			tail := lay.TotalW - 2 - padL - used
			if tail < 0 {
				tail = 0
			}
			interior = append(interior, Segment{Text: repeat(" ", tail), FG: th.Text})
			rows = append(rows, edgeRow(th, lay, interior))
		} else {
			rows = append(rows, spacerRow(th, lay))
		}
	}
	rows = append(rows, edgeRow(th, lay, helpStripInterior(th, lay, "")) )
	rows = append(rows, bottomBorderRow(th, lay))
	return rows
}

// buildThemePicker overlays a small theme list (interactive: ↑↓ + enter to apply).
func buildThemePicker(th theme.Theme, lay Layout, base []Row, themes []string, idx int) []Row {
	W := 36
	left := (lay.TotalW - W) / 2
	lines := [][]Segment{
		{{Text: BTL + BH + " theme ", FG: th.Accent, Bold: true}, {Text: repeat(BH, W-9) + BTR, FG: th.Accent}},
	}
	for i, id := range themes {
		t := theme.Get(id)
		marker := "  "
		nameFG := th.Text
		nameBG := ""
		if i == idx {
			marker = "▸ "
			nameFG = th.AccentFg
			nameBG = th.Accent
		}
		row := []Segment{
			{Text: BV + " ", FG: th.Accent},
			{Text: rpad(marker+t.Name, W-4), FG: nameFG, BG: nameBG, Bold: i == idx},
			{Text: " " + BV, FG: th.Accent},
		}
		lines = append(lines, row)
	}
	lines = append(lines, []Segment{{Text: BBL + repeat(BH, W-2) + BBR, FG: th.Accent}})

	top := (lay.TotalH - len(lines)) / 2
	out := make([]Row, len(base))
	copy(out, base)
	for i, ln := range lines {
		out[top+i] = paintOverlay(out[top+i], ln, left, W, th)
	}
	return out
}

// buildHelpOverlay paints a centered help modal on top of buildMain.
func buildHelpOverlay(th theme.Theme, lay Layout, base []Row) []Row {
	W := 64
	left := (lay.TotalW - W) / 2

	sec := func(label string) []Segment {
		return []Segment{
			{Text: BV + "  ", FG: th.Accent},
			{Text: rpad(label, W-5), FG: th.Accent, Bold: true},
			{Text: " " + BV, FG: th.Accent},
		}
	}
	key := func(k, desc string) []Segment {
		keyCol := 18
		descCol := W - 6 - keyCol
		return []Segment{
			{Text: BV + "    ", FG: th.Accent},
			{Text: rpad(k, keyCol), FG: th.Text, Bold: true},
			{Text: " ", FG: th.Text},
			{Text: rpad(desc, descCol), FG: th.Muted},
			{Text: " " + BV, FG: th.Accent},
		}
	}
	empty := []Segment{{Text: BV + repeat(" ", W-2) + BV, FG: th.Accent}}

	lines := [][]Segment{
		{{Text: BTL + BH + " keybindings ", FG: th.Accent, Bold: true}, {Text: repeat(BH, W-16) + BTR, FG: th.Accent}},
		empty,
		sec("Navigation"),
		key("↑ / ↓ / k / j", "move in current pane"),
		key("enter", "open service (focus logs)"),
		key("tab / shift+tab", "switch nav ↔ logs"),
		key("1 .. 5", "jump to tab"),
		empty,
		sec("Logs"),
		key("/", "search"),
		key("n / N", "next / previous match"),
		key("f", "toggle level filter strip"),
		key("space", "pause / resume tail"),
		key("w", "toggle line wrap"),
		key("z", "zoom logs (hide nav)"),
		key("y", "yank visible buffer to clipboard"),
		key("g / G", "jump top / bottom"),
		key("c", "clear buffer"),
		empty,
		sec("Global"),
		key("t", "cycle theme"),
		key("?", "toggle this overlay"),
		key("r", "reconnect current stream"),
		key("q / ctrl-c", "quit"),
		empty,
		{{Text: BV + repeat(" ", W-14), FG: th.Accent}, {Text: "esc to close ", FG: th.Dim}, {Text: BV, FG: th.Accent}},
		{{Text: BBL + repeat(BH, W-2) + BBR, FG: th.Accent}},
	}

	top := (lay.TotalH - len(lines)) / 2
	out := make([]Row, len(base))
	copy(out, base)
	for i, ln := range lines {
		out[top+i] = paintOverlay(out[top+i], ln, left, W, th)
	}
	return out
}

// paintOverlay replaces TotalW cells of `base` starting at column `left`,
// for `width` cells, with overlay row `ov`.
func paintOverlay(base Row, ov []Segment, left, width int, th theme.Theme) Row {
	cells := make([]Segment, 0, 256)
	for _, s := range base {
		for _, r := range s.Text {
			cells = append(cells, Segment{Text: string(r), FG: s.FG, BG: s.BG, Bold: s.Bold, Strike: s.Strike})
		}
	}
	for len(cells) < left+width {
		cells = append(cells, Segment{Text: " ", FG: th.Text})
	}
	ovCells := make([]Segment, 0, 256)
	for _, s := range ov {
		for _, r := range s.Text {
			bg := s.BG
			if bg == "" {
				bg = th.Bg
			}
			ovCells = append(ovCells, Segment{Text: string(r), FG: s.FG, BG: bg, Bold: s.Bold, Strike: s.Strike})
		}
	}
	for len(ovCells) < width {
		ovCells = append(ovCells, Segment{Text: " ", BG: th.Bg})
	}
	for i := 0; i < width; i++ {
		cells[left+i] = ovCells[i]
	}
	out := Row{}
	var cur *Segment
	for i := range cells {
		c := cells[i]
		if cur != nil && cur.FG == c.FG && cur.BG == c.BG && cur.Bold == c.Bold && cur.Strike == c.Strike {
			cur.Text += c.Text
			out[len(out)-1] = *cur
		} else {
			out = append(out, c)
			cur = &out[len(out)-1]
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// screenState bundles values for the screen builders.
type screenState struct {
	SelectedPath string // human-readable breadcrumb path
	SelectedKey  string // unique container name (used for nav match)
	Focused      string // "nav" | "logs"
	Connection   string // "connected" | "connecting" | "disconnected"
	TailSize     int
	NavRows      []ServiceRow
	NavQuery     string
	LogsState    LogsState
	Notice       string // transient toast, e.g. "Copied 42 lines"
	Fullscreen   bool   // hide nav pane and give logs the full width
}
