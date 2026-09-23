package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/launcher"
)

type SettingsModelOption struct {
	Label string
	Key   string
}

type SettingsModel struct {
	cfg          *config.Config
	cfgPath      string
	cacheDir     string
	runner       launcher.ProcessRunner
	cursor       int
	options      []SettingsModelOption
	statusMsg    string
	shouldReturn bool
	OnReload     func(newCfg *config.Config)
}

func NewSettingsModel(cfg *config.Config, cfgPath string, runner launcher.ProcessRunner, cacheDir ...string) *SettingsModel {
	dir := ""
	if len(cacheDir) > 0 && cacheDir[0] != "" {
		dir = cacheDir[0]
	} else {
		userCache, err := os.UserCacheDir()
		if err == nil {
			dir = filepath.Join(userCache, "wptui")
		}
	}

	options := []SettingsModelOption{
		{Label: "Open config.json in VS Code", Key: "vscode"},
		{Label: "Open Cache Directory", Key: "cache"},
		{Label: "Reload Configuration", Key: "reload"},
	}

	return &SettingsModel{
		cfg:          cfg,
		cfgPath:      cfgPath,
		cacheDir:     dir,
		runner:       runner,
		cursor:       0,
		options:      options,
		shouldReturn: false,
	}
}

func (m *SettingsModel) Init() tea.Cmd {
	return nil
}

func (m *SettingsModel) ShouldReturn() bool {
	return m.shouldReturn
}

func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()
		switch str {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
			return m, nil
		case "esc":
			m.shouldReturn = true
			return m, nil
		case "enter":
			opt := m.options[m.cursor]
			switch opt.Key {
			case "vscode":
				err := launcher.OpenInVSCode(context.Background(), m.cfgPath, m.runner)
				if err != nil {
					m.statusMsg = StyleError.Render(fmt.Sprintf("Failed to open VS Code: %v", err))
				} else {
					m.statusMsg = StyleSuccess.Render(fmt.Sprintf("Opened %s in VS Code", m.cfgPath))
				}
				return m, nil

			case "cache":
				err := launcher.OpenDirectory(context.Background(), m.cacheDir, m.runner)
				if err != nil {
					m.statusMsg = StyleError.Render(fmt.Sprintf("Failed to open cache directory: %v", err))
				} else {
					m.statusMsg = StyleSuccess.Render(fmt.Sprintf("Opened cache directory: %s", m.cacheDir))
				}
				return m, nil

			case "reload":
				newCfg, err := config.Load(m.cfgPath)
				if err != nil {
					m.statusMsg = StyleError.Render(fmt.Sprintf("Failed to reload configuration: %v", err))
				} else {
					m.cfg = newCfg
					if m.OnReload != nil {
						m.OnReload(newCfg)
					}
					m.statusMsg = StyleSuccess.Render("Configuration reloaded successfully")
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m *SettingsModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func (m *SettingsModel) Render(width, height int) string {
	var sb strings.Builder
	sb.WriteString(StyleFocus.Render("  Application Settings") + "\n")
	sb.WriteString(StyleMuted.Render("  Manage application configuration and system caches") + "\n\n")

	for i, opt := range m.options {
		cursor := "  "
		label := opt.Label
		if i == m.cursor {
			cursor = StyleFocus.Render("> ")
			label = StyleFocus.Render(label)
		} else {
			label = StyleMuted.Render(label)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n\n", cursor, label))
	}

	if m.statusMsg != "" {
		sb.WriteString("\n  " + m.statusMsg + "\n")
	}

	return sb.String()
}
