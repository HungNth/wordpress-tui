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
	maxLineWidth := width - 4
	if maxLineWidth < 30 {
		maxLineWidth = 30
	}

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

		firstLineMax := maxLineWidth - 6
		if firstLineMax < 15 {
			firstLineMax = 15
		}

		lines := wrapWords(rawTitle, firstLineMax)
		if len(lines) == 0 {
			lines = []string{""}
		}

		sb.WriteString("  " + icon + titleStyle.Render(lines[0]) + "\n")
		for _, nextLine := range lines[1:] {
			sb.WriteString("      " + titleStyle.Render(nextLine) + "\n")
		}
	}
	sb.WriteString("\n")

	// 3. Log Viewport
	boxWidth := width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	if width-2 > 0 && boxWidth > width-2 {
		boxWidth = width - 2
	}

	vpBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1).
		Width(boxWidth)
	logContent := m.viewport.View()
	if logContent == "" {
		logContent = StyleMuted.Render("Waiting for command output...")
	}
	renderedBox := vpBorder.Render(StyleMuted.Render("Logs") + "\n" + logContent)
	if width > 2 {
		sb.WriteString(lipgloss.PlaceHorizontal(width-2, lipgloss.Center, renderedBox))
	} else {
		sb.WriteString(renderedBox)
	}
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
					rawLines := wrapWords(line, maxLineWidth-4)
					for _, rl := range rawLines {
						sb.WriteString("    " + StyleWarning.Render(rl) + "\n")
					}
				} else if strings.HasPrefix(line, "[✗]") {
					rawLines := wrapWords(line, maxLineWidth-4)
					for _, rl := range rawLines {
						sb.WriteString("    " + StyleError.Render(rl) + "\n")
					}
				} else if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
					key := parts[0] + ":"
					val := strings.TrimSpace(parts[1])
					valMaxWidth := maxLineWidth - 19
					if valMaxWidth < 15 {
						valMaxWidth = 15
					}
					valLines := wrapWords(val, valMaxWidth)
					if len(valLines) == 0 {
						valLines = []string{""}
					}
					sb.WriteString("    " + StyleMuted.Render(fmt.Sprintf("%-14s", key)) + " " + StyleHighlight.Render(valLines[0]) + "\n")
					valIndent := strings.Repeat(" ", 19)
					for _, nextVal := range valLines[1:] {
						sb.WriteString(valIndent + StyleHighlight.Render(nextVal) + "\n")
					}
				} else {
					rawLines := wrapWords(line, maxLineWidth-4)
					for _, rl := range rawLines {
						sb.WriteString("    " + StyleMuted.Render(rl) + "\n")
					}
				}
			}
		}
		sb.WriteString("\n" + StyleFocus.Render("  Press [ Enter / Esc ] to return") + "\n")
	}

	return sb.String()
}
