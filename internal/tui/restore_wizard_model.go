package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/config"
	"wptui/internal/restore"
)

type RestoreWizardStep int

const (
	RestoreWizardStepFormat RestoreWizardStep = iota
	RestoreWizardStepArchive
	RestoreWizardStepInputs
)

type RestoreFormatOption struct {
	Strategy restore.Strategy
	Label    string
}

type RestoreWizardModel struct {
	cfg             *config.Config
	checker         SlugAvailabilityChecker
	step            RestoreWizardStep
	strategy        restore.Strategy
	formatCursor    int
	formatOptions   []RestoreFormatOption
	archives        []string
	archiveCursor   int
	selectedArchive string
	fieldIndex      int
	inputs          CreateInputs
	validationErr   string
	submitted       bool
	shouldReturn    bool
}

func NewRestoreWizardModel(cfg *config.Config, checker SlugAvailabilityChecker) *RestoreWizardModel {
	formats := []RestoreFormatOption{
		{Strategy: restore.StrategyFullZIP, Label: "Full source code & database (.zip)"},
		{Strategy: restore.StrategyAI1WM, Label: "All-in-One WP Migration (.wpress)"},
	}

	initialInputs := CreateInputs{
		WebsiteName:   "",
		WebsiteSlug:   "",
		AdminUsername: cfg.DefaultAdminUsername,
		AdminPassword: cfg.DefaultAdminPassword,
		AdminEmail:    cfg.DefaultAdminEmail,
		ApplyTweaks:   false,
	}

	return &RestoreWizardModel{
		cfg:           cfg,
		checker:       checker,
		step:          RestoreWizardStepFormat,
		strategy:      restore.StrategyFullZIP,
		formatCursor:  0,
		formatOptions: formats,
		fieldIndex:    0,
		inputs:        initialInputs,
		submitted:     false,
		shouldReturn:  false,
	}
}

func (m *RestoreWizardModel) Step() RestoreWizardStep {
	return m.step
}

func (m *RestoreWizardModel) Strategy() restore.Strategy {
	return m.strategy
}

func (m *RestoreWizardModel) SelectedArchive() string {
	return m.selectedArchive
}

func (m *RestoreWizardModel) Inputs() CreateInputs {
	return m.inputs
}

func (m *RestoreWizardModel) IsSubmitted() bool {
	return m.submitted
}

func (m *RestoreWizardModel) ShouldReturn() bool {
	return m.shouldReturn
}

func (m *RestoreWizardModel) Init() tea.Cmd {
	return nil
}

func (m *RestoreWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()

		switch m.step {
		case RestoreWizardStepFormat:
			switch str {
			case "up", "k":
				if m.formatCursor > 0 {
					m.formatCursor--
				}
				return m, nil
			case "down", "j":
				if m.formatCursor < len(m.formatOptions)-1 {
					m.formatCursor++
				}
				return m, nil
			case "esc":
				m.shouldReturn = true
				return m, nil
			case "enter":
				m.strategy = m.formatOptions[m.formatCursor].Strategy
				archives, _ := ScanBackupArchives(m.cfg.BackupPath, m.strategy)
				m.archives = archives
				m.archiveCursor = 0
				m.step = RestoreWizardStepArchive
				return m, nil
			}

		case RestoreWizardStepArchive:
			switch str {
			case "up", "k":
				if m.archiveCursor > 0 {
					m.archiveCursor--
				}
				return m, nil
			case "down", "j":
				if m.archiveCursor < len(m.archives)-1 {
					m.archiveCursor++
				}
				return m, nil
			case "esc":
				m.step = RestoreWizardStepFormat
				return m, nil
			case "enter":
				if len(m.archives) > 0 {
					m.selectedArchive = m.archives[m.archiveCursor]
					m.step = RestoreWizardStepInputs
				}
				return m, nil
			}

		case RestoreWizardStepInputs:
			switch str {
			case "esc":
				m.step = RestoreWizardStepArchive
				return m, nil
			case "tab", "down":
				m.fieldIndex = (m.fieldIndex + 1) % 6
				return m, nil
			case "shift+tab", "backtab", "up":
				m.fieldIndex = (m.fieldIndex - 1 + 6) % 6
				return m, nil
			case "enter":
				if m.fieldIndex == 5 { // Submit button
					resolvedSlug, err := ResolveAndValidateSlug(m.inputs.WebsiteName, m.inputs.WebsiteSlug, m.checker)
					if err != nil {
						m.validationErr = err.Error()
						return m, nil
					}
					m.inputs.WebsiteSlug = resolvedSlug
					m.validationErr = ""
					m.submitted = true
					return m, nil
				}
				m.fieldIndex = (m.fieldIndex + 1) % 6
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
		}
	}

	return m, nil
}

func (m *RestoreWizardModel) appendChar(text string) {
	switch m.fieldIndex {
	case 0:
		m.inputs.WebsiteName += text
	case 1:
		m.inputs.WebsiteSlug += text
	case 2:
		m.inputs.AdminUsername += text
	case 3:
		m.inputs.AdminPassword += text
	case 4:
		m.inputs.AdminEmail += text
	}
}

func (m *RestoreWizardModel) deleteLastChar() {
	switch m.fieldIndex {
	case 0:
		if len(m.inputs.WebsiteName) > 0 {
			m.inputs.WebsiteName = m.inputs.WebsiteName[:len(m.inputs.WebsiteName)-1]
		}
	case 1:
		if len(m.inputs.WebsiteSlug) > 0 {
			m.inputs.WebsiteSlug = m.inputs.WebsiteSlug[:len(m.inputs.WebsiteSlug)-1]
		}
	case 2:
		if len(m.inputs.AdminUsername) > 0 {
			m.inputs.AdminUsername = m.inputs.AdminUsername[:len(m.inputs.AdminUsername)-1]
		}
	case 3:
		if len(m.inputs.AdminPassword) > 0 {
			m.inputs.AdminPassword = m.inputs.AdminPassword[:len(m.inputs.AdminPassword)-1]
		}
	case 4:
		if len(m.inputs.AdminEmail) > 0 {
			m.inputs.AdminEmail = m.inputs.AdminEmail[:len(m.inputs.AdminEmail)-1]
		}
	}
}

func (m *RestoreWizardModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func (m *RestoreWizardModel) Render(width, height int) string {
	var sb strings.Builder

	switch m.step {
	case RestoreWizardStepFormat:
		sb.WriteString(StyleFocus.Render("  Restore Website from Backup") + "\n")
		sb.WriteString(StyleMuted.Render("  Choose the backup archive format to restore") + "\n\n")

		for i, opt := range m.formatOptions {
			cursor := "  "
			label := opt.Label
			if i == m.formatCursor {
				cursor = StyleFocus.Render("> ")
				label = StyleFocus.Render(label)
			} else {
				label = StyleMuted.Render(label)
			}
			sb.WriteString(fmt.Sprintf("%s%s\n\n", cursor, label))
		}

	case RestoreWizardStepArchive:
		sb.WriteString(StyleFocus.Render("  Select Backup Archive") + "\n")
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("  Scanning archives in %s", m.cfg.BackupPath)) + "\n\n")

		if len(m.archives) == 0 {
			sb.WriteString(StyleError.Render("  No backup archives found matching format.") + "\n\n")
			sb.WriteString(StyleMuted.Render("  Press Esc to go back and choose a different format or backup location."))
		} else {
			for i, arch := range m.archives {
				cursor := "  "
				base := filepath.Base(arch)
				if i == m.archiveCursor {
					cursor = StyleFocus.Render("> ")
					base = StyleFocus.Render(base)
				} else {
					base = StyleMuted.Render(base)
				}
				sb.WriteString(fmt.Sprintf("%s%s\n", cursor, base))
			}
		}

	case RestoreWizardStepInputs:
		sb.WriteString(StyleFocus.Render("  Restoration Target Parameters") + "\n")
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("  Archive: %s", filepath.Base(m.selectedArchive))) + "\n\n")

		fields := []struct {
			label string
			val   string
			idx   int
		}{
			{"Website Name", m.inputs.WebsiteName, 0},
			{"Website Slug", m.inputs.WebsiteSlug, 1},
			{"Admin Username", m.inputs.AdminUsername, 2},
			{"Admin Password", strings.Repeat("*", len(m.inputs.AdminPassword)), 3},
			{"Admin Email", m.inputs.AdminEmail, 4},
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

		btnCursor := "  "
		btnText := "[ Start Website Restoration ]"
		if m.fieldIndex == 5 {
			btnCursor = StyleFocus.Render("> ")
			btnText = StyleFocus.Render(btnText)
		} else {
			btnText = StyleMuted.Render(btnText)
		}
		sb.WriteString(fmt.Sprintf("\n%s%s\n", btnCursor, btnText))

		if m.validationErr != "" {
			sb.WriteString("\n  " + StyleError.Render("✗ "+m.validationErr) + "\n")
		}
	}

	return sb.String()
}
