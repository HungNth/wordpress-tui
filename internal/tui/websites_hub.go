package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/launcher"
	"wptui/internal/packages"
)

type WebsitesHubState int

const (
	WebsitesHubList WebsitesHubState = iota
	WebsitesHubActions
	WebsitesHubBatchDeleteConfirm
	WebsitesHubSingleDeleteConfirm
	WebsitesHubChangeAdmin
	WebsitesHubInstallPlugins
	WebsitesHubInstallThemes
	WebsitesHubThemeActivateConfirm
	WebsitesHubBackupStrategy
)

type WebsiteAction string

const (
	WebsiteActionConfig         WebsiteAction = "Config"
	WebsiteActionInstallPlugins WebsiteAction = "Install Plugins"
	WebsiteActionInstallThemes  WebsiteAction = "Install Theme"
	WebsiteActionChangeAdmin    WebsiteAction = "Change admin credentials"
	WebsiteActionBackup         WebsiteAction = "Backup"
	WebsiteActionBrowser        WebsiteAction = "Open in Browser"
	WebsiteActionEditor         WebsiteAction = "Open in Editor"
	WebsiteActionDelete         WebsiteAction = "Delete"
	WebsiteActionBack           WebsiteAction = "Back"
)

var DefaultWebsiteActions = []WebsiteAction{
	WebsiteActionConfig,
	WebsiteActionInstallPlugins,
	WebsiteActionInstallThemes,
	WebsiteActionChangeAdmin,
	WebsiteActionBackup,
	WebsiteActionBrowser,
	WebsiteActionEditor,
	WebsiteActionDelete,
	WebsiteActionBack,
}

type WebsitesHubModel struct {
	cfg                  *config.Config
	candidates           []deprovision.Candidate
	cursor               int
	selectedMap          map[string]bool
	state                WebsitesHubState
	actionCursor         int
	actions              []WebsiteAction
	deleteConfirmChoice  bool // false = No, true = Yes
	statusMsg            string
	runner               launcher.ProcessRunner
	catalog              []packages.CatalogItem
	packagePicker        *LiveSearchModel
	themeActivateChoice  bool
	backupStrategyCursor int
	backupStrategies     []backup.BackupStrategy
	adminUsernameInput   string
	adminPasswordInput   string
	adminEmailInput      string
	adminFieldIndex      int // 0: Username, 1: Password, 2: Email, 3: Submit, 4: Cancel
	OnAction             func(action WebsiteAction, cand deprovision.Candidate) tea.Cmd
	OnBatchDelete        func(cands []deprovision.Candidate) tea.Cmd
	OnChangeAdmin        func(cand deprovision.Candidate, username, password, email string) tea.Cmd
	OnInstallPackages    func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool) tea.Cmd
	OnBackup             func(cand deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd
}

func NewWebsitesHubModel(cfg *config.Config) *WebsitesHubModel {
	candidates, _ := deprovision.DiscoverCandidates(context.Background(), cfg.WebsitesPath, cfg.DeleteExcludes)
	return &WebsitesHubModel{
		cfg:                  cfg,
		candidates:           candidates,
		cursor:               0,
		selectedMap:          make(map[string]bool),
		state:                WebsitesHubList,
		actionCursor:         0,
		actions:              DefaultWebsiteActions,
		deleteConfirmChoice:  false,
		backupStrategyCursor: 0,
		backupStrategies:     []backup.BackupStrategy{backup.StrategyFull, backup.StrategyAI1WM},
	}
}

func (m *WebsitesHubModel) Refresh() {
	m.candidates, _ = deprovision.DiscoverCandidates(context.Background(), m.cfg.WebsitesPath, m.cfg.DeleteExcludes)
	if m.cursor >= len(m.candidates) {
		if len(m.candidates) > 0 {
			m.cursor = len(m.candidates) - 1
		} else {
			m.cursor = 0
		}
	}
}

func (m *WebsitesHubModel) siteURL(slug string) string {
	if m.cfg != nil && m.cfg.UsedHerd {
		return fmt.Sprintf("https://%s.test", slug)
	}
	return fmt.Sprintf("http://%s.test", slug)
}

func (m *WebsitesHubModel) SetRunner(r launcher.ProcessRunner) {
	m.runner = r
}

func (m *WebsitesHubModel) SetCatalog(catalog []packages.CatalogItem) {
	m.catalog = catalog
}

func (m *WebsitesHubModel) State() WebsitesHubState {
	return m.state
}

func (m *WebsitesHubModel) SelectedCandidate() deprovision.Candidate {
	if len(m.candidates) == 0 || m.cursor >= len(m.candidates) {
		return deprovision.Candidate{}
	}
	return m.candidates[m.cursor]
}

func (m *WebsitesHubModel) IsSelected(slug string) bool {
	return m.selectedMap[slug]
}

func (m *WebsitesHubModel) SelectedCount() int {
	count := 0
	for _, cand := range m.candidates {
		if m.selectedMap[cand.Slug] {
			count++
		}
	}
	return count
}

func (m *WebsitesHubModel) SelectedCandidates() []deprovision.Candidate {
	var out []deprovision.Candidate
	for _, cand := range m.candidates {
		if m.selectedMap[cand.Slug] {
			out = append(out, cand)
		}
	}
	return out
}

func (m *WebsitesHubModel) DeleteConfirmChoice() bool {
	return m.deleteConfirmChoice
}

func (m *WebsitesHubModel) Init() tea.Cmd {
	return nil
}

func (m *WebsitesHubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()

		switch m.state {
		case WebsitesHubList:
			switch str {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "j":
				if m.cursor < len(m.candidates)-1 {
					m.cursor++
				}
				return m, nil
			case " ", "space":
				if len(m.candidates) > 0 {
					slug := m.candidates[m.cursor].Slug
					if m.selectedMap[slug] {
						delete(m.selectedMap, slug)
					} else {
						m.selectedMap[slug] = true
					}
				}
				return m, nil
			case "a":
				if len(m.candidates) > 0 {
					allSelected := len(m.SelectedCandidates()) == len(m.candidates)
					if allSelected {
						m.selectedMap = make(map[string]bool)
					} else {
						for _, cand := range m.candidates {
							m.selectedMap[cand.Slug] = true
						}
					}
				}
				return m, nil
			case "d":
				if m.SelectedCount() > 0 {
					m.state = WebsitesHubBatchDeleteConfirm
					m.deleteConfirmChoice = false
				}
				return m, nil
			case "enter":
				if m.SelectedCount() > 1 {
					m.state = WebsitesHubBatchDeleteConfirm
					m.deleteConfirmChoice = false
					return m, nil
				}
				if len(m.candidates) > 0 {
					m.state = WebsitesHubActions
					m.actionCursor = 0
					m.statusMsg = ""
				}
				return m, nil
			}

		case WebsitesHubActions:
			switch str {
			case "up", "k":
				if m.actionCursor > 0 {
					m.actionCursor--
				}
				return m, nil
			case "down", "j":
				if m.actionCursor < len(m.actions)-1 {
					m.actionCursor++
				}
				return m, nil
			case "esc":
				m.state = WebsitesHubList
				m.statusMsg = ""
				return m, nil
			case "enter":
				act := m.actions[m.actionCursor]
				cand := m.SelectedCandidate()
				switch act {
				case WebsiteActionBack:
					m.state = WebsitesHubList
					m.statusMsg = ""
					return m, nil
				case WebsiteActionInstallPlugins:
					m.packagePicker = NewLiveSearchModel(packages.PackageTypePlugin, m.catalog, nil)
					m.state = WebsitesHubInstallPlugins
					m.statusMsg = ""
					return m, nil
				case WebsiteActionInstallThemes:
					m.packagePicker = NewLiveSearchModel(packages.PackageTypeTheme, m.catalog, nil)
					m.state = WebsitesHubInstallThemes
					m.statusMsg = ""
					return m, nil
				case WebsiteActionChangeAdmin:
					m.state = WebsitesHubChangeAdmin
					m.adminUsernameInput = ""
					m.adminPasswordInput = ""
					m.adminEmailInput = ""
					if m.cfg != nil {
						m.adminUsernameInput = m.cfg.DefaultAdminUsername
						m.adminPasswordInput = m.cfg.DefaultAdminPassword
						m.adminEmailInput = m.cfg.DefaultAdminEmail
					}
					if m.adminUsernameInput == "" {
						m.adminUsernameInput = "admin"
					}
					if m.adminEmailInput == "" {
						m.adminEmailInput = "admin@example.com"
					}
					m.adminFieldIndex = 0
					m.statusMsg = ""
					return m, nil
				case WebsiteActionBackup:
					m.state = WebsitesHubBackupStrategy
					m.backupStrategyCursor = 0
					m.statusMsg = ""
					return m, nil
				case WebsiteActionDelete:
					m.state = WebsitesHubSingleDeleteConfirm
					m.deleteConfirmChoice = false
					return m, nil
				case WebsiteActionBrowser:
					url := m.siteURL(cand.Slug)
					err := launcher.OpenURL(context.Background(), url, m.runner)
					if err != nil {
						m.statusMsg = StyleError.Render(fmt.Sprintf("Failed to open browser: %v", err))
					} else {
						m.statusMsg = StyleSuccess.Render("Opened " + url + " in browser")
					}
					return m, nil
				case WebsiteActionEditor:
					err := launcher.OpenInEditorNonBlocking(context.Background(), cand.Path, m.runner)
					if err != nil {
						m.statusMsg = StyleError.Render(fmt.Sprintf("Failed to launch editor: %v", err))
					} else {
						m.statusMsg = StyleSuccess.Render("Opened " + cand.Path + " in editor")
					}
					return m, nil
				default:
					if m.OnAction != nil {
						return m, m.OnAction(act, cand)
					}
					return m, nil
				}
			}

		case WebsitesHubBatchDeleteConfirm:
			switch str {
			case "left", "right", "tab":
				m.deleteConfirmChoice = !m.deleteConfirmChoice
				return m, nil
			case "esc":
				m.state = WebsitesHubList
				return m, nil
			case "enter":
				if m.deleteConfirmChoice && m.OnBatchDelete != nil {
					return m, m.OnBatchDelete(m.SelectedCandidates())
				}
				m.state = WebsitesHubList
				return m, nil
			}

		case WebsitesHubSingleDeleteConfirm:
			switch str {
			case "left", "right", "tab":
				m.deleteConfirmChoice = !m.deleteConfirmChoice
				return m, nil
			case "esc":
				m.state = WebsitesHubActions
				return m, nil
			case "enter":
				if m.deleteConfirmChoice && m.OnAction != nil {
					return m, m.OnAction(WebsiteActionDelete, m.SelectedCandidate())
				}
				m.state = WebsitesHubActions
				return m, nil
			}

		case WebsitesHubChangeAdmin:
			switch str {
			case "esc":
				m.state = WebsitesHubActions
				m.statusMsg = ""
				return m, nil
			case "tab", "down":
				m.adminFieldIndex = (m.adminFieldIndex + 1) % 5
				return m, nil
			case "shift+tab", "backtab", "up":
				m.adminFieldIndex = (m.adminFieldIndex - 1 + 5) % 5
				return m, nil
			case "left":
				if m.adminFieldIndex == 4 {
					m.adminFieldIndex = 3
					return m, nil
				}
			case "right":
				if m.adminFieldIndex == 3 {
					m.adminFieldIndex = 4
					return m, nil
				}
			case "enter":
				if m.adminFieldIndex == 4 { // Cancel button
					m.state = WebsitesHubActions
					m.statusMsg = ""
					return m, nil
				}
				if m.adminFieldIndex < 3 {
					m.adminFieldIndex++
					return m, nil
				}
				// Submit (index 3)
				if strings.TrimSpace(m.adminUsernameInput) == "" {
					m.statusMsg = StyleError.Render("Username cannot be empty")
					m.adminFieldIndex = 0
					return m, nil
				}
				if strings.TrimSpace(m.adminPasswordInput) == "" {
					m.statusMsg = StyleError.Render("Password cannot be empty")
					m.adminFieldIndex = 1
					return m, nil
				}
				if strings.TrimSpace(m.adminEmailInput) == "" {
					m.statusMsg = StyleError.Render("Email cannot be empty")
					m.adminFieldIndex = 2
					return m, nil
				}
				cand := m.SelectedCandidate()
				u, p, e := m.adminUsernameInput, m.adminPasswordInput, m.adminEmailInput
				m.state = WebsitesHubList
				m.statusMsg = ""
				if m.OnChangeAdmin != nil {
					return m, m.OnChangeAdmin(cand, u, p, e)
				}
				return m, nil
			case "backspace":
				m.deleteLastAdminChar()
				return m, nil
			case " ", "space":
				if m.adminFieldIndex < 3 {
					m.appendAdminChar(" ")
				}
				return m, nil
			default:
				text := msg.Text
				if text == "" && len(str) == 1 {
					text = str
				}
				if text != "" && !strings.Contains(str, "+") && m.adminFieldIndex < 3 {
					m.appendAdminChar(text)
				}
				return m, nil
			}

		case WebsitesHubInstallPlugins:
			switch str {
			case "esc":
				m.state = WebsitesHubActions
				m.packagePicker = nil
				return m, nil
			case "enter":
				if m.packagePicker != nil {
					selected := m.packagePicker.FinalSelected()
					if len(selected) == 0 {
						m.statusMsg = StyleWarning.Render("No plugins selected")
						m.state = WebsitesHubActions
						m.packagePicker = nil
						return m, nil
					}
					cand := m.SelectedCandidate()
					m.state = WebsitesHubList
					m.packagePicker = nil
					if m.OnInstallPackages != nil {
						return m, m.OnInstallPackages(cand, packages.PackageTypePlugin, selected, true)
					}
				}
				m.state = WebsitesHubActions
				return m, nil
			default:
				if m.packagePicker != nil {
					newM, cmd := m.packagePicker.Update(msg)
					m.packagePicker = newM.(*LiveSearchModel)
					return m, cmd
				}
				return m, nil
			}

		case WebsitesHubInstallThemes:
			switch str {
			case "esc":
				m.state = WebsitesHubActions
				m.packagePicker = nil
				return m, nil
			case "enter":
				if m.packagePicker != nil {
					selected := m.packagePicker.FinalSelected()
					if len(selected) == 0 {
						m.statusMsg = StyleWarning.Render("No themes selected")
						m.state = WebsitesHubActions
						m.packagePicker = nil
						return m, nil
					}
					m.themeActivateChoice = true
					m.state = WebsitesHubThemeActivateConfirm
					return m, nil
				}
				m.state = WebsitesHubActions
				return m, nil
			default:
				if m.packagePicker != nil {
					newM, cmd := m.packagePicker.Update(msg)
					m.packagePicker = newM.(*LiveSearchModel)
					return m, cmd
				}
				return m, nil
			}

		case WebsitesHubThemeActivateConfirm:
			switch str {
			case "left", "right", "tab":
				m.themeActivateChoice = !m.themeActivateChoice
				return m, nil
			case "esc":
				m.state = WebsitesHubInstallThemes
				return m, nil
			case "enter":
				var selected []string
				if m.packagePicker != nil {
					selected = m.packagePicker.FinalSelected()
				}
				cand := m.SelectedCandidate()
				activate := m.themeActivateChoice
				m.state = WebsitesHubList
				m.packagePicker = nil
				if m.OnInstallPackages != nil {
					return m, m.OnInstallPackages(cand, packages.PackageTypeTheme, selected, activate)
				}
				return m, nil
			}

		case WebsitesHubBackupStrategy:
			switch str {
			case "up", "k":
				if m.backupStrategyCursor > 0 {
					m.backupStrategyCursor--
				}
				return m, nil
			case "down", "j":
				if m.backupStrategyCursor < 2 {
					m.backupStrategyCursor++
				}
				return m, nil
			case "esc":
				m.state = WebsitesHubActions
				return m, nil
			case "enter":
				if m.backupStrategyCursor == 2 { // Back
					m.state = WebsitesHubActions
					return m, nil
				}
				cand := m.SelectedCandidate()
				strategy := m.backupStrategies[m.backupStrategyCursor]
				m.state = WebsitesHubList
				if m.OnBackup != nil {
					return m, m.OnBackup(cand, strategy)
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *WebsitesHubModel) appendAdminChar(c string) {
	switch m.adminFieldIndex {
	case 0:
		m.adminUsernameInput += c
	case 1:
		m.adminPasswordInput += c
	case 2:
		m.adminEmailInput += c
	}
}

func (m *WebsitesHubModel) deleteLastAdminChar() {
	switch m.adminFieldIndex {
	case 0:
		if len(m.adminUsernameInput) > 0 {
			m.adminUsernameInput = m.adminUsernameInput[:len(m.adminUsernameInput)-1]
		}
	case 1:
		if len(m.adminPasswordInput) > 0 {
			m.adminPasswordInput = m.adminPasswordInput[:len(m.adminPasswordInput)-1]
		}
	case 2:
		if len(m.adminEmailInput) > 0 {
			m.adminEmailInput = m.adminEmailInput[:len(m.adminEmailInput)-1]
		}
	}
}

func (m *WebsitesHubModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func (m *WebsitesHubModel) Render(width, height int) string {
	var sb strings.Builder

	if len(m.candidates) == 0 {
		return fmt.Sprintf("\n  No websites found in %s\n\n  Press 'Create' in sidebar to provision your first website.", m.cfg.WebsitesPath)
	}

	switch m.state {
	case WebsitesHubList:
		sb.WriteString(fmt.Sprintf("  Websites (%d total, %d selected)\n\n", len(m.candidates), m.SelectedCount()))
		for i, cand := range m.candidates {
			cursor := "  "
			if i == m.cursor {
				cursor = StyleFocus.Render("> ")
			}

			check := "[ ]"
			if m.selectedMap[cand.Slug] {
				check = StyleSuccess.Render("[x]")
			}

			itemTitle := cand.Slug
			if i == m.cursor {
				itemTitle = StyleFocus.Render(itemTitle)
			}

			urlText := StyleMuted.Render(m.siteURL(cand.Slug))
			pathText := StyleMuted.Render(cand.Path)

			sb.WriteString(fmt.Sprintf("%s%s %s\n      URL:  %s\n      Path: %s\n\n", cursor, check, itemTitle, urlText, pathText))
		}

	case WebsitesHubActions:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render("  Website Details") + "\n")
		sb.WriteString(fmt.Sprintf("    Name/Slug: %s\n", cand.Slug))
		sb.WriteString(fmt.Sprintf("    URL:       %s\n", m.siteURL(cand.Slug)))
		sb.WriteString(fmt.Sprintf("    Directory: %s\n\n", cand.Path))

		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("  Select Action:") + "\n")
		for i, act := range m.actions {
			cursor := "  "
			label := string(act)
			if i == m.actionCursor {
				cursor = StyleFocus.Render("> ")
				label = StyleFocus.Render(label)
			}
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, label))
		}

		if m.statusMsg != "" {
			sb.WriteString("\n  " + m.statusMsg + "\n")
		}

	case WebsitesHubBatchDeleteConfirm:
		selected := m.SelectedCandidates()
		sb.WriteString(StyleError.Render("  ⚠ Delete Selected Websites?") + "\n\n")
		sb.WriteString("  The following websites and their databases will be permanently removed:\n")
		for _, s := range selected {
			sb.WriteString(fmt.Sprintf("    • %s (%s)\n", s.Slug, s.Path))
		}
		sb.WriteString("\n")

		yesStyle := StyleMuted
		noStyle := StyleMuted
		if m.deleteConfirmChoice {
			yesStyle = StyleError
		} else {
			noStyle = StyleFocus
		}

		sb.WriteString(fmt.Sprintf("  Confirm deletion:  %s    %s\n\n",
			yesStyle.Render("[ Yes, Delete All ]"),
			noStyle.Render("[ No, Cancel ]")))
		sb.WriteString(StyleMuted.Render("  Use Left/Right to toggle · Enter to confirm · Esc to abort"))

	case WebsitesHubSingleDeleteConfirm:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleError.Render(fmt.Sprintf("  ⚠ Delete Website %s?", cand.Slug)) + "\n\n")
		sb.WriteString(fmt.Sprintf("  Directory: %s\n", cand.Path))
		sb.WriteString(fmt.Sprintf("  URL:       %s\n\n", m.siteURL(cand.Slug)))

		yesStyle := StyleMuted
		noStyle := StyleMuted
		if m.deleteConfirmChoice {
			yesStyle = StyleError
		} else {
			noStyle = StyleFocus
		}

		sb.WriteString(fmt.Sprintf("  Confirm deletion:  %s    %s\n\n",
			yesStyle.Render("[ Yes, Delete ]"),
			noStyle.Render("[ No, Cancel ]")))
		sb.WriteString(StyleMuted.Render("  Use Left/Right to toggle · Enter to confirm · Esc to abort"))

	case WebsitesHubChangeAdmin:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render("  Change Administrator Credentials") + "\n")
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("  Website: %s (%s)", cand.Slug, cand.Path)) + "\n\n")

		fields := []struct {
			label string
			val   string
			idx   int
		}{
			{"New Username", m.adminUsernameInput, 0},
			{"New Password", strings.Repeat("•", len(m.adminPasswordInput)), 1},
			{"New Email", m.adminEmailInput, 2},
		}

		for _, f := range fields {
			cursor := "  "
			label := fmt.Sprintf("%-16s", f.label)
			val := f.val

			if f.idx == m.adminFieldIndex {
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
		sb.WriteString("\n")

		submitBtn := "[ Update Credentials ]"
		cancelBtn := "[ Cancel ]"

		if m.adminFieldIndex == 3 {
			submitBtn = StyleFocus.Render(submitBtn)
			cancelBtn = StyleMuted.Render(cancelBtn)
		} else if m.adminFieldIndex == 4 {
			submitBtn = StyleMuted.Render(submitBtn)
			cancelBtn = StyleFocus.Render(cancelBtn)
		} else {
			submitBtn = StyleMuted.Render(submitBtn)
			cancelBtn = StyleMuted.Render(cancelBtn)
		}

		sb.WriteString(fmt.Sprintf("    %s    %s\n\n", submitBtn, cancelBtn))
		if m.statusMsg != "" {
			sb.WriteString("  " + m.statusMsg + "\n\n")
		}
		sb.WriteString(StyleMuted.Render("  Tab/Down: Next · Shift+Tab/Up: Prev · Enter: Confirm · Esc: Cancel"))

	case WebsitesHubInstallPlugins:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render(fmt.Sprintf("  Install Plugins: %s", cand.Slug)) + "\n")
		sb.WriteString(StyleMuted.Render("  Search catalog · Space to toggle selection · Enter to install · Esc to cancel") + "\n\n")
		if m.packagePicker != nil {
			sb.WriteString(m.packagePicker.ViewString())
		}

	case WebsitesHubInstallThemes:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render(fmt.Sprintf("  Install Themes: %s", cand.Slug)) + "\n")
		sb.WriteString(StyleMuted.Render("  Search catalog · Space to toggle selection · Enter to confirm · Esc to cancel") + "\n\n")
		if m.packagePicker != nil {
			sb.WriteString(m.packagePicker.ViewString())
		}

	case WebsitesHubThemeActivateConfirm:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render("  Activate Theme after installation?") + "\n\n")
		sb.WriteString(fmt.Sprintf("  Website: %s (%s)\n\n", cand.Slug, cand.Path))
		yesStyle := StyleMuted
		noStyle := StyleMuted
		if m.themeActivateChoice {
			yesStyle = StyleSuccess
		} else {
			noStyle = StyleFocus
		}
		sb.WriteString(fmt.Sprintf("  Activate theme:  %s    %s\n\n",
			yesStyle.Render("[ Yes, Activate ]"),
			noStyle.Render("[ No, Install Only ]")))
		sb.WriteString(StyleMuted.Render("  Use Left/Right to toggle · Enter to confirm · Esc to go back"))

	case WebsitesHubBackupStrategy:
		cand := m.SelectedCandidate()
		sb.WriteString(StyleFocus.Render("  Backup Website: "+cand.Slug) + "\n")
		sb.WriteString(fmt.Sprintf("    Directory: %s\n", cand.Path))
		sb.WriteString(fmt.Sprintf("    URL:       %s\n\n", m.siteURL(cand.Slug)))

		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("  Select Backup Format:") + "\n")
		opts := []string{
			"Full source code & database (.zip)",
			"All-in-One WP Migration (.wpress)",
			"Back",
		}
		for i, opt := range opts {
			cursor := "  "
			label := opt
			if i == m.backupStrategyCursor {
				cursor = StyleFocus.Render("> ")
				label = StyleFocus.Render(label)
			}
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, label))
		}
		sb.WriteString("\n" + StyleMuted.Render("  Use Up/Down to navigate · Enter to confirm · Esc to go back"))
	}

	return sb.String()
}
