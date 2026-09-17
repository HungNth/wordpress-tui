package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"wptui/internal/config"
	"wptui/internal/launcher"
	"wptui/internal/tui"
)

type SettingsFlowDependencies struct {
	ConfigPath     string
	Launcher       launcher.ProcessRunner
	PromptAction   func() (tui.SettingsAction, error)
	OnReloadConfig func(*config.Config)
}

func RunDefaultSettingsFlow(ctx context.Context, cfg *config.Config, onReload func(*config.Config)) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine user home directory: %w", err)
	}
	cfgPath, err := config.ConfigPath(home)
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	return RunSettingsFlowWithDeps(ctx, cfg, SettingsFlowDependencies{
		ConfigPath:     cfgPath,
		Launcher:       &launcher.DefaultRunner{},
		PromptAction:   tui.PromptSettingsAction,
		OnReloadConfig: onReload,
	})
}

func RunSettingsFlowWithDeps(ctx context.Context, cfg *config.Config, deps SettingsFlowDependencies) error {
	promptAction := deps.PromptAction
	if promptAction == nil {
		promptAction = tui.PromptSettingsAction
	}

	for {
		action, err := promptAction()
		if err != nil {
			return err
		}

		if action == tui.ActionSettingsBack || action == "" {
			return nil
		}

		switch action {
		case tui.ActionOpenVSCode:
			fmt.Println("\nOpening config.json in VS Code (waiting for editor to close)...")
			if err := launcher.OpenInVSCode(ctx, deps.ConfigPath, deps.Launcher); err != nil {
				tui.PrintSettingsError(fmt.Sprintf("%v", err))
				break
			}

			// Reload configuration strictly after VS Code closes
			newCfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				tui.PrintSettingsError(fmt.Sprintf("Error reloading configuration: %v (previous configuration retained)", err))
				break
			}

			if deps.OnReloadConfig != nil {
				deps.OnReloadConfig(newCfg)
			}
			tui.PrintSettingsSuccess("Configuration reloaded successfully.")

		case tui.ActionOpenCache:
			userCache, err := os.UserCacheDir()
			if err != nil {
				tui.PrintSettingsError(fmt.Sprintf("Failed to locate user cache directory: %v", err))
				break
			}
			cacheDir := filepath.Join(userCache, "wptui")

			if err := launcher.OpenDirectory(ctx, cacheDir, deps.Launcher); err != nil {
				tui.PrintSettingsError(fmt.Sprintf("%v", err))
				break
			}
			tui.PrintSettingsSuccess(fmt.Sprintf("Opened cache directory: %s", cacheDir))
		}
	}
}
