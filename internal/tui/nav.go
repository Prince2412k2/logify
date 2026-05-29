package tui

import (
	"strings"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/theme"
)

// ServiceRow is the flattened nav-item used for selection arithmetic.
type ServiceRow struct {
	Project       string
	Stage         string
	Service       string
	ContainerName string
	Status        string
	Path          string // human-readable: project/stage/service
	Key           string // unique selection key (container_name)
	UUID          string // Coolify resource UUID (application or service)
	Type          string // "application" | "service"
	IsHeader      bool
}

func FlattenProjects(ps []api.Project) []ServiceRow {
	var out []ServiceRow
	for _, p := range ps {
		header := p.Name
		if header == "" {
			header = "(unnamed)"
		}
		out = append(out, ServiceRow{Project: header, IsHeader: true})
		for _, st := range p.Stages {
			for _, s := range st.Services {
				if s.ContainerName == "" || s.ContainerName == "Not Found" {
					// No live container — skip; can't stream anyway.
					continue
				}
				path := p.Name + "/" + s.Name
				if st.Name != "" && st.Name != p.Name {
					path = p.Name + "/" + st.Name + "/" + s.Name
				}
				out = append(out, ServiceRow{
					Project: p.Name, Stage: st.Name, Service: s.Name,
					ContainerName: s.ContainerName, Status: s.Status,
					Path: path, Key: s.ContainerName,
					UUID: s.UUID, Type: s.Type,
				})
			}
		}
	}
	return out
}

// FlattenContainers is the fallback when /api/projects is empty.
func FlattenContainers(cs []api.Container) []ServiceRow {
	if len(cs) == 0 {
		return nil
	}
	out := []ServiceRow{{Project: "containers", IsHeader: true}}
	for _, c := range cs {
		name := c.Service
		if name == "" {
			name = c.App
		}
		if name == "" {
			name = c.Name
		}
		out = append(out, ServiceRow{
			Project: "containers", Service: name,
			ContainerName: c.Name, Status: c.Status,
			Path: "containers/" + name,
			Key:  c.Name,
		})
	}
	return out
}

func statusGlyph(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return "●"
	case "restarting":
		return "◐"
	case "unknown", "":
		return "○"
	case "exited", "failed", "dead", "stopped":
		return "●"
	}
	return "●"
}

func statusColor(th theme.Theme, status string) string {
	switch strings.ToLower(status) {
	case "running":
		return th.OK
	case "restarting":
		return th.Warn
	case "exited", "failed", "dead", "stopped":
		return th.Error
	}
	return th.Dim
}

func navFilterRow(th theme.Theme, query string, focused bool) []Segment {
	label := "FILTER "
	inputArea := NavW - 2 - runeLen(label)
	show := query
	color := th.Text
	if show == "" {
		show = "filter…"
		color = th.Dim
	}
	if focused {
		color = th.Accent
	}
	return []Segment{
		{Text: " ", FG: th.Text},
		{Text: label, FG: th.Muted, Bold: true},
		{Text: rpad(truncate(show, inputArea), inputArea), FG: color},
		{Text: " ", FG: th.Text},
	}
}

func navBlankRow(th theme.Theme) []Segment {
	return []Segment{{Text: repeat(" ", NavW), FG: th.Text}}
}

func navProjectRow(th theme.Theme, name string) []Segment {
	text := " ▾ " + name
	return []Segment{{Text: rpad(text, NavW), FG: th.Muted, Bold: true}}
}

func navServiceRow(th theme.Theme, row ServiceRow, selected, focused bool) []Segment {
	indent := "  "
	dot := statusGlyph(row.Status)
	nameCol := NavW - runeLen(indent) - 2 - 1
	nameTxt := truncate(row.Service, nameCol)
	if selected {
		bg := th.Border
		fg := th.Text
		if focused {
			bg = th.Accent
			fg = th.AccentFg
		}
		dotFg := statusColor(th, row.Status)
		if focused {
			dotFg = th.AccentFg
		}
		return []Segment{
			{Text: indent, FG: fg, BG: bg},
			{Text: dot, FG: dotFg, BG: bg},
			{Text: " ", FG: fg, BG: bg},
			{Text: rpad(nameTxt, nameCol), FG: fg, BG: bg, Bold: true},
			{Text: " ", FG: fg, BG: bg},
		}
	}
	return []Segment{
		{Text: indent, FG: th.Text},
		{Text: dot, FG: statusColor(th, row.Status)},
		{Text: " ", FG: th.Text},
		{Text: rpad(nameTxt, nameCol), FG: th.Text},
		{Text: " ", FG: th.Text},
	}
}

// navContent builds exactly `contentRows` rows of nav segments.
// `selectedKey` is the container_name (unique) of the currently-selected service.
func navContent(th theme.Theme, rows []ServiceRow, selectedKey, query string, focused bool, contentRows int) [][]Segment {
	out := [][]Segment{}
	out = append(out, navFilterRow(th, query, focused))
	out = append(out, navBlankRow(th))

	q := strings.ToLower(query)
	for _, r := range rows {
		if r.IsHeader {
			out = append(out, navProjectRow(th, r.Project))
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(r.Path), q) && !strings.Contains(strings.ToLower(r.Service), q) {
				continue
			}
		}
		out = append(out, navServiceRow(th, r, r.Key == selectedKey, focused))
	}
	for len(out) < contentRows {
		out = append(out, navBlankRow(th))
	}
	return out[:contentRows]
}
