package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tjmiller/mux/internal/config"
)

type PickerModel struct {
	presets  []config.Preset
	filtered []config.Preset
	cursor   int
	input    textinput.Model
	selected *config.Preset
	quitting bool
	empty    bool
	width    int
	height   int
}

func NewPickerModel(presets []config.Preset) PickerModel {
	if len(presets) == 0 {
		return PickerModel{empty: true}
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.Prompt = "  > "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	return PickerModel{
		presets:  presets,
		filtered: presets,
		input:    ti,
	}
}

func (m PickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.empty {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				p := m.filtered[m.cursor]
				m.selected = &p
				return m, tea.Quit
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	prevQuery := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if m.input.Value() != prevQuery {
		m.filtered = filterPresets(m.presets, m.input.Value())
		m.cursor = 0
	}

	return m, cmd
}

func (m PickerModel) View() string {
	if m.empty {
		style := lipgloss.NewStyle().MarginLeft(2).MarginTop(1)
		return style.Render("No presets configured. Run 'mux create' to create one.\n\nPress q to quit.")
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).MarginLeft(2).MarginTop(1)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	selectedName := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)

	var s strings.Builder
	s.WriteString(titleStyle.Render("Select a preset"))
	s.WriteString("\n\n")
	s.WriteString(m.input.View())
	s.WriteString("\n\n")

	if len(m.filtered) == 0 {
		s.WriteString("    ")
		s.WriteString(dimStyle.Render("No matching presets"))
		s.WriteString("\n")
	} else {
		maxVisible := 10
		if m.height > 0 {
			maxVisible = m.height - 8
			if maxVisible < 3 {
				maxVisible = 3
			}
		}

		start := 0
		if m.cursor >= maxVisible {
			start = m.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := start; i < end; i++ {
			p := m.filtered[i]
			cursor := "  "
			name := nameStyle.Render(p.DisplayName)
			if i == m.cursor {
				cursor = cursorStyle.Render("> ")
				name = selectedName.Render(p.DisplayName)
			}
			tag := dimStyle.Render(fmt.Sprintf("[%s]", p.Name))
			summary := strings.ReplaceAll(config.PresetSummary(&p), "\n", "\n      ")
			s.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, name, tag))
			s.WriteString(fmt.Sprintf("      %s\n", summary))
		}
	}

	s.WriteString("\n")
	s.WriteString(hintStyle.Render("↑/↓: navigate  ·  enter: apply  ·  esc: quit"))
	s.WriteString("\n")

	return s.String()
}

func (m PickerModel) Selected() *config.Preset {
	return m.selected
}

func filterPresets(presets []config.Preset, query string) []config.Preset {
	if query == "" {
		return presets
	}
	q := strings.ToLower(query)
	var result []config.Preset
	for _, p := range presets {
		target := strings.ToLower(p.Name + " " + p.DisplayName)
		if fuzzyMatch(target, q) {
			result = append(result, p)
		}
	}
	return result
}

func fuzzyMatch(text, pattern string) bool {
	pi := 0
	for ti := 0; ti < len(text) && pi < len(pattern); ti++ {
		if text[ti] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}
