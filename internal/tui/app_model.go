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
	"wptui/internal/restore"
)

type CreateHandlerFunc func(inputs CreateInputs, packages []string) tea.Cmd
type RestoreHandlerFunc func(strategy restore.Strategy, archivePath string, inputs CreateInputs) tea.Cmd
type BackupHandlerFunc func(candidate deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd
type DeleteHandlerFunc func(candidates []deprovision.Candidate) tea.Cmd
type ConfigHandlerFunc func(candidate deprovision.Candidate) tea.Cmd
type ChangeAdminHandlerFunc func(cand deprovision.Candidate, username, password, email string) tea.Cmd
type InstallPackagesHandlerFunc func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool) tea.Cmd

type FocusMode int

const (
	FocusSidebar FocusMode = iota
	FocusContent
)

type Section int

const (
	SectionWebsites Section = iota
	SectionCreate
	SectionDelete
	SectionRestore
	SectionSettings
	SectionExit
)

const (
	MinTerminalWidth  = 80
	MinTerminalHeight = 24
	SidebarWidth      = 28
)

type SidebarItem struct {
	Section     Section
	Title       string
	Description string
}

var DefaultSidebarItems = []SidebarItem{
	{Section: SectionWebsites, Title: "Websites", Description: "Local WordPress websites hub"},
	{Section: SectionCreate, Title: "Create", Description: "Provision a new WordPress site"},
	{Section: SectionDelete, Title: "Delete", Description: "Batch de-provision local websites"},
	{Section: SectionRestore, Title: "Restore", Description: "Restore site from backup archive"},
	{Section: SectionSettings, Title: "Settings", Description: "Application settings & maintenance"},
	{Section: SectionExit, Title: "Exit", Description: "Exit WPTUI"},
}

// AppModel is the root Bubble Tea application model for WPTUI.
type AppModel struct {
	cfg             *config.Config
	width           int
	height          int
	focus           FocusMode
	activeSec       Section
	sidebarIndex    int
	sidebarItems    []SidebarItem
	websitesHub     *WebsitesHubModel
	deleteModel     *DeleteModel
	createWizard    *CreateWizardModel
	restoreWizard   *RestoreWizardModel
	settingsModel   *SettingsModel
	progressMonitor *ProgressMonitorModel
	isExecuting     bool
	catalog         []packages.CatalogItem
	checker         SlugAvailabilityChecker
	opEventChan     <-chan tea.Msg
	onCreate        CreateHandlerFunc
	onRestore       RestoreHandlerFunc
	onBackup        BackupHandlerFunc
	onDelete        DeleteHandlerFunc
	onConfig        ConfigHandlerFunc
	onChangeAdmin   ChangeAdminHandlerFunc
	onInstallPackages InstallPackagesHandlerFunc
}

func NewAppModel(cfg *config.Config) *AppModel {
	m := &AppModel{
		cfg:             cfg,
		width:           100,
		height:          30,
		focus:           FocusSidebar,
		activeSec:       SectionWebsites,
		sidebarIndex:    0,
		sidebarItems:    DefaultSidebarItems,
		websitesHub:     NewWebsitesHubModel(cfg),
		deleteModel:     NewDeleteModel(cfg),
		createWizard:    NewCreateWizardModel(cfg, nil, nil),
		restoreWizard:   NewRestoreWizardModel(cfg, nil),
		settingsModel:   NewSettingsModel(cfg, "", nil),
		progressMonitor: nil,
		isExecuting:     false,
	}
	m.wireHubCallbacks()
	m.wireDeleteCallbacks()
	return m
}

func (m *AppModel) wireDeleteCallbacks() {
	if m.deleteModel == nil {
		return
	}
	m.deleteModel.OnDelete = func(cands []deprovision.Candidate) tea.Cmd {
		if m.onDelete != nil {
			return m.onDelete(cands)
		}
		return nil
	}
}

func (m *AppModel) wireHubCallbacks() {
	if m.websitesHub == nil {
		return
	}
	m.websitesHub.OnAction = func(act WebsiteAction, cand deprovision.Candidate) tea.Cmd {
		switch act {
		case WebsiteActionBackup:
			if m.onBackup != nil {
				return m.onBackup(cand, backup.StrategyFull)
			}
		case WebsiteActionDelete:
			if m.onDelete != nil {
				return m.onDelete([]deprovision.Candidate{cand})
			}
		case WebsiteActionConfig:
			if m.onConfig != nil {
				return m.onConfig(cand)
			}
		}
		return nil
	}
	m.websitesHub.OnBackup = func(cand deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd {
		if m.onBackup != nil {
			return m.onBackup(cand, strategy)
		}
		return nil
	}
	m.websitesHub.OnBatchDelete = func(cands []deprovision.Candidate) tea.Cmd {
		if m.onDelete != nil {
			return m.onDelete(cands)
		}
		return nil
	}
	m.websitesHub.OnChangeAdmin = func(cand deprovision.Candidate, username, password, email string) tea.Cmd {
		if m.onChangeAdmin != nil {
			return m.onChangeAdmin(cand, username, password, email)
		}
		return nil
	}
	m.websitesHub.OnInstallPackages = func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool) tea.Cmd {
		if m.onInstallPackages != nil {
			return m.onInstallPackages(cand, pkgType, slugs, activate)
		}
		return nil
	}
}

func (m *AppModel) WebsitesHub() *WebsitesHubModel {
	return m.websitesHub
}

func (m *AppModel) DeleteModel() *DeleteModel {
	return m.deleteModel
}

func (m *AppModel) CreateWizard() *CreateWizardModel {
	return m.createWizard
}

func (m *AppModel) RestoreWizard() *RestoreWizardModel {
	return m.restoreWizard
}

func (m *AppModel) SettingsModel() *SettingsModel {
	return m.settingsModel
}

func (m *AppModel) SetCreateDependencies(catalog []packages.CatalogItem, checker SlugAvailabilityChecker) {
	m.catalog = catalog
	m.checker = checker
	m.createWizard = NewCreateWizardModel(m.cfg, catalog, checker)
	if m.websitesHub != nil {
		m.websitesHub.SetCatalog(catalog)
	}
}

func (m *AppModel) SetRestoreDependencies(checker SlugAvailabilityChecker) {
	m.checker = checker
	m.restoreWizard = NewRestoreWizardModel(m.cfg, checker)
}

func (m *AppModel) SetSettingsDependencies(cfgPath string, runner launcher.ProcessRunner, cacheDir ...string) {
	m.settingsModel = NewSettingsModel(m.cfg, cfgPath, runner, cacheDir...)
}

func (m *AppModel) SetCreateHandler(fn CreateHandlerFunc) {
	m.onCreate = fn
}

func (m *AppModel) SetRestoreHandler(fn RestoreHandlerFunc) {
	m.onRestore = fn
}

func (m *AppModel) SetBackupHandler(fn BackupHandlerFunc) {
	m.onBackup = fn
}

func (m *AppModel) SetDeleteHandler(fn DeleteHandlerFunc) {
	m.onDelete = fn
	m.wireDeleteCallbacks()
}

func (m *AppModel) SetConfigHandler(fn ConfigHandlerFunc) {
	m.onConfig = fn
}

func (m *AppModel) SetChangeAdminHandler(fn ChangeAdminHandlerFunc) {
	m.onChangeAdmin = fn
}

func (m *AppModel) SetInstallPackagesHandler(fn InstallPackagesHandlerFunc) {
	m.onInstallPackages = fn
}

func (m *AppModel) StartProgress(title string, steps []ProgressStep) tea.Cmd {
	m.progressMonitor = NewProgressMonitorModel(title, steps)
	m.progressMonitor.SetSize(m.width-SidebarWidth-4, m.height-4)
	m.isExecuting = true
	m.focus = FocusContent
	return m.progressMonitor.Init()
}

func (m *AppModel) StartProgressWithChannel(title string, steps []ProgressStep, ch <-chan tea.Msg) tea.Cmd {
	m.opEventChan = ch
	cmd := m.StartProgress(title, steps)
	return tea.Batch(cmd, waitForOpEvent(ch))
}

func waitForOpEvent(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *AppModel) IsExecuting() bool {
	return m.isExecuting
}

func (m *AppModel) ProgressMonitor() *ProgressMonitorModel {
	return m.progressMonitor
}

func (m *AppModel) Focus() FocusMode {
	return m.focus
}

func (m *AppModel) ActiveSection() Section {
	return m.activeSec
}

func (m *AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.progressMonitor != nil {
			m.progressMonitor.SetSize(m.width-SidebarWidth-4, m.height-4)
		}
		return m, nil

	case StepStartMsg, StepCompleteMsg, LogLineMsg, OperationCompleteMsg:
		if m.isExecuting && m.progressMonitor != nil {
			newPM, _ := m.progressMonitor.Update(msg)
			m.progressMonitor = newPM.(*ProgressMonitorModel)
			if m.opEventChan != nil {
				return m, waitForOpEvent(m.opEventChan)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		str := msg.String()
		if str == "ctrl+c" {
			return m, tea.Quit
		}

		if m.isExecuting && m.progressMonitor != nil {
			newPM, cmd := m.progressMonitor.Update(msg)
			m.progressMonitor = newPM.(*ProgressMonitorModel)
			if m.progressMonitor.IsFinished() {
				m.isExecuting = false
				m.progressMonitor = nil
				m.opEventChan = nil
				if m.websitesHub != nil {
					m.websitesHub.Refresh()
				}
				if m.deleteModel != nil {
					m.deleteModel.Refresh()
					m.deleteModel.ResetSelection()
				}
				if m.activeSec == SectionCreate {
					m.createWizard = NewCreateWizardModel(m.cfg, m.catalog, m.checker)
					m.focus = FocusSidebar
				}
				if m.activeSec == SectionRestore {
					m.restoreWizard = NewRestoreWizardModel(m.cfg, m.checker)
					m.focus = FocusSidebar
				}
				if m.activeSec == SectionDelete {
					m.focus = FocusSidebar
				}
			}
			return m, cmd
		}

		switch m.focus {
		case FocusSidebar:
			switch str {
			case "q":
				return m, tea.Quit
			case "up", "k":
				if m.sidebarIndex > 0 {
					m.sidebarIndex--
					m.activeSec = m.sidebarItems[m.sidebarIndex].Section
				}
				return m, nil
			case "down", "j":
				if m.sidebarIndex < len(m.sidebarItems)-1 {
					m.sidebarIndex++
					m.activeSec = m.sidebarItems[m.sidebarIndex].Section
				}
				return m, nil
			case "enter":
				if m.activeSec == SectionExit {
					return m, tea.Quit
				}
				m.focus = FocusContent
				return m, nil
			case "right", "tab":
				if m.activeSec != SectionExit {
					m.focus = FocusContent
				}
				return m, nil
			}

		case FocusContent:
			if m.activeSec == SectionWebsites && m.websitesHub != nil {
				if m.websitesHub.State() == WebsitesHubList && (str == "esc" || str == "shift+tab" || str == "backtab") {
					m.focus = FocusSidebar
					return m, nil
				}
				newHub, cmd := m.websitesHub.Update(msg)
				m.websitesHub = newHub.(*WebsitesHubModel)
				return m, cmd
			}

			if m.activeSec == SectionCreate && m.createWizard != nil {
				newW, cmd := m.createWizard.Update(msg)
				m.createWizard = newW.(*CreateWizardModel)
				if m.createWizard.ShouldExitToSidebar() {
					m.focus = FocusSidebar
					m.createWizard.shouldExitToSidebar = false
				}
				if m.createWizard.IsSubmitted() && m.onCreate != nil {
					return m, m.onCreate(m.createWizard.Inputs(), m.createWizard.SelectedPackages())
				}
				return m, cmd
			}

			if m.activeSec == SectionDelete && m.deleteModel != nil {
				if m.deleteModel.State() == DeleteModelList && (str == "esc" || str == "shift+tab" || str == "backtab") {
					m.focus = FocusSidebar
					return m, nil
				}
				newDM, cmd := m.deleteModel.Update(msg)
				m.deleteModel = newDM.(*DeleteModel)
				return m, cmd
			}

			if m.activeSec == SectionRestore && m.restoreWizard != nil {
				newRW, cmd := m.restoreWizard.Update(msg)
				m.restoreWizard = newRW.(*RestoreWizardModel)
				if m.restoreWizard.ShouldReturn() {
					m.focus = FocusSidebar
					m.restoreWizard.shouldReturn = false
				}
				if m.restoreWizard.IsSubmitted() && m.onRestore != nil {
					return m, m.onRestore(m.restoreWizard.Strategy(), m.restoreWizard.SelectedArchive(), m.restoreWizard.Inputs())
				}
				return m, cmd
			}

			if m.activeSec == SectionSettings && m.settingsModel != nil {
				newSM, cmd := m.settingsModel.Update(msg)
				m.settingsModel = newSM.(*SettingsModel)
				if m.settingsModel.ShouldReturn() {
					m.focus = FocusSidebar
					m.settingsModel.shouldReturn = false
				}
				return m, cmd
			}

			switch str {
			case "esc", "left", "shift+tab", "backtab":
				m.focus = FocusSidebar
				return m, nil
			}
		}

	default:
		if m.isExecuting && m.progressMonitor != nil {
			newPM, cmd := m.progressMonitor.Update(msg)
			m.progressMonitor = newPM.(*ProgressMonitorModel)
			if m.progressMonitor.IsFinished() {
				m.isExecuting = false
				m.progressMonitor = nil
				m.opEventChan = nil
				if m.websitesHub != nil {
					m.websitesHub.Refresh()
				}
				if m.deleteModel != nil {
					m.deleteModel.Refresh()
					m.deleteModel.ResetSelection()
				}
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *AppModel) View() tea.View {
	if m.width < MinTerminalWidth || m.height < MinTerminalHeight {
		return tea.NewView(fmt.Sprintf("\n  Terminal window too small (minimum %dx%d required). Current: %dx%d\n  Please resize your terminal window.\n",
			MinTerminalWidth, MinTerminalHeight, m.width, m.height))
	}

	// 1. Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	headerText := headerStyle.Render("  WPTUI — WordPress Local Manager")

	// 2. Dimensions calculation
	bodyHeight := m.height - 4
	if bodyHeight < 10 {
		bodyHeight = 10
	}
	contentWidth := m.width - SidebarWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// 3. Sidebar pane
	var sb strings.Builder
	for i, item := range m.sidebarItems {
		cursor := "  "
		if i == m.sidebarIndex {
			cursor = "> "
		}

		itemText := fmt.Sprintf("%s%-10s", cursor, item.Title)
		if i == m.sidebarIndex {
			if m.focus == FocusSidebar && !m.isExecuting {
				sb.WriteString(StyleFocus.Render(itemText))
			} else {
				sb.WriteString(StyleMuted.Render(itemText))
			}
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(itemText))
		}
		sb.WriteString("\n")
	}

	sidebarBoxStyle := lipgloss.NewStyle().
		Width(SidebarWidth).
		Height(bodyHeight).
		Border(lipgloss.RoundedBorder())

	if m.focus == FocusSidebar && !m.isExecuting {
		sidebarBoxStyle = sidebarBoxStyle.
			BorderForeground(ColorCyan).
			BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	} else {
		sidebarBoxStyle = sidebarBoxStyle.
			BorderForeground(ColorMuted).
			BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	}

	sidebarTitle := " Navigation "
	if m.focus == FocusSidebar && !m.isExecuting {
		sidebarTitle = StyleFocus.Render(sidebarTitle)
	} else {
		sidebarTitle = StyleMuted.Render(sidebarTitle)
	}

	sidebarBox := sidebarBoxStyle.Render(sidebarTitle + "\n\n" + sb.String())

	// 4. Content pane
	contentBoxStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(bodyHeight).
		Border(lipgloss.RoundedBorder())

	if m.focus == FocusContent {
		contentBoxStyle = contentBoxStyle.
			BorderForeground(ColorCyan).
			BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	} else {
		contentBoxStyle = contentBoxStyle.
			BorderForeground(ColorMuted).
			BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	}

	activeItemTitle := m.sidebarItems[m.sidebarIndex].Title
	contentTitle := fmt.Sprintf(" %s ", activeItemTitle)
	if m.focus == FocusContent {
		contentTitle = StyleFocus.Render(contentTitle)
	} else {
		contentTitle = StyleMuted.Render(contentTitle)
	}

	var contentBody string
	if m.isExecuting && m.progressMonitor != nil {
		contentBody = m.progressMonitor.Render(contentWidth, bodyHeight)
	} else if m.activeSec == SectionWebsites && m.websitesHub != nil {
		contentBody = m.websitesHub.Render(contentWidth, bodyHeight)
	} else if m.activeSec == SectionCreate && m.createWizard != nil {
		contentBody = m.createWizard.Render(contentWidth, bodyHeight)
	} else if m.activeSec == SectionDelete && m.deleteModel != nil {
		contentBody = m.deleteModel.Render(contentWidth, bodyHeight)
	} else if m.activeSec == SectionRestore && m.restoreWizard != nil {
		contentBody = m.restoreWizard.Render(contentWidth, bodyHeight)
	} else if m.activeSec == SectionSettings && m.settingsModel != nil {
		contentBody = m.settingsModel.Render(contentWidth, bodyHeight)
	} else {
		contentBody = fmt.Sprintf("\n  %s\n\n  Press Esc to return to sidebar.", StyleMuted.Render(m.sidebarItems[m.sidebarIndex].Description))
	}
	contentBox := contentBoxStyle.Render(contentTitle + "\n\n" + contentBody)

	// Horizontal layout
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, "  ", contentBox)

	// 5. Contextual Footer
	var footerHelp string
	if m.isExecuting {
		if m.progressMonitor != nil && m.progressMonitor.IsCompleted() {
			footerHelp = "  Press [ Enter / Esc ] to return"
		} else {
			footerHelp = "  Operation running... · Viewport: Up/Down/PgUp/PgDown scroll"
		}
	} else if m.focus == FocusSidebar {
		footerHelp = "  Up/Down (j/k) Navigate · Enter Select · Tab Focus Content · q Exit"
	} else {
		if m.activeSec == SectionWebsites && m.websitesHub != nil {
			switch m.websitesHub.State() {
			case WebsitesHubList:
				footerHelp = "  Up/Down (j/k) Navigate · Space Select · a Toggle All · Enter Details/Delete · Esc Sidebar"
			case WebsitesHubActions:
				footerHelp = "  Up/Down (j/k) Select Action · Enter Run · Esc Back to List"
			default:
				footerHelp = "  Left/Right Toggle Choice · Enter Confirm · Esc Cancel"
			}
		} else if m.activeSec == SectionCreate && m.createWizard != nil {
			switch m.createWizard.Step() {
			case CreateWizardStepInputs:
				footerHelp = "  Tab/Shift+Tab Navigate · Space Toggle · Enter Next · Esc Sidebar"
			case CreateWizardStepPackages:
				footerHelp = "  Up/Down Navigate · Space Select · Enter Confirm · Esc Back to Form"
			default:
				footerHelp = "  y Discard Changes · n Keep Editing"
			}
		} else if m.activeSec == SectionDelete && m.deleteModel != nil {
			switch m.deleteModel.State() {
			case DeleteModelList:
				footerHelp = "  Up/Down (j/k) Navigate · Space Select · a Toggle All · Enter Proceed · Esc Sidebar"
			default:
				footerHelp = "  Left/Right Toggle Choice · Enter Confirm · Esc Cancel"
			}
		} else if m.activeSec == SectionRestore && m.restoreWizard != nil {
			switch m.restoreWizard.Step() {
			case RestoreWizardStepFormat:
				footerHelp = "  Up/Down (j/k) Format · Enter Next · Esc Sidebar"
			case RestoreWizardStepArchive:
				footerHelp = "  Up/Down (j/k) Archive · Enter Next · Esc Back to Format"
			default:
				footerHelp = "  Tab/Shift+Tab Navigate · Enter Submit · Esc Back to Archive"
			}
		} else if m.activeSec == SectionSettings && m.settingsModel != nil {
			footerHelp = "  Up/Down (j/k) Option · Enter Run · Esc Sidebar"
		} else {
			footerHelp = "  Esc Back to Sidebar · Shift+Tab Focus Sidebar"
		}
	}
	footerText := StyleMuted.Render(footerHelp)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, headerText, body, footerText))
	v.AltScreen = true
	return v
}

func (m *AppModel) ViewString() string {
	return m.View().Content
}

// RunAppModel executes the root Bubble Tea application in full-screen alternate-screen buffer.
func RunAppModel(ctx context.Context, cfg *config.Config) error {
	m := NewAppModel(cfg)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
