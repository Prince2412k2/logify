package theme

// Theme defines the color palette for the TUI.
// Mirrors reference/themes.jsx — 10 themes total.
type Theme struct {
	ID         string
	Name       string
	Subtitle   string
	Bg         string
	Panel      string
	Text       string
	Muted      string
	Dim        string
	Border     string
	BorderFoc  string
	Accent     string
	AccentFg   string
	MatchBg    string
	MatchFg    string
	Error      string
	Warn       string
	Info       string
	Debug      string
	OK         string
	BadgeProj  string
	BadgeSvc   string
	ErrRowBg   string
	WarnRowBg  string
}

var All = map[string]Theme{
	"amber": {
		ID: "amber", Name: "Amber Dark", Subtitle: "spec default · amber on near-black",
		Bg: "#0a0a0a", Panel: "#0a0a0a", Text: "#ededed", Muted: "#a3a3a3", Dim: "#6b6b6b",
		Border: "#3a3a3a", BorderFoc: "#f59e0b", Accent: "#f59e0b", AccentFg: "#0a0a0a",
		MatchBg: "#f59e0b", MatchFg: "#0a0a0a",
		Error: "#f43f5e", Warn: "#f59e0b", Info: "#38bdf8", Debug: "#6b6b6b", OK: "#22c55e",
		BadgeProj: "#818cf8", BadgeSvc: "#34d399",
		ErrRowBg: "#1a0d11", WarnRowBg: "#1a1408",
	},
	"tappin": {
		ID: "tappin", Name: "Tappin", Subtitle: "turkis on navy",
		Bg: "#00183E", Panel: "#001230", Text: "#e6edf7", Muted: "#9aa8c0", Dim: "#5d6c87",
		Border: "#1f3258", BorderFoc: "#00CCC2", Accent: "#00CCC2", AccentFg: "#00183E",
		MatchBg: "#00CCC2", MatchFg: "#00183E",
		Error: "#f87171", Warn: "#fbbf24", Info: "#38bdf8", Debug: "#5d6c87", OK: "#34d399",
		BadgeProj: "#a5b4fc", BadgeSvc: "#5eead4",
		ErrRowBg: "#2a1418", WarnRowBg: "#241a08",
	},
	"solarizedDark": {
		ID: "solarizedDark", Name: "Solarized Dark", Subtitle: "Ethan Schoonover, 2011",
		Bg: "#002b36", Panel: "#073642", Text: "#93a1a1", Muted: "#839496", Dim: "#586e75",
		Border: "#0e4451", BorderFoc: "#b58900", Accent: "#b58900", AccentFg: "#002b36",
		MatchBg: "#b58900", MatchFg: "#002b36",
		Error: "#dc322f", Warn: "#cb4b16", Info: "#268bd2", Debug: "#586e75", OK: "#859900",
		BadgeProj: "#6c71c4", BadgeSvc: "#2aa198",
		ErrRowBg: "#1f1414", WarnRowBg: "#1f1810",
	},
	"solarizedLight": {
		ID: "solarizedLight", Name: "Solarized Light", Subtitle: "cream + olive accent",
		Bg: "#fdf6e3", Panel: "#f5efdc", Text: "#586e75", Muted: "#657b83", Dim: "#93a1a1",
		Border: "#e3dcc4", BorderFoc: "#b58900", Accent: "#b58900", AccentFg: "#fdf6e3",
		MatchBg: "#b58900", MatchFg: "#fdf6e3",
		Error: "#dc322f", Warn: "#cb4b16", Info: "#268bd2", Debug: "#93a1a1", OK: "#859900",
		BadgeProj: "#6c71c4", BadgeSvc: "#2aa198",
		ErrRowBg: "#f4d8d4", WarnRowBg: "#f4dcc8",
	},
	"dracula": {
		ID: "dracula", Name: "Dracula", Subtitle: "purple on slate",
		Bg: "#282a36", Panel: "#21222c", Text: "#f8f8f2", Muted: "#bfbfd6", Dim: "#6272a4",
		Border: "#44475a", BorderFoc: "#bd93f9", Accent: "#bd93f9", AccentFg: "#282a36",
		MatchBg: "#f1fa8c", MatchFg: "#282a36",
		Error: "#ff5555", Warn: "#ffb86c", Info: "#8be9fd", Debug: "#6272a4", OK: "#50fa7b",
		BadgeProj: "#ff79c6", BadgeSvc: "#50fa7b",
		ErrRowBg: "#3a2128", WarnRowBg: "#3a2a1d",
	},
	"gruvbox": {
		ID: "gruvbox", Name: "Gruvbox Dark", Subtitle: "retro warm",
		Bg: "#282828", Panel: "#1d2021", Text: "#ebdbb2", Muted: "#a89984", Dim: "#7c6f64",
		Border: "#3c3836", BorderFoc: "#fabd2f", Accent: "#fabd2f", AccentFg: "#282828",
		MatchBg: "#fabd2f", MatchFg: "#282828",
		Error: "#fb4934", Warn: "#fe8019", Info: "#83a598", Debug: "#7c6f64", OK: "#b8bb26",
		BadgeProj: "#d3869b", BadgeSvc: "#8ec07c",
		ErrRowBg: "#3a1d18", WarnRowBg: "#3a261a",
	},
	"tokyo": {
		ID: "tokyo", Name: "Tokyo Night", Subtitle: "blue on indigo",
		Bg: "#1a1b26", Panel: "#16161e", Text: "#c0caf5", Muted: "#9aa5ce", Dim: "#565f89",
		Border: "#292e42", BorderFoc: "#7aa2f7", Accent: "#7aa2f7", AccentFg: "#1a1b26",
		MatchBg: "#e0af68", MatchFg: "#1a1b26",
		Error: "#f7768e", Warn: "#e0af68", Info: "#7dcfff", Debug: "#565f89", OK: "#9ece6a",
		BadgeProj: "#bb9af7", BadgeSvc: "#7dcfff",
		ErrRowBg: "#2a1a23", WarnRowBg: "#2a2218",
	},
	"nord": {
		ID: "nord", Name: "Nord", Subtitle: "arctic, blue-grey",
		Bg: "#2e3440", Panel: "#272b35", Text: "#d8dee9", Muted: "#aab3c4", Dim: "#4c566a",
		Border: "#3b4252", BorderFoc: "#88c0d0", Accent: "#88c0d0", AccentFg: "#2e3440",
		MatchBg: "#ebcb8b", MatchFg: "#2e3440",
		Error: "#bf616a", Warn: "#ebcb8b", Info: "#81a1c1", Debug: "#4c566a", OK: "#a3be8c",
		BadgeProj: "#b48ead", BadgeSvc: "#8fbcbb",
		ErrRowBg: "#3a2e35", WarnRowBg: "#3a3526",
	},
	"contrast": {
		ID: "contrast", Name: "High Contrast", Subtitle: "mono + single accent",
		Bg: "#000000", Panel: "#000000", Text: "#ffffff", Muted: "#cccccc", Dim: "#7a7a7a",
		Border: "#ffffff", BorderFoc: "#ffff00", Accent: "#ffff00", AccentFg: "#000000",
		MatchBg: "#ffff00", MatchFg: "#000000",
		Error: "#ff5d5d", Warn: "#ffff00", Info: "#5dd0ff", Debug: "#7a7a7a", OK: "#5dff8d",
		BadgeProj: "#ffffff", BadgeSvc: "#ffffff",
		ErrRowBg: "#2a0808", WarnRowBg: "#2a2a08",
	},
	"paper": {
		ID: "paper", Name: "Paper", Subtitle: "cream + ink (Tappin)",
		Bg: "#F8F4EC", Panel: "#EFE7D7", Text: "#0E1A1F", Muted: "#4a5760", Dim: "#8aa0aa",
		Border: "#d8d3c4", BorderFoc: "#00CCC2", Accent: "#00CCC2", AccentFg: "#0E1A1F",
		MatchBg: "#F2B544", MatchFg: "#0E1A1F",
		Error: "#D9442B", Warn: "#C77B2A", Info: "#2563eb", Debug: "#8aa0aa", OK: "#1F9D6E",
		BadgeProj: "#6366f1", BadgeSvc: "#0d9488",
		ErrRowBg: "#f0dcd6", WarnRowBg: "#f0e2d2",
	},
}

// Order in which themes appear in the picker.
var Order = []string{
	"amber", "tappin", "solarizedDark", "solarizedLight", "dracula",
	"gruvbox", "tokyo", "nord", "contrast", "paper",
}

func Get(id string) Theme {
	if t, ok := All[id]; ok {
		return t
	}
	return All["amber"]
}
