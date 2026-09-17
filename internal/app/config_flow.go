package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/siteconfig"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

type ConfigFlowDependencies struct {
	WPClient       siteconfig.WPClient
	Resolver       PackageResolver
	Catalog        []packages.CatalogItem
	SelectWebsite  func([]deprovision.Candidate) (*deprovision.Candidate, error)
	SelectAction   func(string) (tui.ConfigAction, error)
	SelectAdmin    func([]siteconfig.AdminUser) (*siteconfig.AdminUser, error)
	PromptAdmin    func(siteconfig.AdminUser, *config.Config) (siteconfig.AdminInput, error)
	PromptPackages func(context.Context, *config.Config, []packages.CatalogItem) ([]string, []string, error)
	PromptThemes   func(context.Context, *config.Config, []packages.CatalogItem) ([]string, []string, error)
	PromptThemeAct func() (bool, error)
	PromptContinue func() (bool, error)
	Connector      siteconfig.DBConnector
}

func RunDefaultConfigFlow(ctx context.Context, cfg *config.Config) error {
	client := wpcli.NewClient()
	var resolver PackageResolver
	var catalog []packages.CatalogItem

	if strings.TrimSpace(cfg.PackagesAPIURL) != "" {
		cache, err := packages.NewCacheWithContext(ctx)
		if err == nil {
			resolver = packages.NewResolver(cfg, cache)
		}
		cat, err := packages.FetchCatalog(ctx, nil, cfg.PackagesAPIURL, cfg.PackagesAPIKey)
		if err == nil {
			catalog = cat
		}
	}

	return RunConfigFlowWithDeps(ctx, cfg, ConfigFlowDependencies{
		WPClient: client,
		Resolver: resolver,
		Catalog:  catalog,
	})
}

func RunConfigFlowWithDeps(ctx context.Context, cfg *config.Config, deps ConfigFlowDependencies) error {
	selectWebsite := deps.SelectWebsite
	if selectWebsite == nil {
		selectWebsite = tui.SelectWebsiteForConfig
	}

	selectAction := deps.SelectAction
	if selectAction == nil {
		selectAction = tui.SelectConfigAction
	}

	selectAdmin := deps.SelectAdmin
	if selectAdmin == nil {
		selectAdmin = tui.SelectAdminUser
	}

	promptAdmin := deps.PromptAdmin
	if promptAdmin == nil {
		promptAdmin = tui.PromptAdminInputs
	}

	promptThemeAct := deps.PromptThemeAct
	if promptThemeAct == nil {
		promptThemeAct = tui.PromptThemeActivation
	}

	promptContinue := deps.PromptContinue
	if promptContinue == nil {
		promptContinue = tui.PromptContinueConfiguring
	}

	for {
		candidates, err := deprovision.DiscoverCandidates(ctx, cfg.WebsitesPath, cfg.DeleteExcludes)
		if err != nil {
			return fmt.Errorf("failed to discover websites: %w", err)
		}
		if len(candidates) == 0 {
			fmt.Println("No existing websites found to configure.")
			return nil
		}

		selectedSite, err := selectWebsite(candidates)
		if err != nil {
			return err
		}
		if selectedSite == nil {
			return nil // User backed out to main menu
		}

		// Loop on actions for the selected site
		siteLoop := true
		for siteLoop {
			action, err := selectAction(selectedSite.Slug)
			if err != nil {
				return err
			}

			if action == tui.ActionBack || action == "" {
				siteLoop = false
				break
			}

			switch action {
			case tui.ActionApplyTweaks:
				results := siteconfig.ApplyTweaks(ctx, selectedSite.Path, cfg.WPTweaks, deps.WPClient, tui.PrintProgress)
				tui.PrintTweakSummary(results)

			case tui.ActionChangeAdmin:
				admins, err := siteconfig.DiscoverAdministrators(ctx, selectedSite.Path, deps.WPClient)
				if err != nil {
					fmt.Printf("Error detecting administrator: %v\n", err)
					break
				}
				chosenAdmin, err := selectAdmin(admins)
				if err != nil {
					fmt.Printf("Error selecting administrator: %v\n", err)
					break
				}
				adminInputs, err := promptAdmin(*chosenAdmin, cfg)
				if err != nil {
					fmt.Printf("Error obtaining administrator inputs: %v\n", err)
					break
				}

				tui.PrintProgress(0, 0, "Extracting database configuration from wp-config.php...")
				dbCfg, err := siteconfig.ExtractDBConfig(ctx, selectedSite.Path, deps.WPClient)
				if err != nil {
					fmt.Printf("Error extracting database configuration: %v\n", err)
					break
				}

				err = siteconfig.UpdateAdminCredentials(ctx, selectedSite.Path, dbCfg, adminInputs, deps.WPClient, deps.Connector, tui.PrintProgress)
				if err != nil {
					fmt.Printf("Failed to update administrator credentials: %v\n", err)
				} else {
					fmt.Println("\nAdministrator credentials updated successfully.")
				}

			case tui.ActionInstallPlugins:
				var chosenPlugins []string
				if deps.PromptPackages != nil {
					plugins, _, err := deps.PromptPackages(ctx, cfg, deps.Catalog)
					if err != nil {
						fmt.Printf("Plugin selection cancelled: %v\n", err)
						break
					}
					chosenPlugins = plugins
				} else {
					opts := tui.BuildPluginOptions(cfg.Plugins)
					plugins, err := tui.SelectPackagesFlow(ctx, packages.PackageTypePlugin, opts, deps.Catalog)
					if err != nil {
						fmt.Printf("Plugin selection error: %v\n", err)
						break
					}
					chosenPlugins = plugins
				}

				if len(chosenPlugins) == 0 {
					fmt.Println("No plugins selected.")
					break
				}

				if deps.Resolver == nil {
					fmt.Println("Package resolver is unavailable (API URL not configured).")
					break
				}

				refs := make([]packages.PackageRef, 0, len(chosenPlugins))
				for _, slug := range chosenPlugins {
					refs = append(refs, packages.PackageRef{Slug: slug, Type: packages.PackageTypePlugin})
				}

				stageDir, err := os.MkdirTemp("", "wptui-config-plugins-*")
				if err != nil {
					fmt.Printf("Failed to create temporary directory: %v\n", err)
					break
				}

				tui.PrintProgress(0, 0, "Resolving plugin archives...")
				artifacts, err := deps.Resolver.ResolveAll(ctx, refs, stageDir)
				if err != nil {
					fmt.Printf("Warning: partial/failed package resolution: %v\n", err)
				}

				results := siteconfig.InstallPackages(ctx, selectedSite.Path, packages.PackageTypePlugin, artifacts, true, deps.WPClient, tui.PrintProgress)
				_ = os.RemoveAll(stageDir)
				tui.PrintPackageInstallSummary(results)

			case tui.ActionInstallThemes:
				var chosenThemes []string
				if deps.PromptThemes != nil {
					themes, _, err := deps.PromptThemes(ctx, cfg, deps.Catalog)
					if err != nil {
						fmt.Printf("Theme selection cancelled: %v\n", err)
						break
					}
					chosenThemes = themes
				} else {
					opts := tui.BuildThemeOptions(cfg.Themes)
					themes, err := tui.SelectPackagesFlow(ctx, packages.PackageTypeTheme, opts, deps.Catalog)
					if err != nil {
						fmt.Printf("Theme selection error: %v\n", err)
						break
					}
					chosenThemes = themes
				}

				if len(chosenThemes) == 0 {
					fmt.Println("No themes selected.")
					break
				}

				if deps.Resolver == nil {
					fmt.Println("Package resolver is unavailable (API URL not configured).")
					break
				}

				activate, err := promptThemeAct()
				if err != nil {
					fmt.Printf("Theme activation prompt cancelled: %v\n", err)
					break
				}

				refs := make([]packages.PackageRef, 0, len(chosenThemes))
				for _, slug := range chosenThemes {
					refs = append(refs, packages.PackageRef{Slug: slug, Type: packages.PackageTypeTheme})
				}

				stageDir, err := os.MkdirTemp("", "wptui-config-themes-*")
				if err != nil {
					fmt.Printf("Failed to create temporary directory: %v\n", err)
					break
				}

				tui.PrintProgress(0, 0, "Resolving theme archives...")
				artifacts, err := deps.Resolver.ResolveAll(ctx, refs, stageDir)
				if err != nil {
					fmt.Printf("Warning: partial/failed package resolution: %v\n", err)
				}

				results := siteconfig.InstallPackages(ctx, selectedSite.Path, packages.PackageTypeTheme, artifacts, activate, deps.WPClient, tui.PrintProgress)
				_ = os.RemoveAll(stageDir)
				tui.PrintPackageInstallSummary(results)
			}

			cont, err := promptContinue()
			if err != nil || !cont {
				siteLoop = false
			}
		}
	}
}
