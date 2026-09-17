package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/packages"
)

// LiveSearchModel is a unified interactive live search model for package catalog.
// It filters results live as the user types, supports Up/Down navigation, Space to toggle selection,
// preserves selections across query modifications, and finishes on Enter or aborts on Esc/Ctrl+C.
type LiveSearchModel struct {
	itemType      packages.PackageType
	catalog       []packages.CatalogItem
	query         string
	cursor        int
	filtered      []packages.CatalogItem
	selectedMap   map[string]bool
	selectedOrder []string
	submitted     bool
	aborted       bool
}

// NewLiveSearchModel initializes a new live search model with initial selections.
func NewLiveSearchModel(itemType packages.PackageType, catalog []packages.CatalogItem, initialSelected []string) *LiveSearchModel {
	m := &LiveSearchModel{
		itemType:    itemType,
		catalog:     catalog,
		selectedMap: make(map[string]bool),
	}
	for _, s := range initialSelected {
		if !m.selectedMap[s] {
			m.selectedMap[s] = true
			m.selectedOrder = append(m.selectedOrder, s)
		}
	}
	m.updateFilter()
	return m
}

func (m *LiveSearchModel) updateFilter() {
	m.filtered = packages.FilterCatalog(m.catalog, m.itemType, m.query)
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		} else {
			m.cursor = 0
		}
	}
}
func (m *LiveSearchModel) Init() tea.Cmd {
	return nil
}

func (m *LiveSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		str := msg.String()
		switch str {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit

		case "enter":
			m.submitted = true
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case " ", "space":
			// Toggle selection of currently focused item
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				slug := m.filtered[m.cursor].Slug
				if m.selectedMap[slug] {
					delete(m.selectedMap, slug)
				} else {
					m.selectedMap[slug] = true
					m.selectedOrder = append(m.selectedOrder, slug)
				}
			}
			return m, nil

		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.updateFilter()
			}
			return m, nil

		default:
			text := msg.Text
			if text == "" && len(str) == 1 {
				text = str
			}
			if text != "" && !strings.Contains(str, "+") && str != "tab" {
				m.query += text
				m.updateFilter()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *LiveSearchModel) View() tea.View {
	var sb strings.Builder

	cyan := lipgloss.Color("#00FFFF")
	green := lipgloss.Color("#04B575")
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(cyan)
	highlightStyle := lipgloss.NewStyle().Foreground(cyan).Bold(true)
	checkedStyle := lipgloss.NewStyle().Foreground(green).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	sb.WriteString(titleStyle.Render(fmt.Sprintf("🔍 Live %s Catalog Search", strings.Title(string(m.itemType)))))
	sb.WriteString("\n")

	// Summary bar
	var names []string
	for _, s := range m.selectedOrder {
		if m.selectedMap[s] {
			names = append(names, s)
		}
	}
	summaryText := fmt.Sprintf("Selected (%d): %s", len(names), strings.Join(names, ", "))
	if len(names) == 0 {
		summaryText = fmt.Sprintf("Selected: 0 %ss", m.itemType)
	}
	sb.WriteString(dimStyle.Render(summaryText))
	sb.WriteString("\n\n")

	// Query input box
	sb.WriteString(fmt.Sprintf("Search: %s█\n", m.query))
	sb.WriteString(dimStyle.Render("(Type to filter live • Up/Down navigate • Space select • Enter finish • Esc cancel)"))
	sb.WriteString("\n\n")

	// Filtered results list
	if len(m.filtered) == 0 {
		sb.WriteString(dimStyle.Render("  [No matching packages found. Modify query or press Enter to finish / Esc to cancel.]\n"))
	} else {
		maxItems := 10
		start := 0
		if m.cursor >= maxItems {
			start = m.cursor - maxItems + 1
		}
		end := start + maxItems
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := start; i < end; i++ {
			item := m.filtered[i]
			cursorIndicator := "  "
			if i == m.cursor {
				cursorIndicator = "> "
			}

			check := "[ ]"
			if m.selectedMap[item.Slug] {
				check = checkedStyle.Render("[x]")
			}
			itemText := fmt.Sprintf("%s (%s v%s)", item.Name, item.Slug, item.Version)
			if m.selectedMap[item.Slug] {
				// When selected, text renders in vibrant green bold
				itemText = checkedStyle.Render(itemText)
			} else if i == m.cursor {
				// When unselected but cursor is on it, text renders in cyan bold
				itemText = highlightStyle.Render(itemText)
			}

			cursorPrefix := cursorIndicator
			if i == m.cursor {
				cursorPrefix = highlightStyle.Render(cursorIndicator)
			}

			sb.WriteString(fmt.Sprintf("%s%s %s\n", cursorPrefix, check, itemText))
		}
		if len(m.filtered) > maxItems {
			sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more (scroll with Up/Down)\n", len(m.filtered)-maxItems)))
		}
	}
	return tea.NewView(sb.String())
}

// ViewString returns the plain rendered view text for testing.
func (m *LiveSearchModel) ViewString() string {
	return m.View().Content
}

// FinalSelected returns the deduplicated list of selected slugs in deterministic selection order.
func (m *LiveSearchModel) FinalSelected() []string {
	var out []string
	for _, s := range m.selectedOrder {
		if m.selectedMap[s] {
			out = append(out, s)
		}
	}
	return deduplicateStrings(out)
}

// IsAborted returns true if the user cancelled the search.
func (m *LiveSearchModel) IsAborted() bool {
	return m.aborted
}
