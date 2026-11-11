package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusRunning
	StepStatusSuccess
	StepStatusWarning
	StepStatusFailed
)

type ProgressStep struct {
	ID     string
	Title  string
	Status StepStatus
	Detail string
}

type StepStartMsg struct {
	ID    string
	Title string
}

type StepCompleteMsg struct {
	ID     string
	Status StepStatus
	Detail string
}

type LogLineMsg string

type OperationCompleteMsg struct {
	Title   string
	Success bool
	Warning bool
	Summary string
}

type ProgressResetStepsMsg struct {
	Title string
	Steps []ProgressStep
}

type ProgressMonitorModel struct {
	title        string
	steps        []ProgressStep
	spinner      spinner.Model
	viewport     viewport.Model
	logLines     []string
	completed    bool
	completedMsg OperationCompleteMsg
	finished     bool
	width        int
	height       int
	scrollOffset int
	focusSummary bool
}

func NewProgressMonitorModel(title string, steps []ProgressStep) *ProgressMonitorModel {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorCyan)),
	)
	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(8))

	return &ProgressMonitorModel{
		title:     title,
		steps:     steps,
		spinner:   s,
		viewport:  vp,
		completed: false,
		finished:  false,
		width:     70,
		height:    24,
	}
}

func (m *ProgressMonitorModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	overhead := len(m.steps) + 6
	if m.completed {
		summaryLines := 0
		if m.completedMsg.Summary != "" {
			summaryLines = len(strings.Split(m.completedMsg.Summary, "\n"))
		}
		overhead = len(m.steps) + summaryLines + 8
	}

	vpHeight := height - overhead - 4
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 10
	if vpWidth < 18 {
		vpWidth = 18
	}
	m.viewport = viewport.New(viewport.WithWidth(vpWidth), viewport.WithHeight(vpHeight))
	if len(m.logLines) > 0 {
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()
	}
}

func (m *ProgressMonitorModel) IsCompleted() bool {
	return m.completed
}

func (m *ProgressMonitorModel) IsFinished() bool {
	return m.finished
}

func (m *ProgressMonitorModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ProgressMonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case StepStartMsg:
		for i := range m.steps {
			if m.steps[i].ID == msg.ID {
				m.steps[i].Status = StepStatusRunning
				if msg.Title != "" {
					m.steps[i].Title = msg.Title
				}
				break
			}
		}

	case StepCompleteMsg:
		for i := range m.steps {
			if m.steps[i].ID == msg.ID {
				m.steps[i].Status = msg.Status
				m.steps[i].Detail = msg.Detail
				break
			}
		}


	case ProgressResetStepsMsg:
		if msg.Title != "" {
			m.title = msg.Title
		}
		m.steps = slices.Clone(msg.Steps)
		m.SetSize(m.width, m.height)
	case LogLineMsg:
		m.logLines = append(m.logLines, string(msg))
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()

	case OperationCompleteMsg:
		m.completed = true
		m.completedMsg = msg
		m.focusSummary = true
		m.scrollOffset = 0
		m.SetSize(m.width, m.height)

	case tea.KeyPressMsg:
		str := msg.String()
		if m.completed && (str == "enter" || str == "esc") {
			m.finished = true
			return m, nil
		}
		if m.completed && m.handleCompletedScroll(str) {
			return m, nil
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ProgressMonitorModel) View() tea.View {
	return tea.NewView(m.Render(m.width, m.height))
}

func (m *ProgressMonitorModel) handleCompletedScroll(key string) bool {
	switch key {
	case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
	default:
		return false
	}
	allLines, summaryStart := m.buildLines(m.width, m.height)
	if m.height <= 0 || len(allLines) <= m.height {
		return false
	}
	maxOffset := len(allLines) - m.height
	cur := m.effectiveOffset(len(allLines), summaryStart, m.height)
	page := max(1, m.height/2)
	switch key {
	case "up", "k":
		cur--
	case "down", "j":
		cur++
	case "pgup":
		cur -= page
	case "pgdown":
		cur += page
	case "home":
		cur = 0
	case "end":
		cur = maxOffset
	}
	m.focusSummary = false
	m.scrollOffset = max(0, min(cur, maxOffset))
	return true
}

func (m *ProgressMonitorModel) effectiveOffset(totalLines, summaryStart, height int) int {
	if height <= 0 || totalLines <= height || !m.completed {
		return 0
	}
	maxOffset := totalLines - height
	if m.focusSummary {
		return min(summaryStart, maxOffset)
	}
	return max(0, min(m.scrollOffset, maxOffset))
}

func (m *ProgressMonitorModel) Render(width, height int) string {
	allLines, summaryStart := m.buildLines(width, height)
	if height > 0 && len(allLines) > height {
		offset := m.effectiveOffset(len(allLines), summaryStart, height)
		allLines = allLines[offset : offset+height]
	}
	return strings.Join(allLines, "\n")
}

func (m *ProgressMonitorModel) buildLines(width, height int) ([]string, int) {
	maxLineWidth := max(18, width-4)

	var stepperSB strings.Builder
	titleLines := wrapDisplayText(m.title, maxLineWidth)
	if len(titleLines) == 0 {
		titleLines = []string{""}
	}
	for _, tl := range titleLines {
		stepperSB.WriteString(StyleFocus.Render("  "+tl) + "\n")
	}
	stepperSB.WriteString("\n")

	firstLineMax := max(12, maxLineWidth-6)
	for _, step := range m.steps {
		var icon string
		var titleStyle lipgloss.Style

		switch step.Status {
		case StepStatusPending:
			icon = StyleMuted.Render("[ ] ")
			titleStyle = StyleMuted
		case StepStatusRunning:
			icon = m.spinner.View() + " "
			titleStyle = StyleFocus
		case StepStatusSuccess:
			icon = StyleSuccess.Render("[✓] ")
			titleStyle = StyleSuccess
		case StepStatusWarning:
			icon = StyleWarning.Render("[!] ")
			titleStyle = StyleWarning
		case StepStatusFailed:
			icon = StyleError.Render("[✗] ")
			titleStyle = StyleError
		}

		rawTitle := step.Title
		if step.Detail != "" {
			rawTitle += fmt.Sprintf(" (%s)", step.Detail)
		}

		lines := wrapDisplayText(rawTitle, firstLineMax)
		if len(lines) == 0 {
			lines = []string{""}
		}

		stepperSB.WriteString("  " + icon + titleStyle.Render(lines[0]) + "\n")
		for _, nextLine := range lines[1:] {
			stepperSB.WriteString("      " + titleStyle.Render(nextLine) + "\n")
		}
	}
	stepperLines := strings.Split(strings.TrimRight(stepperSB.String(), "\n"), "\n")
	stepperLines = append(stepperLines, "")

	var compLines []string
	if m.completed {
		var compSB strings.Builder
		headingLines := wrapDisplayText(m.completedMsg.Title, max(12, maxLineWidth-4))
		if len(headingLines) == 0 {
			headingLines = []string{""}
		}
		prefix := "  ✗ "
		headingStyle := StyleError
		if m.completedMsg.Success {
			prefix = "  ✓ "
			headingStyle = StyleSuccess
		} else if m.completedMsg.Warning {
			prefix = "  ! "
			headingStyle = StyleWarning
		}
		compSB.WriteString(headingStyle.Render(prefix+headingLines[0]) + "\n")
		for _, hl := range headingLines[1:] {
			compSB.WriteString(headingStyle.Render("    "+hl) + "\n")
		}

		if m.completedMsg.Summary != "" {
			rowWidth := max(12, maxLineWidth-4)
			detailWidth := max(12, maxLineWidth-6)
			for _, line := range strings.Split(m.completedMsg.Summary, "\n") {
				if strings.HasPrefix(line, "[✓]") {
					for _, rl := range wrapDisplayText(line, rowWidth) {
						compSB.WriteString("    " + StyleSuccess.Render(rl) + "\n")
					}
				} else if strings.HasPrefix(line, "[-]") {
					for _, rl := range wrapDisplayText(line, rowWidth) {
						compSB.WriteString("    " + StyleMuted.Render(rl) + "\n")
					}
				} else if strings.HasPrefix(line, "[!]") {
					for _, rl := range wrapDisplayText(line, rowWidth) {
						compSB.WriteString("    " + StyleWarning.Render(rl) + "\n")
					}
				} else if strings.HasPrefix(line, "[✗]") {
					for _, rl := range wrapDisplayText(line, rowWidth) {
						compSB.WriteString("    " + StyleError.Render(rl) + "\n")
					}
				} else if strings.HasPrefix(line, "  ") {
					for _, rl := range wrapDisplayText(strings.TrimSpace(line), detailWidth) {
						compSB.WriteString("      " + StyleMuted.Render(rl) + "\n")
					}
				} else if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
					key := parts[0] + ":"
					val := strings.TrimSpace(parts[1])
					valLines := wrapDisplayText(val, max(12, maxLineWidth-19))
					if len(valLines) == 0 {
						valLines = []string{""}
					}
					compSB.WriteString("    " + StyleMuted.Render(fmt.Sprintf("%-14s", key)) + " " + StyleHighlight.Render(valLines[0]) + "\n")
					valIndent := strings.Repeat(" ", 19)
					for _, nextVal := range valLines[1:] {
						compSB.WriteString(valIndent + StyleHighlight.Render(nextVal) + "\n")
					}
				} else {
					for _, rl := range wrapDisplayText(line, rowWidth) {
						compSB.WriteString("    " + StyleMuted.Render(rl) + "\n")
					}
				}
			}
		}
		compSB.WriteString("\n" + StyleFocus.Render("  Press [ Enter / Esc ] to return"))
		compLines = strings.Split(compSB.String(), "\n")
	}

	boxWidth := max(20, width-6)
	if width-2 > 0 && boxWidth > width-2 {
		boxWidth = width - 2
	}
	vpBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1).
		Width(boxWidth)
	if !m.completed && height > 0 {
		availLogLines := max(1, height-len(stepperLines)-3)
		if m.viewport.Height() != availLogLines {
			wasAtBottom := m.viewport.AtBottom()
			m.viewport.SetHeight(availLogLines)
			if wasAtBottom {
				m.viewport.GotoBottom()
			}
		}
	}
	logContent := m.viewport.View()
	if strings.TrimSpace(logContent) == "" && len(m.logLines) == 0 {
		logContent = StyleMuted.Render("Waiting for command output...")
	}
	renderedBox := vpBorder.Render(StyleMuted.Render("Logs") + "\n" + logContent)
	if width > 2 {
		renderedBox = lipgloss.PlaceHorizontal(width-2, lipgloss.Center, renderedBox)
	}
	boxLines := strings.Split(strings.TrimRight(renderedBox, "\n"), "\n")

	allLines := make([]string, 0, len(stepperLines)+len(boxLines)+len(compLines))
	allLines = append(allLines, stepperLines...)
	allLines = append(allLines, boxLines...)
	summaryStart := len(allLines)
	allLines = append(allLines, compLines...)
	return allLines, summaryStart
}
