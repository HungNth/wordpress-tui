package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/config"
	"wptui/internal/packages"
)

type CreateWizardStep int

const (
	CreateWizardStepInputs CreateWizardStep = iota
	CreateWizardStepPlugins
	CreateWizardStepThemes
	CreateWizardStepActiveTheme
	CreateWizardConfirmDiscard

	CreateWizardStepPackages = CreateWizardStepPlugins
)

const (
	createFieldWebsiteName = 0
	createFieldWebsiteSlug = 1
	createFieldUsername    = 2
	createFieldPassword    = 3
	createFieldEmail       = 4
	createFieldTweaks      = 5
	createFieldSubmit      = 6
	createFieldCount       = 7
)

type CreateWizardModel struct {
	cfg                 *config.Config
	checker             SlugAvailabilityChecker
	catalog             []packages.CatalogItem
	step                CreateWizardStep
	fieldIndex          int
	inputs              CreateInputs
	cleanInputs         CreateInputs
	validationErr       string
	pluginModel         *LiveSearchModel
	themeModel          *LiveSearchModel
	activeThemeSlug     string
	activeThemeCursor   int
	activeThemeChoices  []string
	shouldExitToSidebar bool
	submitted           bool
}

func NewCreateWizardModel(cfg *config.Config, catalog []packages.CatalogItem, checker SlugAvailabilityChecker) *CreateWizardModel {
	initialInputs := CreateInputs{
		WebsiteName:   "",
		WebsiteSlug:   "",
		AdminUsername: cfg.DefaultAdminUsername,
		AdminPassword: cfg.DefaultAdminPassword,
		AdminEmail:    cfg.DefaultAdminEmail,
		ApplyTweaks:   true,
	}

	return &CreateWizardModel{
		cfg:                 cfg,
		checker:             checker,
		catalog:             catalog,
		step:                CreateWizardStepInputs,
		fieldIndex:          0,
		inputs:              initialInputs,
		cleanInputs:         initialInputs,
		shouldExitToSidebar: false,
		submitted:           false,
	}
}

func (m *CreateWizardModel) Step() CreateWizardStep {
	return m.step
}

func (m *CreateWizardModel) FieldIndex() int {
	return m.fieldIndex
}

func (m *CreateWizardModel) hasPlugins() bool {
	for _, it := range m.catalog {
		if it.Type == "plugin" {
			return true
		}
	}
	return false
}

func (m *CreateWizardModel) hasThemes() bool {
	for _, it := range m.catalog {
		if it.Type == "theme" {
			return true
		}
	}
	return false
}

func (m *CreateWizardModel) hasPackages() bool {
	return m.hasPlugins() || m.hasThemes()
}

func (m *CreateWizardModel) Inputs() CreateInputs {
	inp := m.inputs
	inp.ActiveThemeSlug = m.activeThemeSlug
	return inp
}

func (m *CreateWizardModel) ShouldExitToSidebar() bool {
	return m.shouldExitToSidebar
}

func (m *CreateWizardModel) IsSubmitted() bool {
	return m.submitted
}

func (m *CreateWizardModel) SelectedPlugins() []string {
	if m.pluginModel != nil {
		return m.pluginModel.FinalSelected()
	}
	return nil
}

func (m *CreateWizardModel) SelectedThemes() []string {
	if m.themeModel != nil {
		return m.themeModel.FinalSelected()
	}
	return nil
}

func (m *CreateWizardModel) ActiveThemeSlug() string {
	return m.activeThemeSlug
}

func (m *CreateWizardModel) SelectedPackages() []string {
	var pkgs []string
	pkgs = append(pkgs, m.SelectedPlugins()...)
	pkgs = append(pkgs, m.SelectedThemes()...)
	return pkgs
}

func (m *CreateWizardModel) IsDirty() bool {
	return m.inputs.WebsiteName != m.cleanInputs.WebsiteName ||
		m.inputs.WebsiteSlug != m.cleanInputs.WebsiteSlug ||
		m.inputs.AdminUsername != m.cleanInputs.AdminUsername ||
		m.inputs.AdminPassword != m.cleanInputs.AdminPassword ||
		m.inputs.AdminEmail != m.cleanInputs.AdminEmail ||
		m.inputs.ApplyTweaks != m.cleanInputs.ApplyTweaks
}

func (m *CreateWizardModel) Init() tea.Cmd {
	return nil
}

func (m *CreateWizardModel) submitStep1() {
	resolvedSlug, err := ResolveAndValidateSlug(m.inputs.WebsiteName, m.inputs.WebsiteSlug, m.checker)
	if err != nil {
		m.validationErr = err.Error()
		return
	}

	m.inputs.WebsiteSlug = resolvedSlug
	m.validationErr = ""

	if m.hasPlugins() {
		m.pluginModel = NewLiveSearchModel(packages.PackageTypePlugin, m.catalog, nil)
		m.step = CreateWizardStepPlugins
	} else if m.hasThemes() {
		m.themeModel = NewLiveSearchModel(packages.PackageTypeTheme, m.catalog, nil)
		m.step = CreateWizardStepThemes
	} else {
		m.submitted = true
	}
}

func (m *CreateWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()

		switch m.step {
		case CreateWizardStepInputs:
			switch str {
			case "esc":
				if m.IsDirty() {
					m.step = CreateWizardConfirmDiscard
				} else {
					m.shouldExitToSidebar = true
				}
				return m, nil

			case "tab", "down":
				m.fieldIndex = (m.fieldIndex + 1) % createFieldCount
				return m, nil

			case "shift+tab", "backtab":
				if m.fieldIndex == 0 {
					if m.IsDirty() {
						m.step = CreateWizardConfirmDiscard
					} else {
						m.shouldExitToSidebar = true
					}
					return m, nil
				}
				m.fieldIndex = (m.fieldIndex - 1 + createFieldCount) % createFieldCount
				return m, nil

			case "up":
				m.fieldIndex = (m.fieldIndex - 1 + createFieldCount) % createFieldCount
				return m, nil

			case "enter":
				if m.fieldIndex == createFieldSubmit {
					m.submitStep1()
					return m, nil
				}
				m.fieldIndex = (m.fieldIndex + 1) % createFieldCount
				return m, nil

			case " ", "space":
				if m.fieldIndex == createFieldTweaks {
					m.inputs.ApplyTweaks = !m.inputs.ApplyTweaks
					return m, nil
				}
				m.appendChar(" ")
				return m, nil

			case "backspace":
				m.deleteLastChar()
				return m, nil

			default:
				text := msg.Text
				if text == "" && utf8.RuneCountInString(str) == 1 {
					text = str
				}
				if text != "" && !strings.Contains(str, "+") {
					m.appendChar(text)
				}
				return m, nil
			}

		case CreateWizardStepPlugins:
			if str == "esc" {
				m.step = CreateWizardStepInputs
				return m, nil
			}
			if str == "enter" {
				if m.hasThemes() {
					m.themeModel = NewLiveSearchModel(packages.PackageTypeTheme, m.catalog, nil)
					m.step = CreateWizardStepThemes
					return m, nil
				}
				m.submitted = true
				return m, nil
			}
			if m.pluginModel != nil {
				newM, cmd := m.pluginModel.Update(msg)
				m.pluginModel = newM.(*LiveSearchModel)
				return m, cmd
			}
			return m, nil

		case CreateWizardStepThemes:
			if str == "esc" {
				if m.hasPlugins() {
					m.step = CreateWizardStepPlugins
				} else {
					m.step = CreateWizardStepInputs
				}
				return m, nil
			}
			if str == "enter" {
				selectedThemes := m.SelectedThemes()
				if len(selectedThemes) > 0 {
					m.activeThemeChoices = append([]string{""}, selectedThemes...)
					m.activeThemeCursor = 0
					m.step = CreateWizardStepActiveTheme
					return m, nil
				}
				m.activeThemeSlug = ""
				m.submitted = true
				return m, nil
			}
			if m.themeModel != nil {
				newM, cmd := m.themeModel.Update(msg)
				m.themeModel = newM.(*LiveSearchModel)
				return m, cmd
			}
			return m, nil

		case CreateWizardStepActiveTheme:
			switch str {
			case "esc":
				m.step = CreateWizardStepThemes
				return m, nil
			case "up", "k":
				if m.activeThemeCursor > 0 {
					m.activeThemeCursor--
				}
				return m, nil
			case "down", "j":
				if m.activeThemeCursor < len(m.activeThemeChoices)-1 {
					m.activeThemeCursor++
				}
				return m, nil
			case "enter":
				m.activeThemeSlug = m.activeThemeChoices[m.activeThemeCursor]
				m.submitted = true
				return m, nil
			}
			return m, nil
		case CreateWizardConfirmDiscard:
			switch str {
			case "y", "Y":
				m.shouldExitToSidebar = true
				m.step = CreateWizardStepInputs
				m.inputs = m.cleanInputs
				return m, nil
			case "n", "N", "esc":
				m.step = CreateWizardStepInputs
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *CreateWizardModel) appendChar(text string) {
	switch m.fieldIndex {
	case createFieldWebsiteName:
		m.inputs.WebsiteName += text
	case createFieldWebsiteSlug:
		m.inputs.WebsiteSlug += text
	case createFieldUsername:
		m.inputs.AdminUsername += text
	case createFieldPassword:
		m.inputs.AdminPassword += text
	case createFieldEmail:
		m.inputs.AdminEmail += text
	}
}

func (m *CreateWizardModel) deleteLastChar() {
	switch m.fieldIndex {
	case createFieldWebsiteName:
		m.inputs.WebsiteName = deleteLastRune(m.inputs.WebsiteName)
	case createFieldWebsiteSlug:
		m.inputs.WebsiteSlug = deleteLastRune(m.inputs.WebsiteSlug)
	case createFieldUsername:
		m.inputs.AdminUsername = deleteLastRune(m.inputs.AdminUsername)
	case createFieldPassword:
		m.inputs.AdminPassword = deleteLastRune(m.inputs.AdminPassword)
	case createFieldEmail:
		m.inputs.AdminEmail = deleteLastRune(m.inputs.AdminEmail)
	}
}

func (m *CreateWizardModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func (m *CreateWizardModel) Render(width, height int) string {
	var sb strings.Builder

	switch m.step {
	case CreateWizardStepInputs:
		sb.WriteString(StyleFocus.Render("  Create New WordPress Website") + "\n")
		sb.WriteString(StyleMuted.Render("  Enter basic details and admin credentials. Slug auto-derives if empty.") + "\n\n")

		fields := []struct {
			label string
			val   string
			idx   int
		}{
			{"Website Name", m.inputs.WebsiteName, createFieldWebsiteName},
			{"Website Slug", m.inputs.WebsiteSlug, createFieldWebsiteSlug},
			{"Admin Username", m.inputs.AdminUsername, createFieldUsername},
			{"Admin Password", strings.Repeat("*", len(m.inputs.AdminPassword)), createFieldPassword},
			{"Admin Email", m.inputs.AdminEmail, createFieldEmail},
		}

		for _, f := range fields {
			cursor := "  "
			label := fmt.Sprintf("%-16s", f.label)
			val := f.val

			if f.idx == m.fieldIndex {
				cursor = StyleFocus.Render("> ")
				label = StyleFocus.Render(label)
				val = lipgloss.NewStyle().Foreground(ColorCyan).Underline(true).Render(val + "_")
			} else {
				if val == "" {
					val = StyleMuted.Render("(empty)")
				}
			}
			sb.WriteString(fmt.Sprintf("%s%s : %s\n", cursor, label, val))
		}

		// Tweaks checkbox
		tweaksCursor := "  "
		tweaksLabel := "Apply Tweaks"
		check := "[ ]"
		if m.inputs.ApplyTweaks {
			check = StyleSuccess.Render("[x]")
		}
		if m.fieldIndex == createFieldTweaks {
			tweaksCursor = StyleFocus.Render("> ")
			tweaksLabel = StyleFocus.Render(tweaksLabel)
		}
		sb.WriteString(fmt.Sprintf("%s%-16s : %s (apply configured WordPress tweaks)\n\n", tweaksCursor, tweaksLabel, check))

		// Submit button
		btnCursor := "  "
		btnText := "[ Next: Select Packages ]"
		if !m.hasPackages() {
			btnText = "[ Create Website ]"
		}
		if m.fieldIndex == createFieldSubmit {
			btnCursor = StyleFocus.Render("> ")
			btnText = StyleFocus.Render(btnText)
		} else {
			btnText = StyleMuted.Render(btnText)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", btnCursor, btnText))

		if m.validationErr != "" {
			sb.WriteString("\n  " + StyleError.Render("✗ "+m.validationErr) + "\n")
		}

	case CreateWizardStepPlugins:
		sb.WriteString(StyleFocus.Render("  Select Plugins") + "\n")
		sb.WriteString(StyleMuted.Render("  Search catalog live · Space to toggle · Enter to continue · Esc to go back") + "\n\n")
		if m.pluginModel != nil {
			sb.WriteString(m.pluginModel.ViewString())
		}

	case CreateWizardStepThemes:
		sb.WriteString(StyleFocus.Render("  Select Themes") + "\n")
		sb.WriteString(StyleMuted.Render("  Search catalog live · Space to toggle · Enter to continue · Esc to go back") + "\n\n")
		if m.themeModel != nil {
			sb.WriteString(m.themeModel.ViewString())
		}

	case CreateWizardStepActiveTheme:
		sb.WriteString(StyleFocus.Render("  Select Active Theme") + "\n")
		sb.WriteString(StyleMuted.Render("  Choose which theme to activate (optional). Up/Down to navigate · Enter to confirm") + "\n\n")
		for i, slug := range m.activeThemeChoices {
			cursor := "  "
			label := slug
			if slug == "" {
				label = "None (do not activate any theme)"
			}
			if i == m.activeThemeCursor {
				cursor = StyleFocus.Render("> ")
				label = StyleFocus.Render(label)
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", cursor, label))
		}
	case CreateWizardConfirmDiscard:
		sb.WriteString(StyleWarning.Render("  ⚠ Unsaved Changes") + "\n\n")
		sb.WriteString("  You have modified form values that have not been submitted.\n\n")
		sb.WriteString(StyleFocus.Render("  Discard changes? (y/n)") + "\n\n")
		sb.WriteString(StyleMuted.Render("  Press 'y' to discard and return to sidebar · 'n' to resume editing"))
	}

	return sb.String()
}
