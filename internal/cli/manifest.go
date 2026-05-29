package cli

type commandDef struct {
	Name        string    `json:"name"`
	Args        []string  `json:"args"`
	Flags       []flagDef `json:"flags"`
	Description string    `json:"description"`
}

type flagDef struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
	Desc    string `json:"description,omitempty"`
}

// Manifest is the static command catalogue agents introspect via `logify`.
var Manifest = []commandDef{
	{Name: "login", Description: "Interactive: prompts for gateway URL and API key, verifies them, saves to config."},
	{Name: "logout", Description: "Clear the saved API token from config."},
	{Name: "list", Description: "List services in the bound project (or every accessible service with --all). Each row carries runtime health and the latest deployment status.",
		Flags: []flagDef{
			{Name: "all", Type: "bool", Desc: "list every accessible service, not just the bound project"},
		}},
	{Name: "logs", Args: []string{"service"}, Description: "Runtime or build logs for one service. Snapshot by default; --follow streams NDJSON.",
		Flags: []flagDef{
			{Name: "build", Type: "bool", Desc: "show build (deployment) logs instead of runtime"},
			{Name: "deployment", Type: "string", Desc: "specific deployment uuid (with --build)"},
			{Name: "tail", Type: "int", Default: 200, Desc: "runtime snapshot line count"},
			{Name: "level", Type: "csv", Desc: "runtime filter: err,warn,info,debug"},
			{Name: "grep", Type: "string", Desc: "substring filter"},
			{Name: "follow", Type: "bool", Desc: "stream NDJSON; for --build, poll until terminal status"},
			{Name: "max", Type: "duration", Desc: "bound --follow (e.g. 60s)"},
		}},
	{Name: "bind", Args: []string{"project?"}, Description: "Bind the working directory to a Coolify project. With no arg on a TTY, an interactive numbered prompt runs.",
		Flags: []flagDef{
			{Name: "remove", Type: "bool", Desc: "remove the binding"},
			{Name: "list", Type: "bool", Desc: "print current binding as JSON"},
		}},
	{Name: "unbind", Description: "Alias for `bind --remove`."},
}

// BoundContext is the agent-friendly summary of the .logify state.
type BoundContext struct {
	ConfigPath string `json:"config_path"`
	Project    string `json:"project,omitempty"`
	ProjectID  string `json:"project_id,omitempty"`
	Bound      bool   `json:"bound"`
}

// ManifestResponse is the top-level shape of `logify` (no args).
type ManifestResponse struct {
	Version  string       `json:"version"`
	Context  BoundContext `json:"context"`
	Commands []commandDef `json:"commands"`
}
