package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/config"
	"github.com/princepatel/logify/internal/theme"
)

// cmdLogin runs a styled interactive flow that asks for a gateway URL + API
// key, verifies them against /api/projects, and persists to the config file.
// Visuals match the `bind` picker: foreground-only theme colors, stderr,
// host terminal background untouched.
func cmdLogin(env *Env, argv []string) int {
	_ = argv

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return emitErr(ExitBadInput, CLIError{
			Code:    "BAD_INPUT",
			Message: "login requires an interactive terminal",
			Hint:    "set LOGIFY_URL and LOGIFY_TOKEN env vars, or use --url/--token flags",
		})
	}

	th := theme.Get(env.Cfg.Theme)
	fg := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	accent := fg(th.Accent).Bold(true)
	muted := fg(th.Muted)
	dim := fg(th.Dim)
	text := fg(th.Text)
	okSty := fg(th.OK).Bold(true)
	warnSty := fg(th.Warn).Bold(true)
	errSty := fg(th.Error).Bold(true)

	// ── header ──
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, accent.Render("▸ logify · login"))
	fmt.Fprintln(os.Stderr, dim.Render("  enter accepts default · ctrl-c cancels"))
	fmt.Fprintln(os.Stderr)

	reader := bufio.NewReader(os.Stdin)

	// ── URL ──
	defaultURL := env.Cfg.URL
	if defaultURL == "" {
		defaultURL = "http://localhost:8089"
	}
	fmt.Fprintf(os.Stderr, "  %s %s %s ",
		muted.Render("gateway url"),
		dim.Render("["+defaultURL+"]"),
		accent.Render("▸"))
	urlLine, err := reader.ReadString('\n')
	if err != nil && urlLine == "" {
		return ExitBadInput
	}
	urlLine = strings.TrimSpace(urlLine)
	if urlLine == "" {
		urlLine = defaultURL
	}
	urlLine = strings.TrimRight(urlLine, "/")

	// ── token (no echo) ──
	fmt.Fprintf(os.Stderr, "  %s              %s ",
		muted.Render("api key"),
		accent.Render("▸"))
	tokBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}
	token := strings.TrimSpace(string(tokBytes))
	if token == "" {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, errSty.Render("  ✕ api key cannot be empty"))
		return ExitBadInput
	}

	// ── verify ──
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  %s %s ",
		warnSty.Render("●"),
		muted.Render("verifying…"))
	client := api.New(urlLine, token)
	ctx, cancel := httpCtx()
	defer cancel()
	projects, err := client.Projects(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, errSty.Render("failed"))
		_, body := translate(err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  "+errSty.Render("✕ "+body.Code+" — "+body.Message))
		if body.Hint != "" {
			fmt.Fprintln(os.Stderr, "  "+dim.Render("hint: "+body.Hint))
		}
		return codeFor(body.Code)
	}
	fmt.Fprintln(os.Stderr, okSty.Render("ok"))

	// ── save ──
	cfg := env.Cfg
	cfg.URL = urlLine
	cfg.Token = token
	path, err := config.Save(cfg)
	if err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}

	if EmitJSON {
		_ = emitJSON(map[string]any{
			"url":              urlLine,
			"config_path":      path,
			"projects_visible": len(projects),
		})
		return ExitOK
	}

	// ── success block ──
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  %s %s\n", okSty.Render("✓"), text.Render("logged in to "+urlLine))
	fmt.Fprintf(os.Stderr, "  %s %s\n", dim.Render("saved to"), text.Render(path))
	if len(projects) == 0 {
		fmt.Fprintf(os.Stderr, "  %s %s\n",
			warnSty.Render("⚠"),
			muted.Render("no projects accessible — ask an admin to grant your key project access"))
	} else {
		s := "s"
		if len(projects) == 1 {
			s = ""
		}
		fmt.Fprintf(os.Stderr, "  %s %s\n",
			accent.Render("●"),
			muted.Render(fmt.Sprintf("%d project%s accessible", len(projects), s)))
	}
	fmt.Fprintln(os.Stderr)
	return ExitOK
}

// cmdLogout clears the saved token (but keeps url + theme).
func cmdLogout(env *Env) int {
	cfg := env.Cfg
	if cfg.Token == "" {
		return emitErr(ExitNotFound, CLIError{Code: "NOT_FOUND", Message: "no token saved"})
	}
	cfg.Token = ""
	path, err := config.Save(cfg)
	if err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}
	if EmitJSON {
		_ = emitJSON(map[string]any{"logged_out": true, "config_path": path})
		return ExitOK
	}
	fmt.Printf("logged out — token removed from %s\n", path)
	return ExitOK
}
