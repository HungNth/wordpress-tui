package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/config"
	"wptui/internal/packages"
)

type CreateWizardStep int

const (
	CreateWizardStepInputs CreateWizardStep = iota
	CreateWizardStepPackages
	CreateWizardConfirmDiscard
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
	packageModel        *LiveSearchModel
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

func (m *CreateWizardModel) Inputs() CreateInputs {
	return m.inputs
}

func (m *CreateWizardModel) ShouldExitToSidebar() bool {
	return m.shouldExitToSidebar
}

func (m *CreateWizardModel) IsSubmitted() bool {
	return m.submitted
}

func (m *CreateWizardModel) SelectedPackages() []string {
	if m.packageModel != nil {
		return m.packageModel.FinalSelected()
	}
	return nil
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
	m.packageModel = NewLiveSearchModel(packages.PackageTypePlugin, m.catalog, nil)
	m.step = CreateWizardStepPackages
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
				if text == "" && len(str) == 1 {
					text = str
				}
				if text != "" && !strings.Contains(str, "+") {
					m.appendChar(text)
				}
				return m, nil
			}

		case CreateWizardStepPackages:
			if str == "esc" {
				m.step = CreateWizardStepInputs
				return m, nil
			}
			if str == "enter" {
				m.submitted = true
				return m, nil
			}
			if m.packageModel != nil {
				newM, cmd := m.packageModel.Update(msg)
				m.packageModel = newM.(*LiveSearchModel)
				return m, cmd
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
		if len(m.inputs.WebsiteName) > 0 {
			m.inputs.WebsiteName = m.inputs.WebsiteName[:len(m.inputs.WebsiteName)-1]
		}
	case createFieldWebsiteSlug:
		if len(m.inputs.WebsiteSlug) > 0 {
			m.inputs.WebsiteSlug = m.inputs.WebsiteSlug[:len(m.inputs.WebsiteSlug)-1]
		}
	case createFieldUsername:
		if len(m.inputs.AdminUsername) > 0 {
			m.inputs.AdminUsername = m.inputs.AdminUsername[:len(m.inputs.AdminUsername)-1]
		}
	case createFieldPassword:
		if len(m.inputs.AdminPassword) > 0 {
			m.inputs.AdminPassword = m.inputs.AdminPassword[:len(m.inputs.AdminPassword)-1]
		}
	case createFieldEmail:
		if len(m.inputs.AdminEmail) > 0 {
			m.inputs.AdminEmail = m.inputs.AdminEmail[:len(m.inputs.AdminEmail)-1]
		}
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
		sb.WriteString(fmt.Sprintf("%s%-16s : %s (clean default theme, sample posts)\n\n", tweaksCursor, tweaksLabel, check))

		// Submit button
		btnCursor := "  "
		btnText := "[ Next: Select Packages ]"
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

	case CreateWizardStepPackages:
		sb.WriteString(StyleFocus.Render("  Select Packages (Plugins & Themes)") + "\n")
		sb.WriteString(StyleMuted.Render("  Search catalog live · Space to toggle · Enter to confirm · Esc to go back") + "\n\n")
		if m.packageModel != nil {
			sb.WriteString(m.packageModel.ViewString())
		}

	case CreateWizardConfirmDiscard:
		sb.WriteString(StyleWarning.Render("  ⚠ Unsaved Changes") + "\n\n")
		sb.WriteString("  You have modified form values that have not been submitted.\n\n")
		sb.WriteString(StyleFocus.Render("  Discard changes? (y/n)") + "\n\n")
		sb.WriteString(StyleMuted.Render("  Press 'y' to discard and return to sidebar · 'n' to resume editing"))
	}

	return sb.String()
}
