package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/princepatel/logify/internal/config"
)

// Version is set at build time via -ldflags.
var Version = "0.1.0"

// Run is the main entry point. argv excludes the program name. Returns
// the desired exit code.
type TUILauncher func(cfg config.Config, cfgPath string, mock bool, themeOverride string) int

// Run dispatches on argv. tuiLauncher is supplied by main so we don't take
// a hard dep on the bubble-tea TUI inside the CLI package.
func Run(argv []string, tuiLauncher TUILauncher) int {
	gf := newGlobalFlags()
	args, code := gf.parse(argv)
	if code != 0 {
		return code
	}
	cfg := gf.applyTo(loadConfig())
	EmitJSON = gf.json

	// `logify` (no sub-command) → manifest + context.
	if len(args) == 0 {
		return cmdManifest(cfg, gf)
	}

	sub, rest := args[0], args[1:]

	// Special-case tui — it's interactive, not JSON.
	if sub == "tui" {
		fs := flag.NewFlagSet("tui", flag.ContinueOnError)
		mock := fs.Bool("mock", false, "use mock data")
		themeOverride := fs.String("theme", "", "theme override")
		if err := fs.Parse(rest); err != nil {
			return ExitBadInput
		}
		return tuiLauncher(cfg, gf.cfgPath, *mock, *themeOverride)
	}

	env, err := NewEnv(cfg, gf.cfgPath)
	if err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}

	switch sub {
	case "version":
		if EmitJSON {
			_ = emitJSON(map[string]string{"version": Version})
		} else {
			fmt.Println("logify " + Version)
		}
		return ExitOK
	case "login":
		return cmdLogin(env, rest)
	case "logout":
		return cmdLogout(env)
	case "list":
		return cmdList(env, rest)
	case "logs":
		return cmdLogs(env, rest)
	case "bind":
		return cmdBind(env, rest)
	case "unbind":
		return removeBinding(env)
	case "admin":
		return cmdAdmin(env, rest)
	case "help", "-h", "--help":
		return cmdManifest(cfg, gf)
	}
	return emitErr(ExitBadInput, CLIError{
		Code: "BAD_INPUT", Message: fmt.Sprintf("unknown command: %s", sub),
		Hint: "run `logify` for the command manifest",
	})
}

// ─── global flags ──────────────────────────────────────────────────────

type globalFlags struct {
	url     string
	token   string
	cfgPath string
	json    bool
}

func newGlobalFlags() *globalFlags { return &globalFlags{} }

// parse splits argv into global-flag args and subcommand args. Global flags
// must appear before the subcommand.
func (g *globalFlags) parse(argv []string) (rest []string, code int) {
	i := 0
	for i < len(argv) {
		a := argv[i]
		if a == "--" {
			i++
			break
		}
		switch {
		case a == "--url":
			if i+1 >= len(argv) {
				return nil, errBadInput("--url requires a value")
			}
			g.url = argv[i+1]
			i += 2
		case len(a) > 6 && a[:6] == "--url=":
			g.url = a[6:]
			i++
		case a == "--token":
			if i+1 >= len(argv) {
				return nil, errBadInput("--token requires a value")
			}
			g.token = argv[i+1]
			i += 2
		case len(a) > 8 && a[:8] == "--token=":
			g.token = a[8:]
			i++
		case a == "--config":
			if i+1 >= len(argv) {
				return nil, errBadInput("--config requires a value")
			}
			g.cfgPath = argv[i+1]
			i += 2
		case len(a) > 9 && a[:9] == "--config=":
			g.cfgPath = a[9:]
			i++
		case a == "--json":
			g.json = true
			i++
		case a == "-h", a == "--help":
			return []string{"help"}, 0
		default:
			return argv[i:], 0
		}
	}
	return argv[i:], 0
}

func (g *globalFlags) applyTo(cfg config.Config) config.Config {
	if g.url != "" {
		cfg.URL = g.url
	}
	if g.token != "" {
		cfg.Token = g.token
	}
	if v := os.Getenv("LOGIFY_URL"); v != "" && g.url == "" {
		cfg.URL = v
	}
	if v := os.Getenv("LOGIFY_TOKEN"); v != "" && g.token == "" {
		cfg.Token = v
	}
	return cfg
}

func loadConfig() config.Config {
	c, _, _ := config.Load()
	return c
}

// parseIntermixed parses argv against fs, allowing flags to appear anywhere
// (before or after positional arguments). Returns all non-flag args in order.
func parseIntermixed(fs *flag.FlagSet, argv []string) ([]string, error) {
	var positional []string
	args := argv
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return positional, nil
}

func cmdManifest(cfg config.Config, gf *globalFlags) int {
	env, _ := NewEnv(cfg, gf.cfgPath)
	resp := ManifestResponse{Version: Version, Commands: Manifest}
	if env != nil && env.BindingPath != "" {
		resp.Context.ConfigPath = env.BindingPath
		if env.Bindings != nil && env.Bindings.IsBound() {
			resp.Context.Bound = true
			resp.Context.Project = env.Bindings.Project
			resp.Context.ProjectID = env.Bindings.ProjectID
		}
	}
	if EmitJSON {
		_ = emitJSON(resp)
		return ExitOK
	}
	printManifestText(resp)
	return ExitOK
}

func printManifestText(r ManifestResponse) {
	w := os.Stdout
	fmt.Fprintf(w, "logify %s\n\n", r.Version)
	if r.Context.Bound {
		fmt.Fprintf(w, "bound:   %s (id=%s)\n", r.Context.Project, r.Context.ProjectID)
		fmt.Fprintf(w, "config:  %s\n", r.Context.ConfigPath)
	} else {
		fmt.Fprintln(w, "bound:   (none — run `logify bind <project>` first, or pass --all to `list`)")
		if r.Context.ConfigPath != "" {
			fmt.Fprintf(w, "config:  %s\n", r.Context.ConfigPath)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	for _, c := range r.Commands {
		args := ""
		if len(c.Args) > 0 {
			args = " " + strings.Join(c.Args, " ")
		}
		head := "  logify " + c.Name + args
		fmt.Fprintln(w, head)
		if c.Description != "" {
			fmt.Fprintln(w, "      "+c.Description)
		}
		if len(c.Flags) > 0 {
			parts := make([]string, 0, len(c.Flags))
			for _, f := range c.Flags {
				parts = append(parts, "--"+f.Name)
			}
			fmt.Fprintln(w, "      flags: "+strings.Join(parts, " "))
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "global flags: --url --token --config --json")
	fmt.Fprintln(w, "add --json (before the subcommand) for structured output.")
}
