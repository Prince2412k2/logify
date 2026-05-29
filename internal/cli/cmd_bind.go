package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/princepatel/logify/internal/binding"
	"github.com/princepatel/logify/internal/picker"
	"github.com/princepatel/logify/internal/theme"
)

// cmdBind implements `logify bind [project]` and `logify bind --remove`.
func cmdBind(env *Env, argv []string) int {
	fs := flag.NewFlagSet("bind", flag.ContinueOnError)
	remove := fs.Bool("remove", false, "remove the project binding")
	list := fs.Bool("list", false, "print current binding as JSON")
	rest, err := parseIntermixed(fs, argv)
	if err != nil {
		return ExitBadInput
	}

	if *list {
		return emitBinding(env)
	}
	if *remove {
		return removeBinding(env)
	}
	if len(rest) == 0 {
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return emitErr(ExitBadInput, CLIError{
				Code:    "BAD_INPUT",
				Message: "bind requires a project name (or --remove / --list)",
				Hint:    "logify bind <project> | run interactively to pick from a list",
			})
		}
		return interactiveBind(env)
	}
	if len(rest) != 1 {
		return emitErr(ExitBadInput, CLIError{
			Code: "BAD_INPUT", Message: "bind takes at most one positional argument",
		})
	}
	return setBindingByName(env, rest[0])
}

func emitBinding(env *Env) int {
	if EmitJSON {
		out := map[string]any{"config_path": env.BindingPath}
		if env.Bindings != nil && env.Bindings.IsBound() {
			out["project"] = env.Bindings.Project
			out["project_id"] = env.Bindings.ProjectID
		}
		_ = emitJSON(out)
		return ExitOK
	}
	if env.Bindings == nil || !env.Bindings.IsBound() {
		fmt.Println("(not bound)")
		return ExitOK
	}
	fmt.Printf("project: %s\nid:      %s\nconfig:  %s\n",
		env.Bindings.Project, env.Bindings.ProjectID, env.BindingPath)
	return ExitOK
}

func removeBinding(env *Env) int {
	if env.BindingPath == "" || env.Bindings == nil || !env.Bindings.IsBound() {
		return emitErr(ExitNotFound, CLIError{Code: "NOT_FOUND", Message: "no binding configured"})
	}
	if err := os.Remove(env.BindingPath); err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}
	if EmitJSON {
		_ = emitJSON(map[string]any{"removed": true, "config_path": env.BindingPath})
	} else {
		fmt.Printf("removed binding (%s)\n", env.BindingPath)
	}
	return ExitOK
}

// setBindingByName resolves a project name (case-insensitive) and writes .logify.
func setBindingByName(env *Env, name string) int {
	projects, errx := fetchProjects(env)
	if errx != nil {
		return emitErr(codeFor(errx.Code), *errx)
	}

	// Exact, case-insensitive name match.
	var match *projectChoice
	candidates := []ErrCandidate{}
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) || p.ID == name {
			c := projectChoice{Name: p.Name, ID: p.ID}
			match = &c
			break
		}
		candidates = append(candidates, ErrCandidate{Path: p.Name, UUID: p.ID})
	}
	if match == nil {
		return emitErr(ExitNotFound, CLIError{
			Code: "NOT_FOUND", Message: "no project matches '" + name + "'",
			Hint:       "run `logify list` for project names",
			Candidates: candidates,
		})
	}

	return writeBinding(env, *match)
}

type projectChoice struct {
	Name string
	ID   string
}

func writeBinding(env *Env, choice projectChoice) int {
	target := env.BindingPath
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
		}
		target = filepath.Join(cwd, binding.FileName)
	}
	f := &binding.File{Project: choice.Name, ProjectID: choice.ID}
	if err := binding.Save(target, f); err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}
	if EmitJSON {
		_ = emitJSON(map[string]any{
			"config_path": target,
			"project":     choice.Name,
			"project_id":  choice.ID,
		})
	} else {
		fmt.Printf("bound to %s (id=%s)\nsaved to %s\n", choice.Name, choice.ID, target)
	}
	return ExitOK
}

// interactiveBind runs the inline gum-style picker (no alt-screen, theme
// colors, host terminal background preserved).
func interactiveBind(env *Env) int {
	projects, errx := fetchProjects(env)
	if errx != nil {
		return emitErr(codeFor(errx.Code), *errx)
	}
	if len(projects) == 0 {
		return emitErr(ExitNotFound, CLIError{Code: "NOT_FOUND",
			Message: "no projects accessible to this API key"})
	}

	items := make([]picker.Item, 0, len(projects))
	for _, p := range projects {
		items = append(items, picker.Item{
			Label: p.Name,
			Value: projectChoice{Name: p.Name, ID: p.ID},
		})
	}
	res, err := picker.Pick(items, "Pick a project:", theme.Get(env.Cfg.Theme))
	if err != nil {
		return emitErr(ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()})
	}
	if !res.Picked {
		_ = emitJSON(map[string]any{"cancelled": true})
		return ExitOK
	}
	choice, _ := res.Item.Value.(projectChoice)
	return writeBinding(env, choice)
}

// silence unused-import noise if strings becomes unused later.
var _ = strings.TrimSpace
