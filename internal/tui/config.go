package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/theme"
)

// ConfigState bundles the bits needed to render the Config tab body.
type ConfigState struct {
	ResourceUUID string
	Type         string
	Loading      bool
	Err          string
	Config       api.ResourceConfig
}

type kv struct {
	Key, Val string
	Mono     bool
}

func appKVs(c api.ResourceConfig) []kv {
	out := []kv{
		{Key: "Name", Val: c.Name},
		{Key: "Status", Val: c.Status},
	}
	if c.FQDN != "" {
		out = append(out, kv{Key: "Domain", Val: c.FQDN, Mono: true})
	}
	if c.GitRepository != "" {
		out = append(out, kv{Key: "Git repo", Val: c.GitRepository, Mono: true})
	}
	if c.GitBranch != "" {
		out = append(out, kv{Key: "Branch", Val: c.GitBranch, Mono: true})
	}
	if c.GitCommit != "" {
		out = append(out, kv{Key: "Commit", Val: c.GitCommit, Mono: true})
	}
	if c.BuildPack != "" {
		out = append(out, kv{Key: "Build pack", Val: c.BuildPack})
	}
	if c.BaseDirectory != "" {
		out = append(out, kv{Key: "Base dir", Val: c.BaseDirectory, Mono: true})
	}
	if c.InstallCommand != "" {
		out = append(out, kv{Key: "Install", Val: c.InstallCommand, Mono: true})
	}
	if c.BuildCommand != "" {
		out = append(out, kv{Key: "Build", Val: c.BuildCommand, Mono: true})
	}
	if c.StartCommand != "" {
		out = append(out, kv{Key: "Start", Val: c.StartCommand, Mono: true})
	}
	if c.PortsExposed != "" {
		out = append(out, kv{Key: "Ports (exposed)", Val: c.PortsExposed, Mono: true})
	}
	if c.PortsMappings != "" {
		out = append(out, kv{Key: "Ports (mapped)", Val: c.PortsMappings, Mono: true})
	}
	if c.Dockerfile != "" {
		out = append(out, kv{Key: "Dockerfile", Val: c.Dockerfile, Mono: true})
	}
	if c.ImageName != "" {
		img := c.ImageName
		if c.ImageTag != "" {
			img += ":" + c.ImageTag
		}
		out = append(out, kv{Key: "Image", Val: img, Mono: true})
	}
	if c.HealthCheckEnabled {
		hc := "enabled"
		if c.HealthCheckPath != "" {
			hc += " · " + c.HealthCheckPath
		}
		out = append(out, kv{Key: "Health check", Val: hc, Mono: true})
	}
	if c.UpdatedAt > 0 {
		out = append(out, kv{Key: "Updated", Val: time.Unix(c.UpdatedAt, 0).Format("Jan 2 15:04")})
	}
	return out
}

func serviceKVs(c api.ResourceConfig) []kv {
	out := []kv{
		{Key: "Name", Val: c.Name},
		{Key: "Status", Val: c.Status},
	}
	if c.Description != "" {
		out = append(out, kv{Key: "Description", Val: c.Description})
	}
	if c.UpdatedAt > 0 {
		out = append(out, kv{Key: "Updated", Val: time.Unix(c.UpdatedAt, 0).Format("Jan 2 15:04")})
	}
	return out
}

func configContent(th theme.Theme, lay Layout, st ConfigState, contentRows int) [][]Segment {
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
		out = append(out, center("Could not load config", th.Error, true))
		out = append(out, logsBlankRow(th, lay))
		out = append(out, center(truncate(st.Err, lay.LogsW-4), th.Muted, false))
	case st.Loading && st.Config.Name == "":
		out = append(out, center("Loading config…", th.Muted, false))
	case st.Config.Name == "":
		out = append(out, center("Config not available.", th.Muted, false))
	default:
		c := st.Config
		var rows []kv
		switch c.Kind {
		case "service":
			rows = serviceKVs(c)
		default:
			rows = appKVs(c)
		}

		// Column 1 = key, column 2 = value.
		keyCol := 18
		valCol := lay.LogsW - 2 - keyCol - 1
		if valCol < 20 {
			valCol = 20
		}

		for _, row := range rows {
			val := truncate(row.Val, valCol)
			valFG := th.Text
			if row.Mono {
				valFG = th.Accent
			}
			out = append(out, fitRow([]Segment{
				{Text: " ", FG: th.Text},
				{Text: rpad(row.Key, keyCol), FG: th.Muted, Bold: true},
				{Text: " ", FG: th.Text},
				{Text: val, FG: valFG},
			}, lay.LogsW, th.Text))
		}

		// Service: tag on compose preview.
		if c.Kind == "service" && c.DockerComposeRaw != "" {
			out = append(out, logsBlankRow(th, lay))
			out = append(out, fitRow([]Segment{
				{Text: " ", FG: th.Text},
				{Text: "compose", FG: th.Muted, Bold: true},
			}, lay.LogsW, th.Text))
			out = append(out, fitRow([]Segment{
				{Text: " ", FG: th.Text},
				{Text: strings.Repeat(BH, lay.LogsW-2), FG: th.Dim},
			}, lay.LogsW, th.Text))
			for _, ln := range strings.Split(c.DockerComposeRaw, "\n") {
				out = append(out, fitRow([]Segment{
					{Text: " ", FG: th.Text},
					{Text: truncate(ln, lay.LogsW-2), FG: th.Dim},
				}, lay.LogsW, th.Text))
				if len(out) >= contentRows {
					break
				}
			}
		}
	}

	for len(out) < contentRows {
		out = append(out, logsBlankRow(th, lay))
	}
	return out[:contentRows]
}

var _ = fmt.Sprintf
