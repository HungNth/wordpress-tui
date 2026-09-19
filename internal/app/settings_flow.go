package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/launcher"
	"wptui/internal/tui"
)

type SettingsFlowDependencies struct {
	ConfigPath     string
	CacheDir       func() (string, error)
	Launcher       launcher.ProcessRunner
	PromptAction   func() (tui.SettingsAction, error)
	OnReloadConfig func(*config.Config)
}

func defaultSettingsCacheDir() (string, error) {
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCache, "wptui"), nil
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
		CacheDir:       defaultSettingsCacheDir,
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
	cacheDir := deps.CacheDir
	if cacheDir == nil {
		cacheDir = defaultSettingsCacheDir
	}
	for {
		action, err := promptAction()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, ErrUserCancelled) {
				return nil
			}
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
			dir, err := cacheDir()
			if err != nil {
				tui.PrintSettingsError(fmt.Sprintf("Failed to locate user cache directory: %v", err))
				break
			}
			if err := launcher.OpenDirectory(ctx, dir, deps.Launcher); err != nil {
				tui.PrintSettingsError(fmt.Sprintf("%v", err))
				break
			}
			tui.PrintSettingsSuccess(fmt.Sprintf("Opened cache directory: %s", dir))
		}
	}
}
