package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

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
	cfg                 *config.Config
	checker             SlugAvailabilityChecker
	step                RestoreWizardStep
	strategy            restore.Strategy
	formatCursor        int
	formatOptions       []RestoreFormatOption
	archives            []string
	archiveCursor       int
	archiveScrollOffset int
	scanErr             string
	archiveErr          string
	enteringCustomPath  bool
	customArchivePath   string
	selectedArchive     string
	selectedFromCustom  bool
	fieldIndex          int
	inputs              CreateInputs
	cleanInputs         CreateInputs
	validationErr       string
	confirmingDiscard   bool
	submitted           bool
	shouldReturn        bool
}

func (m *RestoreWizardModel) clampArchiveScroll(maxVisible int) {
	if maxVisible <= 0 {
		return
	}
	if m.archiveCursor < m.archiveScrollOffset {
		m.archiveScrollOffset = m.archiveCursor
	} else if m.archiveCursor >= m.archiveScrollOffset+maxVisible {
		m.archiveScrollOffset = m.archiveCursor - maxVisible + 1
	}
	total := len(m.archives)
	if total > maxVisible && m.archiveScrollOffset > total-maxVisible {
		m.archiveScrollOffset = total - maxVisible
	}
	if m.archiveScrollOffset < 0 {
		m.archiveScrollOffset = 0
	}
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
		cleanInputs:   initialInputs,
		submitted:     false,
		shouldReturn:  false,
	}
}

func (m *RestoreWizardModel) IsEnteringCustomPath() bool {
	return m.step == RestoreWizardStepArchive && m.enteringCustomPath
}

func (m *RestoreWizardModel) IsConfirmingDiscard() bool {
	return m.confirmingDiscard
}

func (m *RestoreWizardModel) IsDirty() bool {
	return strings.TrimSpace(m.customArchivePath) != "" || m.inputs != m.cleanInputs
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

func sanitizeSingleLinePaste(raw string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(raw)
}

func extractPrintableInput(msg tea.KeyPressMsg) (string, bool) {
	if msg.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModSuper|tea.ModMeta) != 0 {
		return "", false
	}
	str := msg.String()
	if str == " " || str == "space" {
		return " ", true
	}
	if msg.Text != "" {
		return msg.Text, true
	}
	if utf8.RuneCountInString(str) == 1 {
		return str, true
	}
	return "", false
}

func (m *RestoreWizardModel) loadArchivesForStrategy() {
	scanned, err := ScanBackupArchives(m.cfg.BackupPath, m.strategy)
	if err != nil {
		m.scanErr = err.Error()
	} else {
		m.scanErr = ""
	}
	m.archives = append([]string{CustomPathOption}, scanned...)
	m.archiveCursor = 0
	m.archiveScrollOffset = 0
	m.archiveErr = ""
	m.enteringCustomPath = false
}

func (m *RestoreWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		pasted := sanitizeSingleLinePaste(msg.Content)
		if pasted == "" {
			return m, nil
		}
		if m.step == RestoreWizardStepArchive && m.enteringCustomPath {
			m.customArchivePath += pasted
			return m, nil
		}
		if m.step == RestoreWizardStepInputs && m.fieldIndex < 5 {
			m.appendChar(pasted)
			return m, nil
		}
		return m, nil

	case tea.KeyPressMsg:
		str := msg.String()

		if m.confirmingDiscard {
			switch str {
			case "y", "Y":
				m.customArchivePath = ""
				m.selectedArchive = ""
				m.selectedFromCustom = false
				m.enteringCustomPath = false
				m.archiveErr = ""
				m.validationErr = ""
				m.inputs = m.cleanInputs
				m.confirmingDiscard = false
				m.step = RestoreWizardStepFormat
				m.shouldReturn = true
				return m, nil
			case "n", "N", "esc":
				m.confirmingDiscard = false
				return m, nil
			}
			return m, nil
		}

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
			case "shift+tab", "backtab":
				m.shouldReturn = true
				return m, nil
			case "esc":
				if m.IsDirty() {
					m.confirmingDiscard = true
				} else {
					m.shouldReturn = true
				}
				return m, nil
			case "enter":
				m.strategy = m.formatOptions[m.formatCursor].Strategy
				m.loadArchivesForStrategy()
				m.step = RestoreWizardStepArchive
				return m, nil
			}

		case RestoreWizardStepArchive:
			if m.enteringCustomPath {
				switch str {
				case "esc":
					m.enteringCustomPath = false
					m.archiveErr = ""
					return m, nil
				case "enter":
					cleanPath, err := NormalizeAndValidateArchivePath(m.customArchivePath, m.strategy)
					if err != nil {
						m.archiveErr = err.Error()
						return m, nil
					}
					m.customArchivePath = cleanPath
					m.selectedArchive = cleanPath
					m.selectedFromCustom = true
					m.enteringCustomPath = false
					m.archiveErr = ""
					m.step = RestoreWizardStepInputs
					return m, nil
				case "backspace":
					m.customArchivePath = deleteLastRune(m.customArchivePath)
					return m, nil
				case "ctrl+u":
					m.customArchivePath = ""
					return m, nil
				default:
					if text, ok := extractPrintableInput(msg); ok {
						m.customArchivePath += text
					}
					return m, nil
				}
			}

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
				m.archiveErr = ""
				m.step = RestoreWizardStepFormat
				return m, nil
			case "enter":
				if len(m.archives) == 0 {
					return m, nil
				}
				if m.archiveCursor == 0 {
					m.enteringCustomPath = true
					m.archiveErr = ""
					return m, nil
				}
				choice := m.archives[m.archiveCursor]
				cleanPath, err := NormalizeAndValidateArchivePath(choice, m.strategy)
				if err != nil {
					m.archiveErr = err.Error()
					return m, nil
				}
				m.selectedArchive = cleanPath
				m.selectedFromCustom = false
				m.archiveErr = ""
				m.step = RestoreWizardStepInputs
				return m, nil
			}

		case RestoreWizardStepInputs:
			switch str {
			case "esc":
				m.validationErr = ""
				m.step = RestoreWizardStepArchive
				m.enteringCustomPath = m.selectedFromCustom
				return m, nil
			case "tab", "down":
				m.fieldIndex = (m.fieldIndex + 1) % 6
				return m, nil
			case "shift+tab", "backtab", "up":
				m.fieldIndex = (m.fieldIndex - 1 + 6) % 6
				return m, nil
			case "enter":
				if m.fieldIndex == 5 { // Submit button
					cleanArchive, err := NormalizeAndValidateArchivePath(m.selectedArchive, m.strategy)
					if err != nil {
						m.validationErr = ""
						m.archiveErr = err.Error()
						m.step = RestoreWizardStepArchive
						m.enteringCustomPath = m.selectedFromCustom
						return m, nil
					}
					resolvedSlug, err := ResolveAndValidateSlug(m.inputs.WebsiteName, m.inputs.WebsiteSlug, m.checker)
					if err != nil {
						m.validationErr = err.Error()
						return m, nil
					}
					m.selectedArchive = cleanArchive
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
				if text, ok := extractPrintableInput(msg); ok {
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
		m.inputs.WebsiteName = deleteLastRune(m.inputs.WebsiteName)
	case 1:
		m.inputs.WebsiteSlug = deleteLastRune(m.inputs.WebsiteSlug)
	case 2:
		m.inputs.AdminUsername = deleteLastRune(m.inputs.AdminUsername)
	case 3:
		m.inputs.AdminPassword = deleteLastRune(m.inputs.AdminPassword)
	case 4:
		m.inputs.AdminEmail = deleteLastRune(m.inputs.AdminEmail)
	}
}

func (m *RestoreWizardModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func renderWrappedPrefixed(sb *strings.Builder, indent, prefix string, style lipgloss.Style, text string, width int) {
	prefixWidth := utf8.RuneCountInString(indent) + utf8.RuneCountInString(prefix)
	avail := max(width-prefixWidth-2, 14)
	lines := wrapDisplayText(text, avail)
	if len(lines) == 0 {
		return
	}
	sb.WriteString(indent + style.Render(prefix+lines[0]) + "\n")
	hang := strings.Repeat(" ", prefixWidth)
	for _, line := range lines[1:] {
		sb.WriteString(hang + style.Render(line) + "\n")
	}
}

func renderWrappedPath(sb *strings.Builder, firstIndent, contIndent string, style lipgloss.Style, text string, width int) {
	contWidth := utf8.RuneCountInString(contIndent)
	avail := max(width-contWidth-2, 14)
	lines := wrapDisplayText(firstIndent+text, avail)
	if len(lines) == 0 {
		return
	}
	sb.WriteString(style.Render(lines[0]) + "\n")
	for _, line := range lines[1:] {
		sb.WriteString(contIndent + style.Render(line) + "\n")
	}
}

func (m *RestoreWizardModel) Render(width, height int) string {
	var sb strings.Builder
	innerWidth := max(width-4, 20)
	compact := height <= 22
	if m.confirmingDiscard {
		sb.WriteString("\n  " + StyleWarning.Render("[!] Discard changes? (y/n)") + "\n\n")
		sb.WriteString(StyleMuted.Render("  You have unsaved Restore Wizard inputs.\n  Press 'y' to discard or 'n' to keep editing."))
		return sb.String()
	}

	switch m.step {
	case RestoreWizardStepFormat:
		sb.WriteString(StyleFocus.Render("  Restore Website") + "\n")
		sb.WriteString(StyleMuted.Render("  Choose the Restore Archive format") + "\n\n")

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
		if m.enteringCustomPath {
			sb.WriteString(StyleFocus.Render("  Enter Restore Archive Path") + "\n")
			extHint, _ := expectedArchiveExtension(m.strategy)
			if extHint == "" {
				extHint = ".zip"
			}
			desc := fmt.Sprintf("Enter or paste an absolute %s path (outer quotes allowed)", extHint)
			for _, line := range wrapDisplayText(desc, innerWidth) {
				sb.WriteString("  " + StyleMuted.Render(line) + "\n")
			}
			if !compact {
				sb.WriteString("\n")
			}

			sb.WriteString(StyleFocus.Render("> Archive Path:") + "\n")
			fieldStyle := lipgloss.NewStyle().Foreground(ColorCyan).Underline(true)
			if m.customArchivePath == "" {
				placeholder := "e.g. D:\\backups\\website" + extHint
				sb.WriteString("    " + fieldStyle.Render("_") + " " + StyleMuted.Render(placeholder) + "\n")
			} else {
				valLines := wrapDisplayText(m.customArchivePath+"_", max(innerWidth-2, 14))
				maxValLines := max(height-12, 3)
				if len(valLines) > maxValLines {
					valLines = valLines[len(valLines)-maxValLines:]
				}
				for _, line := range valLines {
					sb.WriteString("    " + fieldStyle.Render(line) + "\n")
				}
			}

			if m.archiveErr != "" {
				if !compact {
					sb.WriteString("\n")
				}
				renderWrappedPrefixed(&sb, "  ", "[✗] ", StyleError, m.archiveErr, width)
			}
			return strings.TrimRight(sb.String(), "\n")
		}

		sb.WriteString(StyleFocus.Render("  Select Restore Archive") + "\n")
		if m.scanErr != "" {
			renderWrappedPrefixed(&sb, "  ", "[!] ", StyleWarning, m.scanErr, width)
		} else {
			for _, line := range wrapDisplayText(fmt.Sprintf("Backup directory: %s", m.cfg.BackupPath), innerWidth) {
				sb.WriteString("  " + StyleMuted.Render(line) + "\n")
			}
		}
		if !compact {
			sb.WriteString("\n")
		}

		var errBlock strings.Builder
		if m.archiveErr != "" {
			if !compact {
				errBlock.WriteString("\n")
			}
			renderWrappedPrefixed(&errBlock, "  ", "[✗] ", StyleError, m.archiveErr, width)
		}

		usedLines := strings.Count(sb.String(), "\n") + strings.Count(errBlock.String(), "\n")
		if m.scanErr == "" && len(m.archives) == 1 {
			usedLines += 2
		}
		maxItems := max(height-usedLines-5, 2)
		m.clampArchiveScroll(maxItems)
		start := m.archiveScrollOffset
		end := min(start+maxItems, len(m.archives))

		if start > 0 {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("    ↑ %d more above\n", start)))
		}

		maxItemRunes := max(width-6, 16)
		for i := start; i < end; i++ {
			arch := m.archives[i]
			cursor := "  "
			display := arch
			if i > 0 {
				display = filepath.Base(arch)
			}
			if utf8.RuneCountInString(display) > maxItemRunes {
				runes := []rune(display)
				display = string(runes[:maxItemRunes-3]) + "..."
			}
			if i == m.archiveCursor {
				cursor = StyleFocus.Render("> ")
				display = StyleFocus.Render(display)
			} else {
				display = StyleMuted.Render(display)
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", cursor, display))
		}

		if end < len(m.archives) {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("    ↓ %d more below\n", len(m.archives)-end)))
		}

		if m.scanErr == "" && len(m.archives) == 1 {
			sb.WriteString("\n  " + StyleMuted.Render("No matching archives in backup directory.") + "\n")
		}

		sb.WriteString(errBlock.String())

	case RestoreWizardStepInputs:
		sb.WriteString(StyleFocus.Render("  Restoration Target Parameters") + "\n")
		renderWrappedPath(&sb, "  Archive: ", "    ", StyleMuted, m.selectedArchive, width)
		if !compact {
			sb.WriteString("\n")
		}

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
			} else if val == "" {
				val = StyleMuted.Render("(empty)")
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
		if !compact {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", btnCursor, btnText))

		if m.validationErr != "" {
			if !compact {
				sb.WriteString("\n")
			}
			renderWrappedPrefixed(&sb, "  ", "[✗] ", StyleError, m.validationErr, width)
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
