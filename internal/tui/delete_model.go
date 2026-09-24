package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/deprovision"
)

type DeleteModelState int

const (
	DeleteModelList DeleteModelState = iota
	DeleteModelConfirm
)

// DeleteModel provides a dedicated batch de-provisioning view in the Content Pane.
type DeleteModel struct {
	cfg           *config.Config
	candidates    []deprovision.Candidate
	cursor        int
	scrollOffset  int
	selectedMap   map[string]bool
	state         DeleteModelState
	confirmChoice bool // false = [ No, Cancel ], true = [ Yes, Delete All ]
	warningMsg    string
	OnDelete      func(candidates []deprovision.Candidate) tea.Cmd
}

func NewDeleteModel(cfg *config.Config) *DeleteModel {
	var candidates []deprovision.Candidate
	if cfg != nil {
		candidates, _ = deprovision.DiscoverCandidates(context.Background(), cfg.WebsitesPath, cfg.DeleteExcludes)
	}
	return &DeleteModel{
		cfg:           cfg,
		candidates:    candidates,
		cursor:        0,
		scrollOffset:  0,
		selectedMap:   make(map[string]bool),
		state:         DeleteModelList,
		confirmChoice: false,
	}
}

func (m *DeleteModel) Refresh() {
	if m.cfg != nil {
		m.candidates, _ = deprovision.DiscoverCandidates(context.Background(), m.cfg.WebsitesPath, m.cfg.DeleteExcludes)
	}
	if m.cursor >= len(m.candidates) {
		if len(m.candidates) > 0 {
			m.cursor = len(m.candidates) - 1
		} else {
			m.cursor = 0
		}
	}
	if m.scrollOffset > m.cursor {
		m.scrollOffset = m.cursor
	}
}

func (m *DeleteModel) clampScroll(maxVisible int) {
	if maxVisible <= 0 {
		return
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+maxVisible {
		m.scrollOffset = m.cursor - maxVisible + 1
	}
	total := len(m.candidates)
	if total > maxVisible && m.scrollOffset > total-maxVisible {
		m.scrollOffset = total - maxVisible
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *DeleteModel) ResetSelection() {
	m.selectedMap = make(map[string]bool)
	m.state = DeleteModelList
	m.confirmChoice = false
	m.warningMsg = ""
}

func (m *DeleteModel) State() DeleteModelState {
	return m.state
}

func (m *DeleteModel) Candidates() []deprovision.Candidate {
	return m.candidates
}

func (m *DeleteModel) Cursor() int {
	return m.cursor
}

func (m *DeleteModel) ConfirmChoice() bool {
	return m.confirmChoice
}

func (m *DeleteModel) IsSelected(slug string) bool {
	return m.selectedMap[slug]
}

func (m *DeleteModel) SelectedCount() int {
	count := 0
	for _, cand := range m.candidates {
		if m.selectedMap[cand.Slug] {
			count++
		}
	}
	return count
}

func (m *DeleteModel) SelectedCandidates() []deprovision.Candidate {
	var out []deprovision.Candidate
	for _, cand := range m.candidates {
		if m.selectedMap[cand.Slug] {
			out = append(out, cand)
		}
	}
	return out
}

func (m *DeleteModel) siteURL(slug string) string {
	if m.cfg != nil && m.cfg.UsedHerd {
		return fmt.Sprintf("https://%s.test", slug)
	}
	return fmt.Sprintf("http://%s.test", slug)
}

func (m *DeleteModel) Init() tea.Cmd {
	return nil
}

func (m *DeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()

		switch m.state {
		case DeleteModelList:
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
					m.warningMsg = ""
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
					m.warningMsg = ""
				}
				return m, nil
			case "enter":
				if m.SelectedCount() > 0 {
					m.state = DeleteModelConfirm
					m.confirmChoice = false
					m.warningMsg = ""
				} else {
					m.warningMsg = "No websites selected. Press Space to select at least one."
				}
				return m, nil
			}

		case DeleteModelConfirm:
			switch str {
			case "left", "right", "tab":
				m.confirmChoice = !m.confirmChoice
				return m, nil
			case "esc":
				m.state = DeleteModelList
				return m, nil
			case "enter":
				if m.confirmChoice && m.OnDelete != nil {
					return m, m.OnDelete(m.SelectedCandidates())
				}
				m.state = DeleteModelList
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *DeleteModel) View() tea.View {
	return tea.NewView(m.Render(80, 24))
}

func (m *DeleteModel) Render(contentWidth, bodyHeight int) string {
	var sb strings.Builder

	switch m.state {
	case DeleteModelList:
		sb.WriteString(StyleFocus.Render("  Batch Website De-provisioning") + "\n")
		sb.WriteString(StyleMuted.Render("  Select websites to delete permanently · Space Toggle · a All · Enter Proceed") + "\n\n")

		if len(m.candidates) == 0 {
			path := ""
			if m.cfg != nil {
				path = m.cfg.WebsitesPath
			}
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("  No local WordPress websites found in %s\n", path)))
			return sb.String()
		}

		maxItems := bodyHeight - 9
		if maxItems < 3 {
			maxItems = 3
		}
		m.clampScroll(maxItems)
		start := m.scrollOffset
		end := start + maxItems
		if end > len(m.candidates) {
			end = len(m.candidates)
		}

		if start > 0 {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("    ↑ %d more above\n", start)))
		}

		for i := start; i < end; i++ {
			cand := m.candidates[i]
			cursor := "  "
			if i == m.cursor {
				cursor = StyleFocus.Render("> ")
			}

			check := "[ ]"
			if m.selectedMap[cand.Slug] {
				check = StyleSuccess.Render("[x]")
			}

			slugWidth := 20
			slugStr := cand.Slug
			if len(slugStr) < slugWidth {
				slugStr = slugStr + strings.Repeat(" ", slugWidth-len(slugStr))
			}
			if i == m.cursor {
				slugStr = StyleFocus.Render(slugStr)
			}

			urlText := StyleMuted.Render(m.siteURL(cand.Slug))

			sb.WriteString(fmt.Sprintf("%s%s %s  %s\n", cursor, check, slugStr, urlText))
		}

		if end < len(m.candidates) {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("    ↓ %d more below\n", len(m.candidates)-end)))
		}

		if m.warningMsg != "" {
			sb.WriteString("\n  " + StyleWarning.Render(m.warningMsg) + "\n")
		}

	case DeleteModelConfirm:
		selected := m.SelectedCandidates()
		sb.WriteString(StyleError.Render("  ⚠ Delete Selected Websites?") + "\n\n")
		sb.WriteString("  The following websites and their databases will be permanently removed:\n")
		maxPreview := 6
		for i, s := range selected {
			if i >= maxPreview {
				sb.WriteString(StyleMuted.Render(fmt.Sprintf("    ... and %d more website(s)\n", len(selected)-maxPreview)))
				break
			}
			sb.WriteString(fmt.Sprintf("    • %s (%s)\n", s.Slug, s.Path))
		}
		sb.WriteString("\n")

		yesStyle := StyleMuted
		noStyle := StyleMuted
		if m.confirmChoice {
			yesStyle = StyleError
		} else {
			noStyle = StyleFocus
		}

		sb.WriteString(fmt.Sprintf("  Confirm deletion:  %s    %s\n\n",
			yesStyle.Render("[ Yes, Delete All ]"),
			noStyle.Render("[ No, Cancel ]")))
		sb.WriteString(StyleMuted.Render("  Use Left/Right to toggle · Enter to confirm · Esc to abort"))
	}

	return sb.String()
}
