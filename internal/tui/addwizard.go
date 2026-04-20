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
	order      []int // indices into candidates shown on the naming form, sorted roots-first
	cursor     int  // position into order[] on stepNames (and stepSelect for selection)
	selectCur  int  // cursor for stepSelect
	step       wizardStep

	// aliased[candidateIdx] == true means this worktree's alias field is active.
	// Roots are always "active" and absent from this map.
	aliased map[int]bool

	groupBuf  string
	cancelled bool
}

// RunAddWizard runs the wizard and returns the finalized candidates. Only
// those with Selected=true should be written.
func RunAddWizard(cfg model.Config, candidates []Candidate) ([]Candidate, bool, error) {
	for i := range candidates {
		if candidates[i].Icon == "" {
			candidates[i].Icon = string(detect.DetectType(candidates[i].Path))
		}
		candidates[i].Selected = true
	}
	m := &addWizardModel{
		cfg:        cfg,
		candidates: candidates,
		aliased:    map[int]bool{},
	}
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

// ---- step: select ----

func (m *addWizardModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		if m.selectCur > 0 {
			m.selectCur--
		}
	case "down", "j":
		if m.selectCur < len(m.candidates)-1 {
			m.selectCur++
		}
	case " ":
		m.candidates[m.selectCur].Selected = !m.candidates[m.selectCur].Selected
	case "a":
		for i := range m.candidates {
			m.candidates[i].Selected = true
		}
	case "n":
		for i := range m.candidates {
			m.candidates[i].Selected = false
		}
	case "enter":
		m.buildNameOrder()
		if len(m.order) == 0 {
			m.cancelled = true
			return m, tea.Quit
		}
		// Pre-fill root buffers with DefaultName so Enter-through works.
		for _, i := range m.order {
			c := &m.candidates[i]
			if !c.IsWorktree && c.Name == "" {
				c.Name = c.DefaultName
			}
		}
		m.cursor = 0
		m.step = stepNames
	}
	return m, nil
}

// buildNameOrder lists selected candidate indices, roots first (so the prefix
// preview for worktrees has a known parent alias on the same screen).
func (m *addWizardModel) buildNameOrder() {
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
}

// ---- step: names (single-screen form) ----

func (m *addWizardModel) updateNames(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "esc":
		m.step = stepSelect
		return m, nil
	case "ctrl+s":
		m.commitNames()
		m.step = stepGroup
		return m, nil
	case "up", "shift+tab":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "tab":
		if m.cursor < len(m.order)-1 {
			m.cursor++
		}
	case "enter":
		// Enter moves to next field; on the last field it submits.
		if m.cursor < len(m.order)-1 {
			m.cursor++
			return m, nil
		}
		m.commitNames()
		m.step = stepGroup
		return m, nil
	case "alt+a":
		// alt+a toggles aliasing on a worktree row (non-destructive: keeps buffer).
		return m.toggleAlias(), nil
	case "backspace":
		idx := m.order[m.cursor]
		c := &m.candidates[idx]
		if m.fieldActive(idx) && len(c.Name) > 0 {
			c.Name = c.Name[:len(c.Name)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 {
			idx := m.order[m.cursor]
			c := &m.candidates[idx]
			// On an inactive worktree row, pressing "a" toggles aliasing on.
			// Any other letter also switches the row on and starts typing.
			if c.IsWorktree && !m.aliased[idx] {
				if s == "a" {
					m.aliased[idx] = true
					return m, nil
				}
				// Ignore stray keys on inactive rows to avoid surprises.
				return m, nil
			}
			c.Name += s
		}
	}
	return m, nil
}

func (m *addWizardModel) toggleAlias() tea.Model {
	idx := m.order[m.cursor]
	c := &m.candidates[idx]
	if !c.IsWorktree {
		return m
	}
	m.aliased[idx] = !m.aliased[idx]
	return m
}

// fieldActive reports whether the candidate at idx should be treated as
// having an active alias (roots always; worktrees only when enabled).
func (m *addWizardModel) fieldActive(idx int) bool {
	c := m.candidates[idx]
	if !c.IsWorktree {
		return true
	}
	return m.aliased[idx]
}

// commitNames enforces invariants before advancing: roots get DefaultName if
// blank; inactive worktree fields get cleared to signal "unaliased".
func (m *addWizardModel) commitNames() {
	for _, idx := range m.order {
		c := &m.candidates[idx]
		if c.IsWorktree {
			if !m.aliased[idx] {
				c.Name = ""
			} else {
				c.Name = strings.TrimSpace(c.Name)
			}
			continue
		}
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" {
			c.Name = c.DefaultName
		}
	}
}

// parentAliasFor returns the parent root's alias for a worktree candidate,
// preferring an in-flight buffer over saved config.
func (m *addWizardModel) parentAliasFor(c Candidate) string {
	for _, cc := range m.candidates {
		if cc.Selected && !cc.IsWorktree && filepath.Clean(cc.Path) == filepath.Clean(c.ParentRoot) {
			if strings.TrimSpace(cc.Name) != "" {
				return strings.TrimSpace(cc.Name)
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

// ---- step: group ----

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

// ---- views ----

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
		if i == m.selectCur {
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
	var b strings.Builder
	b.WriteString(StyleTitle.Render("ws · add — name your workspaces"))
	b.WriteString("\n\n")
	b.WriteString(StylePath.Render("  Use ↑/↓ to move between rows. On a worktree row, press "))
	b.WriteString(StyleNameSel.Render("a"))
	b.WriteString(StylePath.Render(" to enable aliasing."))
	b.WriteString("\n\n")

	for row, idx := range m.order {
		c := m.candidates[idx]
		focused := row == m.cursor
		b.WriteString(m.renderNameRow(c, idx, focused))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleFooter.Render(
		"↑/↓ move · ⏎ next/save · a enable alias · ctrl+s save · esc back · ctrl+c cancel"))
	return b.String()
}

func (m *addWizardModel) renderNameRow(c Candidate, idx int, focused bool) string {
	icon := StyleIcon.Render(detect.Icon(detect.Type(c.Icon)))
	caret := "  "
	if focused {
		caret = StyleNameSel.Render("› ")
	}

	header := fmt.Sprintf("%s%s  %s", caret, icon, StylePath.Render(c.Path))

	var input string
	if !c.IsWorktree {
		input = "    name  › " + renderInput(c.Name, focused)
	} else if m.aliased[idx] {
		prefix := m.parentAliasFor(c) + "/"
		input = "    alias › " + StylePath.Render(prefix) + renderInput(c.Name, focused)
	} else {
		hint := StylePath.Render("(unaliased — press ") +
			StyleNameSel.Render("a") +
			StylePath.Render(" to alias this worktree)")
		input = "    alias › " + hint
	}
	return header + "\n" + input
}

func renderInput(buf string, focused bool) string {
	cursor := ""
	if focused {
		cursor = "▌"
	}
	return StyleNameSel.Render(buf + cursor)
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
