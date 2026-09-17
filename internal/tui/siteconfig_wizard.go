package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/siteconfig"
)

type ConfigAction string

const (
	ActionApplyTweaks ConfigAction = "tweaks"
	ActionChangeAdmin ConfigAction = "admin"
	ActionInstallPlugins ConfigAction = "plugins"
	ActionInstallThemes  ConfigAction = "themes"
	ActionBack           ConfigAction = "back"
)

// SelectWebsiteForConfig displays a single-select menu of candidate websites or allows going back.
func SelectWebsiteForConfig(websites []deprovision.Candidate) (*deprovision.Candidate, error) {
	if len(websites) == 0 {
		return nil, errors.New("no existing websites found to configure")
	}

	options := make([]huh.Option[string], 0, len(websites)+1)
	options = append(options, huh.NewOption("← Back to Main Menu", "back"))

	for _, w := range websites {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", w.Slug, w.Path), w.Slug))
	}

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Website to Configure").
				Description("Select a website to modify tweaks, admin credentials, or packages").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	if choice == "back" || choice == "" {
		return nil, nil // graceful back navigation
	}

	for i := range websites {
		if websites[i].Slug == choice {
			return &websites[i], nil
		}
	}

	return nil, nil
}

// SelectConfigAction displays the sub-menu for the selected website.
func SelectConfigAction(siteSlug string) (ConfigAction, error) {
	options := []huh.Option[ConfigAction]{
		huh.NewOption("1. Apply wp_tweaks from config.json", ActionApplyTweaks),
		huh.NewOption("2. Change administrator credentials", ActionChangeAdmin),
		huh.NewOption("3. Install plugins", ActionInstallPlugins),
		huh.NewOption("4. Install themes", ActionInstallThemes),
		huh.NewOption("← Back (Select another website)", ActionBack),
	}

	var choice ConfigAction
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[ConfigAction]().
				Title(fmt.Sprintf("Configure Website: %s", siteSlug)).
				Description("Choose an operation to perform").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return choice, nil
}

// SelectAdminUser prompts the user to select an administrator if multiple exist.
func SelectAdminUser(admins []siteconfig.AdminUser) (*siteconfig.AdminUser, error) {
	if len(admins) == 0 {
		return nil, errors.New("no administrators found")
	}
	if len(admins) == 1 {
		return &admins[0], nil
	}

	options := make([]huh.Option[int], 0, len(admins))
	for _, a := range admins {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (ID: %d, Email: %s)", a.UserLogin, a.ID, a.UserEmail), a.ID))
	}

	selectedID := admins[0].ID
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Multiple administrators detected. Select one to configure:").
				Options(options...).
				Value(&selectedID),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	for i := range admins {
		if admins[i].ID == selectedID {
			return &admins[i], nil
		}
	}
	return &admins[0], nil
}

// PromptAdminInputs displays a form to update administrator credentials, using defaults if left blank.
func PromptAdminInputs(current siteconfig.AdminUser, defaults *config.Config) (siteconfig.AdminInput, error) {
	var (
		newUsername string
		newPassword string
		newEmail    string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("New Username (Current: %s)", current.UserLogin)).
				Description(fmt.Sprintf("Leave blank to use default: %s", defaults.DefaultAdminUsername)).
				Value(&newUsername),

			huh.NewInput().
				Title("New Password").
				Description(fmt.Sprintf("Leave blank to use default: %s", defaults.DefaultAdminPassword)).
				EchoMode(huh.EchoModePassword).
				Value(&newPassword),

			huh.NewInput().
				Title(fmt.Sprintf("New Email (Current: %s)", current.UserEmail)).
				Description(fmt.Sprintf("Leave blank to use default: %s", defaults.DefaultAdminEmail)).
				Value(&newEmail),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return siteconfig.AdminInput{}, err
	}

	resUsername := strings.TrimSpace(newUsername)
	if resUsername == "" {
		resUsername = defaults.DefaultAdminUsername
	}

	resPassword := strings.TrimSpace(newPassword)
	if resPassword == "" {
		resPassword = defaults.DefaultAdminPassword
	}

	resEmail := strings.TrimSpace(newEmail)
	if resEmail == "" {
		resEmail = defaults.DefaultAdminEmail
	}

	return siteconfig.AdminInput{
		UserID:      current.ID,
		NewUsername: resUsername,
		NewPassword: resPassword,
		NewEmail:    resEmail,
	}, nil
}

// PromptThemeActivation asks whether to activate newly installed themes.
func PromptThemeActivation() (bool, error) {
	var activate bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Activate theme after installation?").
				Description("Select Yes to activate immediately, No to install only").
				Affirmative("Yes").
				Negative("No").
				Value(&activate),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return false, err
	}
	return activate, nil
}

// PromptContinueConfiguring asks if the user wants to continue with another action on the same site.
func PromptContinueConfiguring() (bool, error) {
	var cont bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Continue configuring this website?").
				Affirmative("Yes").
				Negative("No").
				Value(&cont),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return false, err
	}
	return cont, nil
}
func PrintProgress(step, total int, message string) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	if total > 0 {
		fmt.Printf("  %s %s\n", cyan.Render(fmt.Sprintf("[%d/%d]", step, total)), message)
	} else {
		fmt.Printf("  %s %s\n", cyan.Render("[-]"), message)
	}
}


// PrintTweakSummary prints the status of applied tweaks in color.
func PrintTweakSummary(results []siteconfig.TweakStatus) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)

	fmt.Println("\n" + cyan.Render("=== WP Tweaks Summary ==="))
	for _, r := range results {
		desc := fmt.Sprintf("[%s] %s=%s", r.Tweak.Type, r.Tweak.Key, r.Tweak.Value)
		if r.Success {
			fmt.Printf("  %s %s\n", green.Render("[✓]"), desc)
		} else {
			fmt.Printf("  %s %s: %s\n", red.Render("[✗]"), desc, r.Err)
		}
	}
	fmt.Println()
}

func PrintPackageInstallSummary(results []siteconfig.PackageStatus) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)

	fmt.Println("\n" + cyan.Render("=== Package Installation Summary ==="))
	for _, r := range results {
		act := ""
		if r.Activated {
			act = " (Activated)"
		}
		if r.Skipped {
			fmt.Printf("  %s %s: %s — %s\n", yellow.Render("[-]"), r.Type, r.Slug, r.SkipReason)
		} else if r.Success {
			fmt.Printf("  %s %s: %s%s\n", green.Render("[✓]"), r.Type, r.Slug, act)
		} else {
			fmt.Printf("  %s %s: %s: %s\n", red.Render("[✗]"), r.Type, r.Slug, r.Err)
		}
	}
	fmt.Println()
}
