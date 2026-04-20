package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/longnguyen/ws-cli/internal/detect"
	"github.com/longnguyen/ws-cli/internal/model"
)

// Candidate is one repo/worktree discovered for the add wizard.
type Candidate struct {
	Path         string
	DefaultName  string
	IsWorktree   bool
	ParentRoot   string // empty for root repos
	Icon         string // detected type key
	Selected     bool
	Name         string // editable alias
	Group        string // chosen group (empty = no group)
}

type wizardStep int

const (
	stepSelect wizardStep = iota
	stepNames
	stepGroup
	stepDone
)

type addWizardModel struct {
	cfg        model.Config
	candidates []Candidate
	cursor     int
	step       wizardStep
	editing    int // index into candidates (stepNames)
	nameBuf    string
	groupBuf   string
	cancelled  bool
}

// RunAddWizard runs the multi-step wizard and returns the finalized candidates.
// Only candidates with Selected=true should be written.
func RunAddWizard(cfg model.Config, candidates []Candidate) ([]Candidate, bool, error) {
	// Pre-populate names & icons if caller didn't.
	for i := range candidates {
		if candidates[i].Name == "" {
			candidates[i].Name = candidates[i].DefaultName
		}
		if candidates[i].Icon == "" {
			candidates[i].Icon = string(detect.DetectType(candidates[i].Path))
		}
		candidates[i].Selected = true // default all on
	}
	m := &addWizardModel{cfg: cfg, candidates: candidates}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	fm := final.(*addWizardModel)
	if fm.cancelled {
		return nil, false, nil
	}
	return fm.candidates, true, nil
}

func (m *addWizardModel) Init() tea.Cmd { return nil }

func (m *addWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch m.step {
	case stepSelect:
		return m.updateSelect(key)
	case stepNames:
		return m.updateNames(key)
	case stepGroup:
		return m.updateGroup(key)
	}
	return m, tea.Quit
}

func (m *addWizardModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.candidates)-1 {
			m.cursor++
		}
	case " ":
		m.candidates[m.cursor].Selected = !m.candidates[m.cursor].Selected
	case "a":
		for i := range m.candidates {
			m.candidates[i].Selected = true
		}
	case "n":
		for i := range m.candidates {
			m.candidates[i].Selected = false
		}
	case "enter":
		// Move to naming step, starting at first selected.
		m.editing = firstSelected(m.candidates)
		if m.editing < 0 {
			m.cancelled = true
			return m, tea.Quit
		}
		m.nameBuf = m.candidates[m.editing].Name
		m.step = stepNames
	}
	return m, nil
}

func (m *addWizardModel) updateNames(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepSelect
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "enter":
		m.candidates[m.editing].Name = strings.TrimSpace(m.nameBuf)
		if m.candidates[m.editing].Name == "" {
			m.candidates[m.editing].Name = m.candidates[m.editing].DefaultName
		}
		next := nextSelected(m.candidates, m.editing)
		if next < 0 {
			m.step = stepGroup
			return m, nil
		}
		m.editing = next
		m.nameBuf = m.candidates[m.editing].Name
	case "backspace":
		if len(m.nameBuf) > 0 {
			m.nameBuf = m.nameBuf[:len(m.nameBuf)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 {
			m.nameBuf += s
		}
	}
	return m, nil
}

func (m *addWizardModel) updateGroup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "enter":
		g := strings.TrimSpace(m.groupBuf)
		for i := range m.candidates {
			if m.candidates[i].Selected {
				m.candidates[i].Group = g
			}
		}
		return m, tea.Quit
	case "backspace":
		if len(m.groupBuf) > 0 {
			m.groupBuf = m.groupBuf[:len(m.groupBuf)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 {
			m.groupBuf += s
		}
	}
	return m, nil
}

func firstSelected(c []Candidate) int {
	for i := range c {
		if c[i].Selected {
			return i
		}
	}
	return -1
}

func nextSelected(c []Candidate, after int) int {
	for i := after + 1; i < len(c); i++ {
		if c[i].Selected {
			return i
		}
	}
	return -1
}

func (m *addWizardModel) View() string {
	var b strings.Builder
	switch m.step {
	case stepSelect:
		b.WriteString(StyleTitle.Render("ws · add — select projects to save as workspaces"))
		b.WriteString("\n\n")
		for i, c := range m.candidates {
			mark := "[ ]"
			if c.Selected {
				mark = StyleCheckbox.Render("[x]")
			}
			prefix := " "
			if i == m.cursor {
				prefix = StyleNameSel.Render(">")
			}
			icon := StyleIcon.Render(detect.Icon(detect.Type(c.Icon)))
			label := c.DefaultName
			if c.IsWorktree {
				label += StylePath.Render("  (worktree)")
			}
			fmt.Fprintf(&b, "%s %s %s  %s  %s\n", prefix, mark, icon, label, StylePath.Render(c.Path))
		}
		b.WriteString("\n")
		b.WriteString(StyleFooter.Render("space toggle · a all · n none · ⏎ next · esc cancel"))
	case stepNames:
		c := m.candidates[m.editing]
		b.WriteString(StyleTitle.Render("ws · add — name this workspace"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(StyleIcon.Render(detect.Icon(detect.Type(c.Icon))))
		b.WriteString("  ")
		b.WriteString(StylePath.Render(c.Path))
		b.WriteString("\n\n")
		b.WriteString("  name › ")
		b.WriteString(StyleNameSel.Render(m.nameBuf + "▌"))
		b.WriteString("\n\n")
		b.WriteString(StyleFooter.Render("⏎ next · esc back · ctrl+c cancel"))
	case stepGroup:
		b.WriteString(StyleTitle.Render("ws · add — group (optional)"))
		b.WriteString("\n\n")
		b.WriteString("  group › ")
		b.WriteString(StyleNameSel.Render(m.groupBuf + "▌"))
		b.WriteString(StylePath.Render("   (leave empty for no group)"))
		b.WriteString("\n\n")
		// Hint: existing groups
		if len(m.cfg.Groups) > 0 {
			var names []string
			for _, g := range m.cfg.Groups {
				names = append(names, g.Name)
			}
			b.WriteString(StylePath.Render("  existing: " + strings.Join(names, ", ")))
			b.WriteString("\n\n")
		}
		b.WriteString(StyleFooter.Render("⏎ confirm · ctrl+c cancel"))
	}
	return b.String()
}

// BasenameDefault returns a reasonable default alias from a filesystem path.
func BasenameDefault(p string) string {
	return filepath.Base(p)
}
