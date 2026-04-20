package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/longnguyen/ws-cli/internal/detect"
	"github.com/longnguyen/ws-cli/internal/model"
)

// Candidate is one repo/worktree discovered for the add wizard.
//
// For IsWorktree=true candidates, `Name` holds the user-entered suffix
// (empty = leave unaliased). The caller (applyAdd) is responsible for
// prefixing it with the parent root's alias.
type Candidate struct {
	Path        string
	DefaultName string
	IsWorktree  bool
	ParentRoot  string
	Icon        string
	Selected    bool
	Name        string
	Group       string
}

type wizardStep int

const (
	stepSelect wizardStep = iota
	stepNames
	stepGroup
)

type addWizardModel struct {
	cfg        model.Config
	candidates []Candidate
	order      []int // indices into candidates, sorted roots-first for naming pass
	cursor     int
	step       wizardStep
	editIdx    int // position into order[] (stepNames)
	nameBuf    string
	groupBuf   string
	cancelled  bool
}

// RunAddWizard runs the multi-step wizard and returns the finalized candidates.
// Only candidates with Selected=true should be written.
func RunAddWizard(cfg model.Config, candidates []Candidate) ([]Candidate, bool, error) {
	for i := range candidates {
		if candidates[i].Icon == "" {
			candidates[i].Icon = string(detect.DetectType(candidates[i].Path))
		}
		candidates[i].Selected = true
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
		m.buildNamingOrder()
		if len(m.order) == 0 {
			m.cancelled = true
			return m, tea.Quit
		}
		m.editIdx = 0
		m.loadNameBuf()
		m.step = stepNames
	}
	return m, nil
}

// buildNamingOrder sorts selected candidate indices so roots come first.
// Within roots and within worktrees, preserve discovery order. This lets the
// naming step show a parent alias prefix while naming a worktree.
func (m *addWizardModel) buildNamingOrder() {
	m.order = nil
	for i, c := range m.candidates {
		if c.Selected && !c.IsWorktree {
			m.order = append(m.order, i)
		}
	}
	for i, c := range m.candidates {
		if c.Selected && c.IsWorktree {
			m.order = append(m.order, i)
		}
	}
	// Stable: Go's append preserves order; no sort needed.
	_ = sort.Stable
}

// loadNameBuf initializes the input for the current candidate. For worktrees
// the buffer starts empty (optional alias); for roots it starts with
// DefaultName so user can hit Enter to accept.
func (m *addWizardModel) loadNameBuf() {
	idx := m.order[m.editIdx]
	c := m.candidates[idx]
	if c.IsWorktree {
		m.nameBuf = "" // optional
	} else {
		m.nameBuf = c.DefaultName
	}
}

func (m *addWizardModel) updateNames(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepSelect
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "enter":
		idx := m.order[m.editIdx]
		c := &m.candidates[idx]
		trimmed := strings.TrimSpace(m.nameBuf)
		if c.IsWorktree {
			// Optional alias. Store just the suffix; prefix is applied in applyAdd.
			c.Name = trimmed
		} else {
			if trimmed == "" {
				trimmed = c.DefaultName
			}
			c.Name = trimmed
		}
		m.editIdx++
		if m.editIdx >= len(m.order) {
			m.step = stepGroup
			return m, nil
		}
		m.loadNameBuf()
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

// parentAliasFor returns the display alias for a worktree candidate's parent.
// Looks first in the current batch (already-named roots), then in cfg, then
// falls back to the parent directory's basename.
func (m *addWizardModel) parentAliasFor(c Candidate) string {
	for _, cc := range m.candidates {
		if cc.Selected && !cc.IsWorktree && filepath.Clean(cc.Path) == filepath.Clean(c.ParentRoot) {
			if cc.Name != "" {
				return cc.Name
			}
			return cc.DefaultName
		}
	}
	for _, w := range m.cfg.Workspaces {
		if filepath.Clean(w.Path) == filepath.Clean(c.ParentRoot) {
			return w.Name
		}
	}
	return filepath.Base(c.ParentRoot)
}

func (m *addWizardModel) View() string {
	switch m.step {
	case stepSelect:
		return m.viewSelect()
	case stepNames:
		return m.viewNames()
	case stepGroup:
		return m.viewGroup()
	}
	return ""
}

func (m *addWizardModel) viewSelect() string {
	var b strings.Builder
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
	return b.String()
}

func (m *addWizardModel) viewNames() string {
	idx := m.order[m.editIdx]
	c := m.candidates[idx]
	var b strings.Builder
	if c.IsWorktree {
		b.WriteString(StyleTitle.Render("ws · add — worktree alias (optional)"))
	} else {
		b.WriteString(StyleTitle.Render("ws · add — name this workspace"))
	}
	b.WriteString("   ")
	b.WriteString(StylePath.Render(fmt.Sprintf("(%d/%d)", m.editIdx+1, len(m.order))))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(StyleIcon.Render(detect.Icon(detect.Type(c.Icon))))
	b.WriteString("  ")
	b.WriteString(StylePath.Render(c.Path))
	b.WriteString("\n\n")

	if c.IsWorktree {
		prefix := m.parentAliasFor(c) + "/"
		b.WriteString("  alias › ")
		b.WriteString(StylePath.Render(prefix))
		b.WriteString(StyleNameSel.Render(m.nameBuf + "▌"))
		b.WriteString("\n\n")
		b.WriteString(StylePath.Render("  leave empty to skip aliasing this worktree\n"))
		b.WriteString(StylePath.Render("  (it's still saved; find it by expanding the parent in the picker)"))
		b.WriteString("\n\n")
		b.WriteString(StyleFooter.Render("⏎ next (empty = skip) · esc back · ctrl+c cancel"))
	} else {
		b.WriteString("  name › ")
		b.WriteString(StyleNameSel.Render(m.nameBuf + "▌"))
		b.WriteString("\n\n")
		b.WriteString(StyleFooter.Render("⏎ next · esc back · ctrl+c cancel"))
	}
	return b.String()
}

func (m *addWizardModel) viewGroup() string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render("ws · add — group (optional)"))
	b.WriteString("\n\n")
	b.WriteString("  group › ")
	b.WriteString(StyleNameSel.Render(m.groupBuf + "▌"))
	b.WriteString(StylePath.Render("   (leave empty for no group)"))
	b.WriteString("\n\n")
	if len(m.cfg.Groups) > 0 {
		var names []string
		for _, g := range m.cfg.Groups {
			names = append(names, g.Name)
		}
		b.WriteString(StylePath.Render("  existing: " + strings.Join(names, ", ")))
		b.WriteString("\n\n")
	}
	b.WriteString(StyleFooter.Render("⏎ confirm · ctrl+c cancel"))
	return b.String()
}

// BasenameDefault returns a reasonable default alias from a filesystem path.
func BasenameDefault(p string) string {
	return filepath.Base(p)
}
