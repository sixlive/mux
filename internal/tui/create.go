package tui

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tjmiller/mux/internal/audio"
	"github.com/tjmiller/mux/internal/config"
)

type createStep int

const (
	stepName createStep = iota
	stepDisplayName
	stepConfigureOutput
	stepOutputDevice
	stepOutputVolume
	stepConfigureInput
	stepInputDevice
	stepInputVolume
	stepConfirm
)

type deviceItem struct {
	device audio.Device
}

func (i deviceItem) FilterValue() string {
	return i.device.Name + " " + i.device.TransportType + " " + i.device.UID
}

type deviceDelegate struct {
	defaultUID string
}

func (d deviceDelegate) Height() int                             { return 1 }
func (d deviceDelegate) Spacing() int                            { return 0 }
func (d deviceDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d deviceDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(deviceItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	dev := item.device

	isDefault := dev.UID == d.defaultUID

	nameStyle := lipgloss.NewStyle()
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	cursor := "  "
	if selected {
		cursor = cursorStyle.Render("> ")
		nameStyle = nameStyle.Foreground(lipgloss.Color("12")).Bold(true)
	}

	marker := "  "
	if isDefault {
		marker = defaultStyle.Render("● ")
	}

	transport := dimStyle.Render(fmt.Sprintf("(%s)", dev.TransportType))
	fmt.Fprintf(w, "%s%s%s %s", cursor, marker, nameStyle.Render(dev.Name), transport)
}

type CreateModel struct {
	step         createStep
	isEdit       bool
	originalName string

	nameInput    textinput.Model
	displayInput textinput.Model
	volumeInput  textinput.Model

	outputDeviceList list.Model
	inputDeviceList  list.Model

	outputDevice        *audio.Device
	outputVolume        int
	outputAcceptsVolume bool
	inputDevice         *audio.Device
	inputVolume         int
	inputAcceptsVolume  bool

	configureOutput bool
	configureInput  bool
	yesNoChoice     int

	allDevices    []audio.Device
	defaultOutUID string
	defaultInUID  string

	width  int
	height int

	result   *config.Preset
	quitting bool
	err      error

	existingNames []string
}

func NewCreateModel(devices []audio.Device, existingNames []string) CreateModel {
	defaultOutUID, _ := audio.GetDefaultOutputUID()
	defaultInUID, _ := audio.GetDefaultInputUID()

	ni := textinput.New()
	ni.Placeholder = "my-preset"
	ni.Focus()
	ni.CharLimit = 64

	di := textinput.New()
	di.Placeholder = "My Preset"
	di.CharLimit = 128

	vi := textinput.New()
	vi.Placeholder = "50"
	vi.CharLimit = 3

	return CreateModel{
		step:          stepName,
		nameInput:     ni,
		displayInput:  di,
		volumeInput:   vi,
		allDevices:    devices,
		defaultOutUID: defaultOutUID,
		defaultInUID:  defaultInUID,
		existingNames: existingNames,
		outputVolume:  50,
		inputVolume:   50,
	}
}

func NewEditModel(devices []audio.Device, preset config.Preset, existingNames []string) CreateModel {
	m := NewCreateModel(devices, existingNames)
	m.isEdit = true
	m.originalName = preset.Name

	m.nameInput.SetValue(preset.Name)
	m.displayInput.SetValue(preset.DisplayName)

	if preset.Output != nil {
		m.configureOutput = true
		for _, d := range devices {
			if d.UID == preset.Output.UID {
				m.outputDevice = &d
				break
			}
		}
		if m.outputDevice == nil {
			for _, d := range devices {
				if strings.EqualFold(d.Name, preset.Output.Name) {
					m.outputDevice = &d
					break
				}
			}
		}
		m.outputVolume = preset.Output.Volume
	}

	if preset.Input != nil {
		m.configureInput = true
		for _, d := range devices {
			if d.UID == preset.Input.UID {
				m.inputDevice = &d
				break
			}
		}
		if m.inputDevice == nil {
			for _, d := range devices {
				if strings.EqualFold(d.Name, preset.Input.Name) {
					m.inputDevice = &d
					break
				}
			}
		}
		m.inputVolume = preset.Input.Volume
	}

	return m
}

func (m CreateModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			return m.goBack()
		}
	}

	switch m.step {
	case stepName:
		return m.updateName(msg)
	case stepDisplayName:
		return m.updateDisplayName(msg)
	case stepConfigureOutput:
		return m.updateYesNo(msg, true)
	case stepOutputDevice:
		return m.updateDeviceList(msg, true)
	case stepOutputVolume:
		return m.updateVolume(msg, true)
	case stepConfigureInput:
		return m.updateYesNo(msg, false)
	case stepInputDevice:
		return m.updateDeviceList(msg, false)
	case stepInputVolume:
		return m.updateVolume(msg, false)
	case stepConfirm:
		return m.updateConfirm(msg)
	}

	return m, nil
}

func (m CreateModel) goBack() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepName:
		m.quitting = true
		return m, tea.Quit
	case stepDisplayName:
		m.step = stepName
		m.nameInput.Focus()
	case stepConfigureOutput:
		m.step = stepDisplayName
		m.displayInput.Focus()
	case stepOutputDevice:
		m.step = stepConfigureOutput
		m.yesNoChoice = 0
	case stepOutputVolume:
		m.step = stepOutputDevice
	case stepConfigureInput:
		if m.configureOutput && m.outputAcceptsVolume {
			m.step = stepOutputVolume
			m.volumeInput.SetValue(strconv.Itoa(m.outputVolume))
			m.volumeInput.Focus()
		} else if m.configureOutput {
			m.step = stepOutputDevice
		} else {
			m.step = stepConfigureOutput
			m.yesNoChoice = 1
		}
	case stepInputDevice:
		m.step = stepConfigureInput
		m.yesNoChoice = 0
	case stepInputVolume:
		m.step = stepInputDevice
	case stepConfirm:
		if m.configureInput && m.inputAcceptsVolume {
			m.step = stepInputVolume
			m.volumeInput.SetValue(strconv.Itoa(m.inputVolume))
			m.volumeInput.Focus()
		} else if m.configureInput {
			m.step = stepInputDevice
		} else {
			m.step = stepConfigureInput
			m.yesNoChoice = 1
		}
	}
	m.err = nil
	return m, nil
}

func (m CreateModel) updateName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		name := m.nameInput.Value()
		if err := config.ValidateName(name); err != nil {
			m.err = err
			return m, nil
		}
		if !m.isEdit || name != m.originalName {
			for _, existing := range m.existingNames {
				if strings.EqualFold(existing, name) {
					m.err = fmt.Errorf("preset %q already exists", name)
					return m, nil
				}
			}
		}
		m.err = nil
		m.step = stepDisplayName
		if m.displayInput.Value() == "" {
			m.displayInput.SetValue(toTitleCase(name))
		}
		m.displayInput.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m CreateModel) updateDisplayName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		if m.displayInput.Value() == "" {
			m.err = fmt.Errorf("display name cannot be empty")
			return m, nil
		}
		m.err = nil
		m.step = stepConfigureOutput
		m.yesNoChoice = 0
		if m.configureOutput {
			m.yesNoChoice = 0
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.displayInput, cmd = m.displayInput.Update(msg)
	return m, cmd
}

func (m CreateModel) updateYesNo(msg tea.Msg, isOutput bool) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "left", "h":
			m.yesNoChoice = 0
		case "right", "l":
			m.yesNoChoice = 1
		case "tab":
			m.yesNoChoice = (m.yesNoChoice + 1) % 2
		case "enter":
			if isOutput {
				m.configureOutput = m.yesNoChoice == 0
				if m.configureOutput {
					m.step = stepOutputDevice
					m.outputDeviceList = m.buildDeviceList(true)
					return m, nil
				}
				m.step = stepConfigureInput
				m.yesNoChoice = 0
				if m.configureInput {
					m.yesNoChoice = 0
				}
			} else {
				m.configureInput = m.yesNoChoice == 0
				if m.configureInput {
					m.step = stepInputDevice
					m.inputDeviceList = m.buildDeviceList(false)
					return m, nil
				}
				m.step = stepConfirm
			}
			return m, nil
		}
	}
	return m, nil
}

func (m CreateModel) buildDeviceList(output bool) list.Model {
	uid := m.defaultInUID
	if output {
		uid = m.defaultOutUID
	}
	del := deviceDelegate{defaultUID: uid}

	var items []list.Item
	for _, d := range m.allDevices {
		if output && d.HasOutput {
			items = append(items, deviceItem{device: d})
		} else if !output && d.HasInput {
			items = append(items, deviceItem{device: d})
		}
	}

	height := 15
	if m.height > 0 {
		height = m.height - 8
		if height < 5 {
			height = 5
		}
	}

	l := list.New(items, del, 60, height)
	if output {
		l.Title = "Select output device"
	} else {
		l.Title = "Select input device"
	}
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).MarginLeft(2)

	return l
}

func (m CreateModel) updateDeviceList(msg tea.Msg, isOutput bool) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		var dl *list.Model
		if isOutput {
			dl = &m.outputDeviceList
		} else {
			dl = &m.inputDeviceList
		}
		if item, ok := dl.SelectedItem().(deviceItem); ok {
			dev := item.device
			if isOutput {
				m.outputDevice = &dev
				probe := audio.ProbeVolumeControl(dev.UID, audio.ScopeOutput)
				m.outputAcceptsVolume = probe.AcceptsVolume
				if probe.AcceptsVolume {
					m.outputVolume = probe.Volume
					m.step = stepOutputVolume
					m.volumeInput.SetValue(strconv.Itoa(m.outputVolume))
					m.volumeInput.Focus()
				} else {
					m.outputVolume = -1
					m.step = stepConfigureInput
					m.yesNoChoice = 0
				}
			} else {
				m.inputDevice = &dev
				probe := audio.ProbeVolumeControl(dev.UID, audio.ScopeInput)
				m.inputAcceptsVolume = probe.AcceptsVolume
				if probe.AcceptsVolume {
					m.inputVolume = probe.Volume
					m.step = stepInputVolume
					m.volumeInput.SetValue(strconv.Itoa(m.inputVolume))
					m.volumeInput.Focus()
				} else {
					m.inputVolume = -1
					m.step = stepConfirm
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	if isOutput {
		m.outputDeviceList, cmd = m.outputDeviceList.Update(msg)
	} else {
		m.inputDeviceList, cmd = m.inputDeviceList.Update(msg)
	}
	return m, cmd
}

func (m CreateModel) updateVolume(msg tea.Msg, isOutput bool) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			val, err := strconv.Atoi(m.volumeInput.Value())
			if err != nil || val < 0 || val > 100 {
				m.err = fmt.Errorf("volume must be 0-100")
				return m, nil
			}
			m.err = nil
			if isOutput {
				m.outputVolume = val
				m.step = stepConfigureInput
				m.yesNoChoice = 0
				if m.configureInput {
					m.yesNoChoice = 0
				}
			} else {
				m.inputVolume = val
				m.step = stepConfirm
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.volumeInput, cmd = m.volumeInput.Update(msg)
	return m, cmd
}

func (m CreateModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter", "y":
			preset := m.buildPreset()
			m.result = &preset
			return m, tea.Quit
		case "n", "q":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m CreateModel) buildPreset() config.Preset {
	p := config.Preset{
		Name:        m.nameInput.Value(),
		DisplayName: m.displayInput.Value(),
	}
	if m.configureOutput && m.outputDevice != nil {
		p.Output = &config.DeviceConfig{
			UID:    m.outputDevice.UID,
			Name:   m.outputDevice.Name,
			Volume: m.outputVolume,
		}
	}
	if m.configureInput && m.inputDevice != nil {
		p.Input = &config.DeviceConfig{
			UID:    m.inputDevice.UID,
			Name:   m.inputDevice.Name,
			Volume: m.inputVolume,
		}
	}
	return p
}

func (m CreateModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).MarginLeft(2).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).MarginLeft(2)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).MarginLeft(2)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	title := "Create Preset"
	if m.isEdit {
		title = "Edit Preset"
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render(title))
	s.WriteString("\n\n")

	stepIndicator := m.renderStepIndicator()
	s.WriteString(stepIndicator)
	s.WriteString("\n\n")

	switch m.step {
	case stepName:
		s.WriteString(labelStyle.Render("Preset name (kebab-case):"))
		s.WriteString("\n")
		s.WriteString("  " + m.nameInput.View())
		s.WriteString("\n")
		if m.err != nil {
			s.WriteString(errStyle.Render(m.err.Error()))
			s.WriteString("\n")
		}
		s.WriteString(hintStyle.Render("Press Enter to continue, Esc to cancel"))

	case stepDisplayName:
		s.WriteString(labelStyle.Render("Display name:"))
		s.WriteString("\n")
		s.WriteString("  " + m.displayInput.View())
		s.WriteString("\n")
		if m.err != nil {
			s.WriteString(errStyle.Render(m.err.Error()))
			s.WriteString("\n")
		}
		s.WriteString(hintStyle.Render("Press Enter to continue, Esc to go back"))

	case stepConfigureOutput:
		s.WriteString(labelStyle.Render("Configure output device?"))
		s.WriteString("\n\n")
		yes := "  Yes  "
		no := "  No  "
		if m.yesNoChoice == 0 {
			yes = selectedStyle.Render("▸ Yes  ")
			no = unselectedStyle.Render("  No  ")
		} else {
			yes = unselectedStyle.Render("  Yes  ")
			no = selectedStyle.Render("▸ No  ")
		}
		s.WriteString("  " + yes + no)
		s.WriteString("\n\n")
		s.WriteString(hintStyle.Render("←/→ to select, Enter to confirm, Esc to go back"))

	case stepOutputDevice:
		s.WriteString(m.outputDeviceList.View())

	case stepOutputVolume:
		s.WriteString(labelStyle.Render(fmt.Sprintf("Output volume for %s:", m.outputDevice.Name)))
		s.WriteString("\n")
		s.WriteString("  " + m.volumeInput.View())
		s.WriteString("\n")
		s.WriteString(m.renderVolumeBar(m.volumeInput.Value()))
		s.WriteString("\n")
		if m.err != nil {
			s.WriteString(errStyle.Render(m.err.Error()))
			s.WriteString("\n")
		}
		s.WriteString(hintStyle.Render("Enter a value 0-100, press Enter to continue"))

	case stepConfigureInput:
		s.WriteString(labelStyle.Render("Configure input device?"))
		s.WriteString("\n\n")
		yes := "  Yes  "
		no := "  No  "
		if m.yesNoChoice == 0 {
			yes = selectedStyle.Render("▸ Yes  ")
			no = unselectedStyle.Render("  No  ")
		} else {
			yes = unselectedStyle.Render("  Yes  ")
			no = selectedStyle.Render("▸ No  ")
		}
		s.WriteString("  " + yes + no)
		s.WriteString("\n\n")
		s.WriteString(hintStyle.Render("←/→ to select, Enter to confirm, Esc to go back"))

	case stepInputDevice:
		s.WriteString(m.inputDeviceList.View())

	case stepInputVolume:
		s.WriteString(labelStyle.Render(fmt.Sprintf("Input volume for %s:", m.inputDevice.Name)))
		s.WriteString("\n")
		s.WriteString("  " + m.volumeInput.View())
		s.WriteString("\n")
		s.WriteString(m.renderVolumeBar(m.volumeInput.Value()))
		s.WriteString("\n")
		if m.err != nil {
			s.WriteString(errStyle.Render(m.err.Error()))
			s.WriteString("\n")
		}
		s.WriteString(hintStyle.Render("Enter a value 0-100, press Enter to continue"))

	case stepConfirm:
		s.WriteString(m.renderConfirmation())
	}

	s.WriteString("\n")
	return s.String()
}

func (m CreateModel) renderStepIndicator() string {
	steps := []string{"Name", "Display", "Output", "Input", "Confirm"}
	stepMap := map[createStep]int{
		stepName:            0,
		stepDisplayName:     1,
		stepConfigureOutput: 2,
		stepOutputDevice:    2,
		stepOutputVolume:    2,
		stepConfigureInput:  3,
		stepInputDevice:     3,
		stepInputVolume:     3,
		stepConfirm:         4,
	}

	current := stepMap[m.step]
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var parts []string
	for i, name := range steps {
		if i < current {
			parts = append(parts, doneStyle.Render("* "+name))
		} else if i == current {
			parts = append(parts, activeStyle.Render("● "+name))
		} else {
			parts = append(parts, pendingStyle.Render("○ "+name))
		}
	}
	return "  " + strings.Join(parts, "  ")
}

func (m CreateModel) renderVolumeBar(valStr string) string {
	val, err := strconv.Atoi(valStr)
	if err != nil || val < 0 {
		val = 0
	}
	if val > 100 {
		val = 100
	}

	filled := val / 5
	empty := 20 - filled

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	bar := barStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", empty))
	return fmt.Sprintf("  [%s] %d%%", bar, val)
}

func (m CreateModel) renderConfirmation() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var s strings.Builder
	s.WriteString("  " + borderStyle.Render("─────────────────────────────"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Name:        "), valueStyle.Render(m.nameInput.Value())))
	s.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Display Name:"), valueStyle.Render(m.displayInput.Value())))

	if m.configureOutput && m.outputDevice != nil {
		volStr := fmt.Sprintf("@ %d%%", m.outputVolume)
		if m.outputVolume < 0 {
			volStr = "(device-controlled)"
		}
		s.WriteString(fmt.Sprintf("  %s  %s %s\n",
			labelStyle.Render("Output:      "),
			valueStyle.Render(fmt.Sprintf("%s (%s)", m.outputDevice.Name, m.outputDevice.TransportType)),
			volStr))
	} else {
		s.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Output:      "), labelStyle.Render("(unchanged)")))
	}

	if m.configureInput && m.inputDevice != nil {
		volStr := fmt.Sprintf("@ %d%%", m.inputVolume)
		if m.inputVolume < 0 {
			volStr = "(device-controlled)"
		}
		s.WriteString(fmt.Sprintf("  %s  %s %s\n",
			labelStyle.Render("Input:       "),
			valueStyle.Render(fmt.Sprintf("%s (%s)", m.inputDevice.Name, m.inputDevice.TransportType)),
			volStr))
	} else {
		s.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Input:       "), labelStyle.Render("(unchanged)")))
	}

	s.WriteString("  " + borderStyle.Render("─────────────────────────────"))
	s.WriteString("\n\n")
	s.WriteString(hintStyle.Render("Enter/y: Save  ·  n/q: Cancel  ·  Esc: Back"))

	return s.String()
}

func (m CreateModel) Result() *config.Preset {
	return m.result
}

func toTitleCase(kebab string) string {
	words := strings.Split(kebab, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
