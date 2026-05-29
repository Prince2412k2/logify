package tui

import (
	"fmt"
	"strings"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/theme"
)

// EnvState bundles the bits needed to render the Env tab body.
type EnvState struct {
	ResourceUUID string
	Loading      bool
	Err          string
	Vars         []api.EnvVar
}

func envContent(th theme.Theme, lay Layout, st EnvState, contentRows int) [][]Segment {
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
	case st.Err != "":
		out = append(out, center("Could not load env", th.Error, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, center(truncate(st.Err, lay.LogsW-4), th.Muted, false))
	case st.Loading && len(st.Vars) == 0:
		out = append(out, center("Loading environment…", th.Muted, false))
	case len(st.Vars) == 0:
		out = append(out, center("No environment variables defined.", th.Muted, false))
	default:
		// Header
		keyW := 36
		flagsW := 18
		lenW := 8
		valW := lay.LogsW - 1 - keyW - 1 - flagsW - 1 - lenW - 2
		if valW < 10 {
			valW = 10
		}
		out = append(out, fitRow([]Segment{
			{Text: " ", FG: th.Text},
			{Text: rpad("KEY", keyW), FG: th.Muted, Bold: true},
			{Text: " ", FG: th.Text},
			{Text: rpad("FLAGS", flagsW), FG: th.Muted, Bold: true},
			{Text: " ", FG: th.Text},
			{Text: rpad("LEN", lenW), FG: th.Muted, Bold: true},
			{Text: " ", FG: th.Text},
			{Text: rpad("VALUE", valW), FG: th.Muted, Bold: true},
		}, lay.LogsW, th.Text))
		out = append(out, fitRow([]Segment{
			{Text: " ", FG: th.Text},
			{Text: strings.Repeat(BH, lay.LogsW-2), FG: th.Dim},
		}, lay.LogsW, th.Text))

		max := contentRows - len(out)
		for i, v := range st.Vars {
			if i >= max {
				break
			}
			flags := []string{}
			if v.IsBuildTime {
				flags = append(flags, "build")
			}
			if v.IsPreview {
				flags = append(flags, "preview")
			}
			if v.IsLiteral {
				flags = append(flags, "literal")
			}
			flagStr := strings.Join(flags, ",")
			if flagStr == "" {
				flagStr = "—"
			}
			mask := "•••••"
			if v.ValueLength == 0 {
				mask = "(empty)"
			}
			out = append(out, fitRow([]Segment{
				{Text: " ", FG: th.Text},
				{Text: rpad(truncate(v.Key, keyW), keyW), FG: th.Accent, Bold: true},
				{Text: " ", FG: th.Text},
				{Text: rpad(truncate(flagStr, flagsW), flagsW), FG: envFlagColor(th, flagStr)},
				{Text: " ", FG: th.Text},
				{Text: rpad(fmt.Sprintf("%d", v.ValueLength), lenW), FG: th.Muted},
				{Text: " ", FG: th.Text},
				{Text: rpad(mask, valW), FG: th.Dim},
			}, lay.LogsW, th.Text))
		}

		// Footer note about masking.
		if len(out) < contentRows-2 {
			out = append(out, logsBlankRow(th, lay))
			out = append(out, fitRow([]Segment{
				{Text: " ", FG: th.Text},
				{Text: "Values are masked. ", FG: th.Dim},
				{Text: "Read access to secrets is intentionally not exposed by the gateway.", FG: th.Dim},
			}, lay.LogsW, th.Text))
		}
	}

	for len(out) < contentRows {
		out = append(out, logsBlankRow(th, lay))
	}
	return out[:contentRows]
}

func envFlagColor(th theme.Theme, flags string) string {
	if flags == "—" {
		return th.Dim
	}
	if strings.Contains(flags, "preview") {
		return th.Warn
	}
	if strings.Contains(flags, "build") {
		return th.Info
	}
	return th.Muted
}
