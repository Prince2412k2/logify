package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ListEntry is one row of `logify list`.
type ListEntry struct {
	Name             string             `json:"name"`
	UUID             string             `json:"uuid"`
	Project          string             `json:"project"`
	Path             string             `json:"path"`
	Type             string             `json:"type"`
	ContainerName    string             `json:"container_name"`
	Health           string             `json:"health,omitempty"`
	LatestDeployment *DeploymentSummary `json:"latest_deployment,omitempty"`
	DeployFetchErr   string             `json:"deploy_fetch_error,omitempty"`
}

// DeploymentSummary mirrors a row of /api/services/{uuid}/deployments
// (kept here so list.go doesn't depend on show.go).
type DeploymentSummary struct {
	UUID            string `json:"uuid"`
	Status          string `json:"status"`
	Commit          string `json:"commit,omitempty"`
	CommitMessage   string `json:"commit_message,omitempty"`
	Trigger         string `json:"trigger,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	StartedAtUnix   int64  `json:"started_at_unix,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	FinishedAtUnix  int64  `json:"finished_at_unix,omitempty"`
	DurationSeconds int64  `json:"duration_seconds"`
}

func cmdList(env *Env, argv []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	all := fs.Bool("all", false, "list every accessible service, not just the bound project")
	if _, err := parseIntermixed(fs, argv); err != nil {
		return ExitBadInput
	}

	projects, errx := fetchProjects(env)
	if errx != nil {
		return emitErr(codeFor(errx.Code), *errx)
	}

	var pool []ResolvedService
	if !*all && env.Bindings != nil && env.Bindings.IsBound() {
		pool = projectServices(projects, env.Bindings.Project, env.Bindings.ProjectID)
		if len(pool) == 0 {
			// Bound project not visible; surface that explicitly.
			return emitErr(ExitNotFound, CLIError{
				Code:    "NOT_FOUND",
				Message: "project '" + env.Bindings.Project + "' has no accessible services",
				Hint:    "run `logify bind --remove` or `logify bind <project>` to fix",
			})
		}
	} else {
		pool = allServices(projects)
	}

	entries := make([]ListEntry, len(pool))
	for i, s := range pool {
		entries[i] = ListEntry{
			Name: s.Name, UUID: s.UUID, Project: s.Project,
			Path: s.Path, Type: s.Type, ContainerName: s.ContainerName,
		}
	}

	// Parallel: fetch latest deployment per application. Skip services.
	var wg sync.WaitGroup
	for i := range entries {
		i := i
		if entries[i].Type != "application" || entries[i].UUID == "" {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := httpCtx()
			defer cancel()
			ds, err := env.Client.Deployments(ctx, entries[i].UUID)
			if err != nil {
				entries[i].DeployFetchErr = err.Error()
				return
			}
			if len(ds) > 0 {
				sum := depSummaryFromAPI(ds[0])
				entries[i].LatestDeployment = &sum
			}
		}()
	}
	wg.Wait()

	// Health: derived from the runtime container status. We piggyback on
	// /api/containers, fetched in parallel.
	hctx, hcancel := httpCtx()
	defer hcancel()
	if cs, err := env.Client.Containers(hctx); err == nil {
		statusByName := make(map[string]string, len(cs))
		for _, c := range cs {
			statusByName[c.Name] = c.Status
		}
		for i := range entries {
			if h := statusByName[entries[i].ContainerName]; h != "" {
				entries[i].Health = h
			}
		}
	}

	if EmitJSON {
		_ = emitJSON(entries)
	} else {
		printListText(entries)
	}
	return ExitOK
}

func printListText(entries []ListEntry) {
	if len(entries) == 0 {
		fmt.Println("(no services)")
		return
	}
	// Column widths sized to content.
	const (
		colName    = 22
		colType    = 13
		colHealth  = 18
		colDeploy  = 12
		colAge     = 10
		colCommit  = 9
	)
	pad := func(s string, w int) string {
		if len(s) >= w {
			return s[:w-1] + "…"
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s  %s  %s\n",
		pad("NAME", colName), pad("TYPE", colType),
		pad("HEALTH", colHealth), pad("DEPLOY", colDeploy),
		pad("AGE", colAge), pad("COMMIT", colCommit), "MESSAGE")
	fmt.Fprintln(os.Stdout, strings.Repeat("─", colName+colType+colHealth+colDeploy+colAge+colCommit+24))

	for _, e := range entries {
		health := e.Health
		if health == "" {
			health = "unreachable"
		}
		depStatus := "—"
		age := "—"
		commit := "—"
		msg := ""
		if e.LatestDeployment != nil {
			d := e.LatestDeployment
			depStatus = shortDeployStatus(d.Status)
			if d.StartedAtUnix > 0 {
				age = humanAgo(time.Unix(d.StartedAtUnix, 0))
			}
			if d.Commit != "" {
				commit = d.Commit
			}
			msg = d.CommitMessage
		}
		fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s  %s  %s\n",
			pad(e.Name, colName), pad(e.Type, colType),
			pad(health, colHealth), pad(depStatus, colDeploy),
			pad(age, colAge), pad(commit, colCommit), msg)
	}
}

// shortDeployStatus normalises Coolify's status strings for display.
// JSON output keeps the raw value; only the text renderer shortens.
func shortDeployStatus(s string) string {
	switch s {
	case "cancelled-by-user":
		return "cancelled"
	case "in_progress":
		return "running"
	}
	return s
}

func humanAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 0:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// depSummaryFromAPI converts an api.Deployment into the JSON shape we emit.
func depSummaryFromAPI(d apiDeployment) DeploymentSummary {
	s := DeploymentSummary{
		UUID: d.UUID, Status: d.Status, Commit: d.Commit, CommitMessage: d.CommitMessage,
		Trigger: d.Trigger, StartedAtUnix: d.CreatedAt, FinishedAtUnix: d.FinishedAt,
		DurationSeconds: d.DurationSeconds,
	}
	if d.CreatedAt > 0 {
		s.StartedAt = time.Unix(d.CreatedAt, 0).UTC().Format(time.RFC3339)
	}
	if d.FinishedAt > 0 {
		s.FinishedAt = time.Unix(d.FinishedAt, 0).UTC().Format(time.RFC3339)
	}
	return s
}

// apiDeployment is a structural shadow of api.Deployment so this file does
// not import the api package directly (kept tidy for future swap-outs).
type apiDeployment = struct {
	UUID            string `json:"uuid"`
	Status          string `json:"status"`
	Commit          string `json:"commit"`
	FullCommit      string `json:"full_commit"`
	CommitMessage   string `json:"commit_message"`
	PRID            string `json:"pr_id"`
	Trigger         string `json:"trigger"`
	ForceRebuild    bool   `json:"force_rebuild"`
	CreatedAt       int64  `json:"created_at"`
	FinishedAt      int64  `json:"finished_at"`
	DurationSeconds int64  `json:"duration_seconds"`
}
