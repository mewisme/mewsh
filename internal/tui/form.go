package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

type WizardStep int

const (
	StepAlias WizardStep = iota
	StepConnectionType
	StepHost
	StepCFHostname
	StepPort
	StepUser
	StepAuthType
	StepKeyPath
	StepPassword
	StepSummary
)

type formModel struct {
	steps       []WizardStep
	stepIndex   int
	editing     bool
	origAlias   string
	profile     profile.Profile
	password    string
	submitted   bool
	cancelled   bool
	errMsg      string
	textInput   textinput.Model
	choiceIndex int
	width       int
	height      int
	cfg         *config.Config
}

func newFormModel(existing *profile.Profile, cfg *config.Config) formModel {
	p := profile.Default()
	editing := false
	orig := ""
	if existing != nil {
		p = *existing
		p.ApplyDefaults()
		editing = true
		orig = existing.Alias
	}
	f := formModel{
		editing:   editing,
		origAlias: orig,
		profile:   p,
		cfg:       cfg,
	}
	f.rebuildSteps()
	f.syncStepInput()
	return f
}

func (f *formModel) rebuildSteps() {
	steps := []WizardStep{StepAlias, StepConnectionType}
	if f.profile.ConnectionType == profile.ConnectionCloudflareAccess {
		steps = append(steps, StepCFHostname, StepUser)
	} else {
		steps = append(steps, StepHost, StepPort, StepUser)
	}
	steps = append(steps, StepAuthType)
	switch f.profile.AuthType {
	case profile.AuthKey:
		steps = append(steps, StepKeyPath)
	case profile.AuthPassword:
		steps = append(steps, StepPassword)
	}
	steps = append(steps, StepSummary)
	f.steps = steps
	if f.stepIndex >= len(f.steps) {
		f.stepIndex = len(f.steps) - 1
	}
}

func (f *formModel) currentStep() WizardStep {
	if f.stepIndex < 0 || f.stepIndex >= len(f.steps) {
		return StepSummary
	}
	return f.steps[f.stepIndex]
}

func (f *formModel) applyWindowSize(w, h int) {
	f.width = w
	f.height = h
	f.textInput.Width = f.inputWidth()
}

func (f formModel) withSize(w, h int) formModel {
	f.applyWindowSize(w, h)
	return f
}

func (f formModel) Init() tea.Cmd {
	return textinput.Blink
}

func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if handled, cmd := f.handleWizardKey(keyMsg); handled {
			return f, cmd
		}
		// Don't pass wizard navigation keys to textinput.
		if f.isTextStep() && isTextInputKey(keyMsg) {
			var cmd tea.Cmd
			f.textInput, cmd = f.textInput.Update(msg)
			return f, cmd
		}
		return f, nil
	}

	if f.isTextStep() && f.currentStep() != StepSummary {
		var cmd tea.Cmd
		f.textInput, cmd = f.textInput.Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f *formModel) handleWizardKey(keyMsg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case isEsc(keyMsg):
		f.cancelled = true
		return true, nil
	case isPrevStep(keyMsg):
		f.errMsg = ""
		if f.currentStep() == StepSummary {
			f.stepIndex = len(f.steps) - 2
			if f.stepIndex < 0 {
				f.stepIndex = 0
			}
		} else if f.stepIndex > 0 {
			f.stepIndex--
		}
		f.syncStepInput()
		return true, textinput.Blink
	case isNextStep(keyMsg):
		f.errMsg = ""
		if f.currentStep() != StepSummary {
			if err := f.applyCurrentStep(); err != nil {
				f.errMsg = err.Error()
				return true, nil
			}
			f.rebuildSteps()
			if f.stepIndex < len(f.steps)-1 {
				f.stepIndex++
			}
			f.syncStepInput()
		}
		return true, textinput.Blink
	case isEnter(keyMsg):
		if f.currentStep() == StepSummary {
			return true, nil
		}
		if err := f.applyCurrentStep(); err != nil {
			f.errMsg = err.Error()
			return true, nil
		}
		f.errMsg = ""
		f.rebuildSteps()
		if f.stepIndex < len(f.steps)-1 {
			f.stepIndex++
			f.syncStepInput()
			return true, textinput.Blink
		}
		return true, nil
	case isSaveKey(keyMsg) && f.currentStep() == StepSummary:
		if err := f.commit(); err != nil {
			f.errMsg = err.Error()
			return true, nil
		}
		f.submitted = true
		return true, nil
	case isEditKey(keyMsg) && f.currentStep() == StepSummary:
		f.stepIndex = 0
		f.errMsg = ""
		f.syncStepInput()
		return true, textinput.Blink
	}

	if f.isChoiceStep() {
		switch {
		case isUp(keyMsg):
			if f.choiceIndex > 0 {
				f.choiceIndex--
			}
			return true, nil
		case isDown(keyMsg):
			if f.choiceIndex < f.choiceCount()-1 {
				f.choiceIndex++
			}
			return true, nil
		}
	}

	return false, nil
}

func isEnter(key tea.KeyMsg) bool {
	return key.Type == tea.KeyEnter || key.String() == "enter" || key.String() == "ctrl+m"
}

func isEsc(key tea.KeyMsg) bool {
	return key.Type == tea.KeyEsc || key.String() == "esc"
}

func isUp(key tea.KeyMsg) bool {
	switch key.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		return true
	}
	switch key.String() {
	case "up", "shift+tab", "k":
		return true
	}
	return false
}

func isDown(key tea.KeyMsg) bool {
	switch key.Type {
	case tea.KeyDown, tea.KeyTab:
		return true
	}
	switch key.String() {
	case "down", "tab", "j":
		return true
	}
	return false
}

func isPrevStep(key tea.KeyMsg) bool {
	if key.Type == tea.KeyCtrlLeft {
		return true
	}
	return key.String() == "ctrl+left" || key.String() == "alt+left"
}

func isNextStep(key tea.KeyMsg) bool {
	if key.Type == tea.KeyCtrlRight {
		return true
	}
	return key.String() == "ctrl+right" || key.String() == "alt+right"
}

func isSaveKey(key tea.KeyMsg) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 's' || key.Runes[0] == 'S')
}

func isEditKey(key tea.KeyMsg) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 'e' || key.Runes[0] == 'E')
}

func isTextInputKey(key tea.KeyMsg) bool {
	switch key.Type {
	case tea.KeyEnter, tea.KeyEsc, tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight,
		tea.KeyTab, tea.KeyShiftTab, tea.KeyCtrlLeft, tea.KeyCtrlRight:
		return false
	}
	switch key.String() {
	case "enter", "esc", "up", "down", "left", "right", "tab", "shift+tab",
		"ctrl+left", "ctrl+right", "alt+left", "alt+right", "ctrl+m":
		return false
	}
	return true
}

func (f *formModel) isChoiceStep() bool {
	switch f.currentStep() {
	case StepConnectionType, StepAuthType:
		return true
	default:
		return false
	}
}

func (f *formModel) isTextStep() bool {
	switch f.currentStep() {
	case StepAlias, StepHost, StepCFHostname, StepPort, StepUser, StepKeyPath, StepPassword:
		return true
	default:
		return false
	}
}

func (f *formModel) choiceCount() int {
	switch f.currentStep() {
	case StepConnectionType:
		return 2
	case StepAuthType:
		return 3
	default:
		return 0
	}
}

func (f *formModel) syncStepInput() {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = f.inputWidth()

	switch f.currentStep() {
	case StepAlias:
		ti.SetValue(f.profile.Alias)
		ti.Placeholder = "e.g. production"
		ti.Focus()
	case StepHost:
		ti.SetValue(f.profile.Host)
		ti.Placeholder = "e.g. 192.168.1.100"
		ti.Focus()
	case StepCFHostname:
		ti.SetValue(f.profile.CFHostname)
		ti.Placeholder = "e.g. ssh.example.com"
		ti.Focus()
	case StepPort:
		ti.SetValue(strconv.Itoa(f.profile.Port))
		ti.Placeholder = "22"
		ti.Focus()
	case StepUser:
		ti.SetValue(f.profile.User)
		ti.Placeholder = "e.g. root"
		ti.Focus()
	case StepKeyPath:
		ti.SetValue(f.profile.KeyPath)
		ti.Placeholder = "e.g. ~/.ssh/id_rsa"
		ti.Focus()
	case StepPassword:
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
		ti.Placeholder = "enter password"
		if f.editing {
			ti.Placeholder = "leave blank to keep existing"
		}
		ti.Focus()
	case StepConnectionType:
		f.choiceIndex = connectionChoiceIndex(f.profile.ConnectionType)
		ti.Blur()
	case StepAuthType:
		f.choiceIndex = authChoiceIndex(f.profile.AuthType)
		ti.Blur()
	case StepSummary:
		ti.Blur()
	default:
		ti.Blur()
	}

	f.textInput = ti
}

func connectionChoiceIndex(v string) int {
	if v == profile.ConnectionCloudflareAccess {
		return 1
	}
	return 0
}

func authChoiceIndex(v string) int {
	switch v {
	case profile.AuthKey:
		return 1
	case profile.AuthPassword:
		return 2
	default:
		return 0
	}
}

func (f *formModel) applyCurrentStep() error {
	switch f.currentStep() {
	case StepAlias:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			return fmt.Errorf("alias is required")
		}
		if !f.editing && f.cfg.AliasExists(v) {
			return fmt.Errorf("alias %q already exists", v)
		}
		if f.editing && v != f.origAlias && f.cfg.AliasExists(v) {
			return fmt.Errorf("alias %q already exists", v)
		}
		f.profile.Alias = v
	case StepConnectionType:
		switch f.choiceIndex {
		case 0:
			f.profile.ConnectionType = profile.ConnectionDirect
		case 1:
			f.profile.ConnectionType = profile.ConnectionCloudflareAccess
		}
	case StepHost:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			return fmt.Errorf("host is required")
		}
		f.profile.Host = v
	case StepCFHostname:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			return fmt.Errorf("cloudflare hostname is required")
		}
		f.profile.CFHostname = v
	case StepPort:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			v = "22"
		}
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid port")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		f.profile.Port = port
	case StepUser:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			return fmt.Errorf("user is required")
		}
		f.profile.User = v
	case StepAuthType:
		switch f.choiceIndex {
		case 0:
			f.profile.AuthType = profile.AuthAgent
		case 1:
			f.profile.AuthType = profile.AuthKey
		case 2:
			f.profile.AuthType = profile.AuthPassword
			f.profile.PasswordMode = profile.PasswordManual
		}
	case StepKeyPath:
		v := strings.TrimSpace(f.textInput.Value())
		if v == "" {
			return fmt.Errorf("key path is required")
		}
		if _, err := os.Stat(v); err != nil {
			return fmt.Errorf("key path: %w", err)
		}
		f.profile.KeyPath = v
	case StepPassword:
		v := f.textInput.Value()
		if v == "" && !f.editing {
			return fmt.Errorf("password is required")
		}
		if v != "" {
			f.password = v
		}
		f.profile.PasswordMode = profile.PasswordManual
		if f.profile.PasswordRef == "" {
			f.profile.PasswordRef = f.profile.Alias
		}
	}
	return nil
}

func (f *formModel) commit() error {
	if f.profile.PasswordRef == "" {
		f.profile.PasswordRef = f.profile.Alias
	}
	aliases := make([]string, len(f.cfg.Profiles))
	for i, p := range f.cfg.Profiles {
		aliases[i] = p.Alias
	}
	return f.profile.Validate(aliases, f.editing)
}

func (f formModel) View() string {
	mode := "Add Profile"
	if f.editing {
		mode = "Edit Profile"
	}

	var body strings.Builder
	body.WriteString(wizardTitleStyle.Render(fmt.Sprintf("mewsh — %s", mode)))
	body.WriteString("\n\n")

	if f.currentStep() == StepSummary {
		body.WriteString(stepLabelStyle.Render("Profile Summary"))
		body.WriteString("\n\n")
		body.WriteString(f.renderSummary())
	} else {
		stepNum := f.stepIndex + 1
		total := len(f.steps)
		body.WriteString(stepLabelStyle.Render(fmt.Sprintf("Step %d/%d", stepNum, total)))
		body.WriteString("\n")
		body.WriteString(questionStyle.Render(f.stepQuestion()))
		body.WriteString("\n\n")
		body.WriteString(f.renderStepContent())
		if f.errMsg != "" {
			body.WriteString("\n")
			body.WriteString(errStyle.Render(f.errMsg))
		}
	}

	return wizardBoxStyle.Width(f.boxWidth()).Render(body.String())
}

func (f formModel) stepQuestion() string {
	switch f.currentStep() {
	case StepAlias:
		return "Profile Alias"
	case StepConnectionType:
		return "Connection Type"
	case StepHost:
		return "Host"
	case StepCFHostname:
		return "CF Hostname"
	case StepPort:
		return "Port"
	case StepUser:
		return "User"
	case StepAuthType:
		return "Auth Type"
	case StepKeyPath:
		return "Key Path"
	case StepPassword:
		return "Password"
	default:
		return ""
	}
}

func (f formModel) renderStepContent() string {
	switch f.currentStep() {
	case StepConnectionType:
		return renderChoices([]string{"Direct SSH", "Cloudflare Access"}, f.choiceIndex)
	case StepAuthType:
		return renderChoices([]string{"SSH Agent", "Key", "Password"}, f.choiceIndex)
	default:
		return f.textInput.View()
	}
}

func renderChoices(options []string, selected int) string {
	var b strings.Builder
	for i, opt := range options {
		line := "  " + opt
		if i == selected {
			line = choiceSelectedStyle.Render("> " + opt)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (f formModel) renderSummary() string {
	conn := "Direct SSH"
	host := f.profile.Host
	if f.profile.ConnectionType == profile.ConnectionCloudflareAccess {
		conn = "Cloudflare Access"
		host = f.profile.CFHostname
	}
	auth := f.profile.AuthType
	lines := []string{
		fmt.Sprintf("Alias: %s", f.profile.Alias),
		fmt.Sprintf("Connection: %s", conn),
	}
	if f.profile.ConnectionType == profile.ConnectionDirect {
		lines = append(lines, fmt.Sprintf("Host: %s:%d", host, f.profile.Port))
	} else {
		lines = append(lines, fmt.Sprintf("Hostname: %s", host))
	}
	lines = append(lines,
		fmt.Sprintf("User: %s", f.profile.User),
		fmt.Sprintf("Auth: %s", authLabel(auth)),
	)
	if f.profile.AuthType == profile.AuthKey {
		lines = append(lines, fmt.Sprintf("Key: %s", f.profile.KeyPath))
	}
	if f.profile.AuthType == profile.AuthPassword {
		if f.password != "" {
			lines = append(lines, "Password: (will be saved to keyring)")
		} else if f.editing {
			lines = append(lines, "Password: (unchanged)")
		}
	}
	return summaryStyle.Render(strings.Join(lines, "\n"))
}

func authLabel(auth string) string {
	switch auth {
	case profile.AuthAgent:
		return "Agent"
	case profile.AuthKey:
		return "Key"
	case profile.AuthPassword:
		return "Password"
	default:
		return auth
	}
}

var (
	wizardTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	wizardBoxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69")).Padding(1, 2)
	stepLabelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	questionStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	choiceSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	summaryStyle        = lipgloss.NewStyle().PaddingLeft(1)
)
