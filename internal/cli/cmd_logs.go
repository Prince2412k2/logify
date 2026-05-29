package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func cmdLogs(env *Env, argv []string) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	tail := fs.Int("tail", 200, "snapshot line count (runtime only)")
	level := fs.String("level", "", "csv: err,warn,info,debug (runtime only)")
	grep := fs.String("grep", "", "substring filter")
	follow := fs.Bool("follow", false, "stream until --max or terminal state")
	maxDur := fs.Duration("max", 0, "bound --follow (e.g. 60s)")
	build := fs.Bool("build", false, "show build (deployment) logs instead of runtime")
	deployment := fs.String("deployment", "", "specific deployment uuid for --build (default: latest)")
	rest, err := parseIntermixed(fs, argv)
	if err != nil {
		return ExitBadInput
	}
	arg := ""
	if len(rest) > 0 {
		arg = rest[0]
	}

	resolved, errx := Resolve(env, arg)
	if errx != nil {
		return emitErr(codeFor(errx.Code), *errx)
	}

	if *build {
		return doBuildLogs(env, resolved, *follow, *maxDur, *deployment, *grep, EmitJSON)
	}
	return doRuntimeLogs(env, resolved, *tail, *level, *grep, *follow, EmitJSON, *maxDur)
}

// ── runtime logs ───────────────────────────────────────────────────────

func doRuntimeLogs(env *Env, r *ResolvedService, tail int, levelCSV, grep string, follow, jsonOut bool, maxDur time.Duration) int {
	if r.ContainerName == "" {
		return emitErr(ExitNotFound, CLIError{Code: "NOT_FOUND", Message: "no container associated with service"})
	}
	wantLevels := parseLevelSet(levelCSV)

	if !follow {
		lines, err := snapshotLogs(env, r.ContainerName, tail)
		if err != nil {
			_, body := translate(err)
			return emitErr(codeFor(body.Code), body)
		}
		filtered := filterLines(lines, wantLevels, grep)
		if jsonOut {
			_ = emitJSON(filtered)
		} else {
			for _, l := range filtered {
				fmt.Fprintln(os.Stdout, formatLogLine(l))
			}
		}
		return ExitOK
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if maxDur > 0 {
		time.AfterFunc(maxDur, cancel)
	}
	out := make(chan string, 256)
	errCh := make(chan error, 1)
	go env.Client.StreamLogs(ctx, r.ContainerName, tail, out, errCh)
	for {
		select {
		case <-ctx.Done():
			return ExitOK
		case line, ok := <-out:
			if !ok {
				return ExitOK
			}
			le := parseLogLine(line)
			if !levelMatches(wantLevels, le.Level) {
				continue
			}
			if grep != "" && !strings.Contains(strings.ToLower(le.Msg), strings.ToLower(grep)) {
				continue
			}
			if jsonOut {
				_ = emitNDJSON(os.Stdout, le)
			} else {
				fmt.Fprintln(os.Stdout, formatLogLine(le))
			}
		case e := <-errCh:
			if e != nil {
				_, body := translate(e)
				return emitErr(codeFor(body.Code), body)
			}
			return ExitOK
		}
	}
}

// formatLogLine renders one log entry in the canonical text shape:
//   HH:MM:SS LEVEL message
// Level is fixed-width 5 to keep columns aligned.
func formatLogLine(l LogEntry) string {
	return l.TS + " " + padLevel(l.Level) + " " + l.Msg
}

func padLevel(s string) string {
	const w = 5
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

// ── build logs ─────────────────────────────────────────────────────────

// BuildSnapshot is the response shape for `logify logs --build`.
type BuildSnapshot struct {
	Deployment      string   `json:"deployment"`
	Status          string   `json:"status"`
	Commit          string   `json:"commit,omitempty"`
	CommitMessage   string   `json:"commit_message,omitempty"`
	StartedAt       string   `json:"started_at,omitempty"`
	StartedAtUnix   int64    `json:"started_at_unix,omitempty"`
	FinishedAt      string   `json:"finished_at,omitempty"`
	FinishedAtUnix  int64    `json:"finished_at_unix,omitempty"`
	DurationSeconds int64    `json:"duration_seconds"`
	Lines           []string `json:"lines"`
}

func doBuildLogs(env *Env, r *ResolvedService, follow bool, maxDur time.Duration, deploymentUUID, grep string, jsonOut bool) int {
	if r.Type == "service" {
		if jsonOut {
			_ = emitJSON(BuildSnapshot{Lines: []string{}})
		} else {
			fmt.Fprintln(os.Stderr, "(services don't have build logs)")
		}
		return ExitOK
	}
	if deploymentUUID != "" {
		return emitErr(ExitBadInput, CLIError{Code: "BAD_INPUT",
			Message: "--deployment selector is not yet supported by the gateway"})
	}

	if !follow {
		ctx, cancel := httpCtx()
		defer cancel()
		bl, err := env.Client.BuildLog(ctx, r.UUID)
		if err != nil {
			_, body := translate(err)
			return emitErr(codeFor(body.Code), body)
		}
		snap := buildSnapshotFromAPI(bl)
		if grep != "" {
			snap.Lines = filterRaw(snap.Lines, grep)
		}
		if jsonOut {
			_ = emitJSON(snap)
		} else {
			printBuildSnapshotText(snap)
		}
		return ExitOK
	}

	// Follow: poll every 2s. Emit deploy.status events on transition + each
	// new build line as NDJSON. Exit on terminal status or --max.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if maxDur > 0 {
		time.AfterFunc(maxDur, cancel)
	}

	var (
		lastDeployment string
		lastStatus     string
		emittedLines   int
	)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		callCtx, callCancel := context.WithTimeout(ctx, 8*time.Second)
		bl, err := env.Client.BuildLog(callCtx, r.UUID)
		callCancel()
		if err != nil {
			_, body := translate(err)
			return emitErr(codeFor(body.Code), body)
		}
		if bl.DeploymentUUID != lastDeployment {
			lastDeployment = bl.DeploymentUUID
			lastStatus = ""
			emittedLines = 0
		}
		if bl.Status != lastStatus {
			if jsonOut {
				_ = emitNDJSON(os.Stdout, map[string]any{
					"event": "deploy." + bl.Status, "deployment": bl.DeploymentUUID,
					"status": bl.Status, "commit": bl.Commit,
				})
			} else {
				fmt.Fprintf(os.Stdout, "[deploy.%s] %s\n", bl.Status, bl.DeploymentUUID[:min(12, len(bl.DeploymentUUID))])
			}
			lastStatus = bl.Status
		}
		for _, ln := range bl.Lines[emittedLines:] {
			if grep != "" && !strings.Contains(strings.ToLower(ln), strings.ToLower(grep)) {
				continue
			}
			if jsonOut {
				_ = emitNDJSON(os.Stdout, map[string]any{
					"event": "build.line", "deployment": bl.DeploymentUUID, "line": ln,
				})
			} else {
				fmt.Fprintln(os.Stdout, ln)
			}
		}
		emittedLines = len(bl.Lines)
		if isTerminalStatus(bl.Status) {
			return ExitOK
		}
		select {
		case <-ctx.Done():
			return ExitOK
		case <-tick.C:
		}
	}
}

func filterRaw(lines []string, grep string) []string {
	if grep == "" {
		return lines
	}
	q := strings.ToLower(grep)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), q) {
			out = append(out, l)
		}
	}
	return out
}

func isTerminalStatus(s string) bool {
	switch s {
	case "finished", "failed", "cancelled-by-user", "cancelled":
		return true
	}
	return false
}

// printBuildSnapshotText emits the canonical text rendering of a build:
// a small header, then each build line raw.
func printBuildSnapshotText(s BuildSnapshot) {
	w := os.Stdout
	if s.Deployment != "" {
		fmt.Fprintf(w, "deployment: %s\n", s.Deployment)
	}
	if s.Status != "" {
		fmt.Fprintf(w, "status:     %s\n", s.Status)
	}
	if s.Commit != "" {
		if s.CommitMessage != "" {
			fmt.Fprintf(w, "commit:     %s — %s\n", s.Commit, s.CommitMessage)
		} else {
			fmt.Fprintf(w, "commit:     %s\n", s.Commit)
		}
	}
	if s.StartedAt != "" {
		fmt.Fprintf(w, "started:    %s\n", s.StartedAt)
	}
	if s.DurationSeconds > 0 {
		fmt.Fprintf(w, "duration:   %ds\n", s.DurationSeconds)
	}
	if len(s.Lines) > 0 {
		fmt.Fprintln(w)
		for _, ln := range s.Lines {
			fmt.Fprintln(w, ln)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildSnapshotFromAPI(bl apiBuildLog) BuildSnapshot {
	s := BuildSnapshot{
		Deployment: bl.DeploymentUUID, Status: bl.Status, Commit: bl.Commit,
		CommitMessage: bl.CommitMessage, Lines: bl.Lines,
		StartedAtUnix: bl.CreatedAt, FinishedAtUnix: bl.FinishedAt,
	}
	if bl.CreatedAt > 0 {
		s.StartedAt = time.Unix(bl.CreatedAt, 0).UTC().Format(time.RFC3339)
	}
	if bl.FinishedAt > 0 {
		s.FinishedAt = time.Unix(bl.FinishedAt, 0).UTC().Format(time.RFC3339)
		if bl.FinishedAt > bl.CreatedAt {
			s.DurationSeconds = bl.FinishedAt - bl.CreatedAt
		}
	}
	return s
}

type apiBuildLog = struct {
	DeploymentUUID string   `json:"deployment_uuid"`
	Status         string   `json:"status"`
	Commit         string   `json:"commit"`
	CommitMessage  string   `json:"commit_message"`
	CreatedAt      int64    `json:"created_at"`
	FinishedAt     int64    `json:"finished_at"`
	Lines          []string `json:"lines"`
}

// ── snapshot helpers (unchanged from previous draft) ──────────────────

func snapshotLogs(env *Env, container string, tail int) ([]LogEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out := make(chan string, 1024)
	errCh := make(chan error, 1)
	go env.Client.StreamLogs(ctx, container, tail, out, errCh)
	var collected []LogEntry
	quiet := time.NewTimer(2 * time.Second)
	defer quiet.Stop()
	receivedAny := false
	for {
		select {
		case <-ctx.Done():
			return collected, nil
		case <-quiet.C:
			cancel()
			return collected, nil
		case line, ok := <-out:
			if !ok {
				return collected, nil
			}
			collected = append(collected, parseLogLine(line))
			receivedAny = true
			if !quiet.Stop() {
				select {
				case <-quiet.C:
				default:
				}
			}
			if receivedAny {
				quiet.Reset(300 * time.Millisecond)
			}
		case err := <-errCh:
			if err != nil {
				return collected, err
			}
			return collected, nil
		}
	}
}

// LogEntry is one structured log line.
type LogEntry struct {
	TS    string `json:"ts"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

func parseLogLine(raw string) LogEntry {
	s := strings.TrimRight(raw, "\r\n")
	ts := time.Now().UTC().Format("15:04:05")
	if len(s) >= 8 && isTimeStamp(s[:8]) {
		ts = s[:8]
		s = strings.TrimLeft(s[8:], " \t")
	}
	level := "INFO"
	up := strings.ToUpper(s)
	switch {
	case strings.HasPrefix(up, "ERROR") || strings.Contains(up, "ERROR") || strings.Contains(up, "FATAL"):
		level = "ERROR"
	case strings.HasPrefix(up, "WARN") || strings.HasPrefix(up, "WARNING"):
		level = "WARN"
	case strings.HasPrefix(up, "DEBUG") || strings.HasPrefix(up, "DBG"):
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
	return LogEntry{TS: ts, Level: level, Msg: s}
}

func isTimeStamp(s string) bool {
	if len(s) != 8 || s[2] != ':' || s[5] != ':' {
		return false
	}
	for _, i := range []int{0, 1, 3, 4, 6, 7} {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseLevelSet(csv string) map[string]bool {
	if csv == "" {
		return nil
	}
	out := map[string]bool{}
	for _, s := range strings.Split(csv, ",") {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "err", "error":
			out["ERROR"] = true
		case "warn", "warning":
			out["WARN"] = true
		case "info":
			out["INFO"] = true
		case "debug", "dbg":
			out["DEBUG"] = true
		}
	}
	return out
}

func levelMatches(want map[string]bool, lvl string) bool {
	if want == nil {
		return true
	}
	return want[lvl]
}

func filterLines(lines []LogEntry, want map[string]bool, grep string) []LogEntry {
	if want == nil && grep == "" {
		return lines
	}
	gr := strings.ToLower(grep)
	out := make([]LogEntry, 0, len(lines))
	for _, l := range lines {
		if !levelMatches(want, l.Level) {
			continue
		}
		if gr != "" && !strings.Contains(strings.ToLower(l.Msg), gr) {
			continue
		}
		out = append(out, l)
	}
	return out
}
