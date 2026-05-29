package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// cmdAdmin dispatches `logify admin <subcommand>`.
func cmdAdmin(env *Env, argv []string) int {
	if len(argv) == 0 {
		return emitErr(ExitBadInput, CLIError{
			Code:    "BAD_INPUT",
			Message: "admin requires a subcommand",
			Hint:    "logify admin restart <svc> | redeploy <svc> [--force] | audit",
		})
	}
	sub, rest := argv[0], argv[1:]
	switch sub {
	case "restart":
		return cmdAdminRestart(env, rest)
	case "redeploy":
		return cmdAdminRedeploy(env, rest)
	case "audit":
		return cmdAdminAudit(env, rest)
	}
	return emitErr(ExitBadInput, CLIError{
		Code:    "BAD_INPUT",
		Message: "unknown admin subcommand: " + sub,
		Hint:    "restart | redeploy | audit",
	})
}

// ── restart ────────────────────────────────────────────────────────

func cmdAdminRestart(env *Env, argv []string) int {
	fs := flag.NewFlagSet("admin restart", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation prompt")
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
	if !confirmDestructive(*yes, "restart "+resolved.Path) {
		return ExitOK
	}

	ctx, cancel := httpCtx()
	defer cancel()
	res, err := env.Client.AdminRestart(ctx, resolved.UUID)
	if err != nil {
		_, body := translate(err)
		return emitErr(codeFor(body.Code), body)
	}

	if EmitJSON {
		_ = emitJSON(res)
	} else {
		fmt.Printf("✓ restarted %s (container %s)\n", resolved.Path, res.ContainerName)
		fmt.Printf("  follow:  logify logs %s --follow\n", resolved.Name)
	}
	return ExitOK
}

// ── redeploy ───────────────────────────────────────────────────────

func cmdAdminRedeploy(env *Env, argv []string) int {
	fs := flag.NewFlagSet("admin redeploy", flag.ContinueOnError)
	force := fs.Bool("force", false, "force-rebuild (skip image cache)")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
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
	verb := "redeploy"
	if *force {
		verb = "force-rebuild + redeploy"
	}
	if !confirmDestructive(*yes, verb+" "+resolved.Path) {
		return ExitOK
	}

	ctx, cancel := httpCtx()
	defer cancel()
	res, err := env.Client.AdminRedeploy(ctx, resolved.UUID, *force)
	if err != nil {
		_, body := translate(err)
		return emitErr(codeFor(body.Code), body)
	}

	if EmitJSON {
		_ = emitJSON(res)
	} else {
		fmt.Printf("✓ %s triggered for %s\n", verb, resolved.Path)
		fmt.Printf("  watch:   logify logs %s --build --follow\n", resolved.Name)
	}
	return ExitOK
}

// ── audit ──────────────────────────────────────────────────────────

func cmdAdminAudit(env *Env, argv []string) int {
	fs := flag.NewFlagSet("admin audit", flag.ContinueOnError)
	limit := fs.Int("limit", 50, "max rows")
	if _, err := parseIntermixed(fs, argv); err != nil {
		return ExitBadInput
	}
	ctx, cancel := httpCtx()
	defer cancel()
	rows, err := env.Client.AdminAudit(ctx, *limit)
	if err != nil {
		_, body := translate(err)
		return emitErr(codeFor(body.Code), body)
	}
	if EmitJSON {
		_ = emitJSON(rows)
		return ExitOK
	}
	if len(rows) == 0 {
		fmt.Println("(no admin actions recorded)")
		return ExitOK
	}
	const (
		colTS     = 20
		colKey    = 16
		colAction = 10
		colResult = 7
	)
	pad := func(s string, w int) string {
		if len(s) >= w {
			return s[:w-1] + "…"
		}
		return s + strings.Repeat(" ", w-len(s))
	}
	fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s  %s\n",
		pad("WHEN", colTS), pad("KEY", colKey), pad("ACTION", colAction),
		pad("RESULT", colResult), "TARGET", "DETAIL")
	fmt.Fprintln(os.Stdout, strings.Repeat("─", 100))
	for _, r := range rows {
		when := r.TS
		if t, err := time.Parse(time.RFC3339, strings.TrimSuffix(r.TS, "Z")+"Z"); err == nil {
			when = t.UTC().Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s  %s\n",
			pad(when, colTS),
			pad(r.KeyName+"("+r.KeyPrefix+")", colKey),
			pad(r.Action, colAction),
			pad(r.Result, colResult),
			r.Target, r.Detail)
	}
	return ExitOK
}

// confirmDestructive prompts for [y/N] on stderr; --yes flag skips it.
func confirmDestructive(skip bool, what string) bool {
	if skip {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s ? [y/N] ", what)
	var buf [1]byte
	_, err := os.Stdin.Read(buf[:])
	if err != nil {
		return false
	}
	c := buf[0]
	for c == ' ' {
		os.Stdin.Read(buf[:])
		c = buf[0]
	}
	return c == 'y' || c == 'Y'
}
