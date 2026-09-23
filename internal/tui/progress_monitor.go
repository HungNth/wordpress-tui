package tui

import (
	"fmt"
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
	vpWidth := width - 8
	if vpWidth < 20 {
		vpWidth = 20
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

	case LogLineMsg:
		m.logLines = append(m.logLines, string(msg))
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()

	case OperationCompleteMsg:
		m.completed = true
		m.completedMsg = msg
		m.SetSize(m.width, m.height)

	case tea.KeyPressMsg:
		str := msg.String()
		if m.completed && (str == "enter" || str == "esc") {
			m.finished = true
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

func (m *ProgressMonitorModel) Render(width, height int) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(StyleFocus.Render("  "+m.title) + "\n\n")

	// 2. Stepper
	for _, step := range m.steps {
		var icon string
		title := step.Title

		switch step.Status {
		case StepStatusPending:
			icon = StyleMuted.Render("[ ] ")
			title = StyleMuted.Render(title)
		case StepStatusRunning:
			icon = m.spinner.View() + " "
			title = StyleFocus.Render(title)
		case StepStatusSuccess:
			icon = StyleSuccess.Render("[✓] ")
			title = StyleSuccess.Render(title)
		case StepStatusWarning:
			icon = StyleWarning.Render("[!] ")
			title = StyleWarning.Render(title)
		case StepStatusFailed:
			icon = StyleError.Render("[✗] ")
			title = StyleError.Render(title)
		}

		sb.WriteString("  " + icon + title)
		if step.Detail != "" {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf(" (%s)", step.Detail)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// 3. Log Viewport
	vpBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Width(width - 6)

	logContent := m.viewport.View()
	if logContent == "" {
		logContent = StyleMuted.Render("Waiting for command output...")
	}
	sb.WriteString(vpBorder.Render(StyleMuted.Render(" Logs ") + "\n" + logContent))
	sb.WriteString("\n")

	// 4. Completion summary
	if m.completed {
		if m.completedMsg.Success {
			sb.WriteString(StyleSuccess.Render("  ✓ " + m.completedMsg.Title))
		} else if m.completedMsg.Warning {
			sb.WriteString(StyleWarning.Render("  ! " + m.completedMsg.Title))
		} else {
			sb.WriteString(StyleError.Render("  ✗ " + m.completedMsg.Title))
		}
		sb.WriteString("\n")

		if m.completedMsg.Summary != "" {
			for _, line := range strings.Split(m.completedMsg.Summary, "\n") {
				if strings.HasPrefix(line, "[!]") {
					sb.WriteString("    " + StyleWarning.Render(line) + "\n")
				} else if strings.HasPrefix(line, "[✗]") {
					sb.WriteString("    " + StyleError.Render(line) + "\n")
				} else if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
					key := parts[0] + ":"
					val := strings.TrimSpace(parts[1])
					sb.WriteString("    " + StyleMuted.Render(fmt.Sprintf("%-14s", key)) + " " + StyleHighlight.Render(val) + "\n")
				} else {
					sb.WriteString("    " + StyleMuted.Render(line) + "\n")
				}
			}
		}
		sb.WriteString("\n" + StyleFocus.Render("  Press [ Enter / Esc ] to return") + "\n")
	}

	return sb.String()
}
