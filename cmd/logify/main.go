package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/princepatel/logify/internal/cli"
	"github.com/princepatel/logify/internal/config"
	"github.com/princepatel/logify/internal/tui"
)

const version = "0.1.0"

func main() {
	cli.Version = version
	os.Exit(cli.Run(os.Args[1:], launchTUI))
}

func launchTUI(cfg config.Config, cfgPath string, mock bool, themeOverride string) int {
	if themeOverride != "" {
		cfg.Theme = themeOverride
	}
	m := tui.New(cfg, cfgPath, mock)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return 1
	}
	return 0
}
