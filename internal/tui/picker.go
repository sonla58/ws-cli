package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"

	"github.com/sonla58/ws-cli/internal/detect"
	"github.com/sonla58/ws-cli/internal/model"
	"github.com/sahilm/fuzzy"
)

// Action returned from the picker to let the caller react.
type Action int

const (
	ActionNone   Action = iota // cancelled
	ActionChoose               // user picked a path
	ActionAdd                  // user asked to add (empty state or `a` key)
)

// Result of running the picker.
type Result struct {
	Action Action
	Path   string
}

// row is either a group header, a workspace row, or a worktree leaf.
type rowKind int

const (
	rowGroup rowKind = iota
	rowWorkspace
	rowWorktree
)

type row struct {
	kind   rowKind
	group  string
	w      *model.Workspace // for workspace and worktree rows
	wt     *model.Worktree  // for worktree rows only
	depth  int
	isRoot bool // when kind==rowWorktree, synthesizes the parent root as a leaf
}

type pickerModel struct {
	cfg           model.Config
	rows          []row
	cursor        int
	expanded      map[string]bool // workspace name -> expanded
	search        string
	inSearch      bool
	width, height int
	result        Result
	quitting      bool

	// restrict: if non-empty, only show these workspace names.
	restrict map[string]bool
}

// RunPicker shows the grouped TUI picker.
func RunPicker(cfg model.Config) (Result, error) {
	return runPickerWith(cfg, nil)
}

// RunRestrictedPicker shows the picker filtered to the given workspace names.
func RunRestrictedPicker(cfg model.Config, names []string) (Result, error) {
	r := make(map[string]bool, len(names))
	for _, n := range names {
		r[n] = true
	}
	return runPickerWith(cfg, r)
}

func runPickerWith(cfg model.Config, restrict map[string]bool) (Result, error) {
	m := &pickerModel{
		cfg:      cfg,
		expanded: map[string]bool{},
		restrict: restrict,
	}
	// Auto-expand when a restricted workspace has worktrees (so user sees choices).
	for i := range cfg.Workspaces {
		w := &cfg.Workspaces[i]
		if restrict != nil && restrict[w.Name] && len(w.Worktrees) > 0 {
			m.expanded[w.Name] = true
		}
	}
	m.rebuild()

	useStderrRenderer()
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	fm := final.(*pickerModel)
	if fm.quitting {
		return Result{Action: ActionNone}, nil
	}
	return fm.result, nil
}

func (m *pickerModel) rebuild() {
	m.rows = nil

	groups := map[string][]*model.Workspace{}
	groupOrder := []string{}
	for _, g := range m.cfg.Groups {
		groups[g.Name] = nil
		groupOrder = append(groupOrder, g.Name)
	}
	for i := range m.cfg.Workspaces {
		w := &m.cfg.Workspaces[i]
		if m.restrict != nil && !m.restrict[w.Name] {
			continue
		}
		if m.search != "" && !matches(w, m.search) {
			continue
		}
		g := w.Group
		if _, ok := groups[g]; !ok && g != "" {
			groups[g] = nil
			groupOrder = append(groupOrder, g)
		}
		groups[g] = append(groups[g], w)
	}
	sort.Strings(groupOrder)

	emit := func(g string) {
		ws := groups[g]
		if len(ws) == 0 {
			return
		}
		label := g
		if label == "" {
			label = "ungrouped"
		}
		m.rows = append(m.rows, row{kind: rowGroup, group: label})
		sort.Slice(ws, func(i, j int) bool { return ws[i].Name < ws[j].Name })
		for _, w := range ws {
			m.rows = append(m.rows, row{kind: rowWorkspace, w: w, depth: 1})
			if m.expanded[w.Name] {
				// Synthetic "root" leaf so user can pick the main worktree
				// directly when the workspace has additional worktrees.
				m.rows = append(m.rows, row{kind: rowWorktree, w: w, depth: 2, isRoot: true})
				for k := range w.Worktrees {
					m.rows = append(m.rows, row{kind: rowWorktree, w: w, wt: &w.Worktrees[k], depth: 2})
				}
			}
		}
	}
	for _, g := range groupOrder {
		if g != "" {
			emit(g)
		}
	}
	emit("")

	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	for m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowGroup {
		m.cursor++
	}
}

func matches(w *model.Workspace, q string) bool {
	names := []string{w.Name, w.Path}
	for _, wt := range w.Worktrees {
		names = append(names, wt.Name, wt.Path)
	}
	if len(fuzzy.Find(q, names)) > 0 {
		return true
	}
	lq := strings.ToLower(q)
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), lq) {
			return true
		}
	}
	return false
}

func (m *pickerModel) Init() tea.Cmd { return nil }

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		// Empty state: only handle a few keys.
		if len(m.cfg.Workspaces) == 0 {
			switch msg.String() {
			case "a", "enter":
				m.result = Result{Action: ActionAdd}
				return m, tea.Quit
			case "q", "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		if m.inSearch {
			return m.updateSearch(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "home", "g":
			m.cursor = 0
			m.skipToSelectable(1)
		case "end", "G":
			m.cursor = len(m.rows) - 1
			m.skipToSelectable(-1)
		case "/":
			m.inSearch = true
		case "a":
			m.result = Result{Action: ActionAdd}
			return m, tea.Quit
		case "enter":
			m.choose()
			if m.result.Action == ActionChoose {
				return m, tea.Quit
			}
		case "right", "l":
			m.expand(true)
		case "left", "h":
			m.expand(false)
		}
	}
	return m, nil
}

func (m *pickerModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inSearch = false
		m.search = ""
		m.rebuild()
	case "enter":
		m.inSearch = false
	case "backspace":
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
			m.rebuild()
		}
	default:
		if len(msg.String()) == 1 {
			m.search += msg.String()
			m.rebuild()
		}
	}
	return m, nil
}

func (m *pickerModel) moveCursor(d int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	m.skipToSelectable(d)
}

func (m *pickerModel) skipToSelectable(d int) {
	step := d
	if step == 0 {
		step = 1
	}
	for m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowGroup {
		m.cursor += step
	}
	if m.cursor < 0 {
		m.cursor = 0
		for m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowGroup {
			m.cursor++
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
		for m.cursor >= 0 && m.rows[m.cursor].kind == rowGroup {
			m.cursor--
		}
	}
}

func (m *pickerModel) expand(open bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	r := m.rows[m.cursor]
	if r.kind != rowWorkspace || len(r.w.Worktrees) == 0 {
		return
	}
	m.expanded[r.w.Name] = open
	m.rebuild()
}

func (m *pickerModel) choose() {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	r := m.rows[m.cursor]
	switch r.kind {
	case rowWorktree:
		if r.isRoot {
			m.result = Result{Action: ActionChoose, Path: r.w.Path}
			return
		}
		m.result = Result{Action: ActionChoose, Path: r.wt.Path}
	case rowWorkspace:
		if len(r.w.Worktrees) == 0 {
			m.result = Result{Action: ActionChoose, Path: r.w.Path}
			return
		}
		if !m.expanded[r.w.Name] {
			m.expanded[r.w.Name] = true
			m.rebuild()
			return
		}
		// Already expanded → pressing Enter on the root row also opens the root.
		m.result = Result{Action: ActionChoose, Path: r.w.Path}
	}
}

func (m *pickerModel) View() string {
	if len(m.cfg.Workspaces) == 0 {
		return m.viewEmpty()
	}
	var b strings.Builder
	b.WriteString(StyleTitle.Render("ws · workspaces"))
	if m.search != "" || m.inSearch {
		b.WriteString("  ")
		b.WriteString(StyleSearch.Render("/" + m.search))
	}
	b.WriteString("\n\n")

	maxRows := m.height - 5
	if maxRows <= 0 {
		maxRows = 20
	}
	start := 0
	if m.cursor > maxRows-3 {
		start = m.cursor - (maxRows - 3)
	}
	end := start + maxRows
	if end > len(m.rows) {
		end = len(m.rows)
	}

	for i := start; i < end; i++ {
		r := m.rows[i]
		switch r.kind {
		case rowGroup:
			b.WriteString(StyleGroup.Render("▸ " + r.group))
			b.WriteString("\n")
		case rowWorkspace:
			b.WriteString(m.renderWorkspace(r, i == m.cursor))
			b.WriteString("\n")
		case rowWorktree:
			b.WriteString(m.renderWorktree(r, i == m.cursor))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(StyleFooter.Render("↑/↓ move · ⏎ open · →/← expand · / search · a add · q quit"))
	return b.String()
}

func (m *pickerModel) viewEmpty() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	logo := StyleLogo.Render(LogoANSIShadow)
	tag := StyleTag.Render("— welcome —")
	blurb := StylePath.Render("Fast, git-aware workspace manager.\nSave projects, jump into them with ") +
		StyleSearch.Render("ws <name>") + StylePath.Render(".")

	bullet1 := "  " + StyleCheckbox.Render("▸") + "  " +
		StyleNameSel.Render("press a") + StyleName.Render("  to add the current directory")
	bullet2 := "  " + StylePath.Render("·") + "  " +
		StyleName.Render("or run ") + StyleSearch.Render("ws add <path>") +
		StyleName.Render(" from the shell")

	center := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(center.Render(logo))
	b.WriteString("\n")
	b.WriteString(center.Render(tag))
	b.WriteString("\n\n")
	b.WriteString(center.Render(blurb))
	b.WriteString("\n\n\n")
	b.WriteString(center.Render(bullet1))
	b.WriteString("\n")
	b.WriteString(center.Render(bullet2))
	b.WriteString("\n\n")
	b.WriteString(center.Render(StyleFooter.Render("a add current dir · ⏎ add · q quit")))
	return b.String()
}

func (m *pickerModel) renderWorkspace(r row, selected bool) string {
	indent := strings.Repeat("  ", r.depth)
	icon := StyleIcon.Render(detect.Icon(detect.Type(r.w.Icon)))
	name := r.w.Name
	caret := " "
	if len(r.w.Worktrees) > 0 {
		if m.expanded[r.w.Name] {
			caret = "▾"
		} else {
			caret = "▸"
		}
	}
	nameStyle := StyleName
	if selected {
		nameStyle = StyleNameSel
	}
	return fmt.Sprintf("%s%s %s %s  %s",
		indent, caret, icon, nameStyle.Render(name), StylePath.Render(r.w.Path))
}

func (m *pickerModel) renderWorktree(r row, selected bool) string {
	indent := strings.Repeat("  ", r.depth)
	var iconKey, name, suffix, path string
	if r.isRoot {
		iconKey = r.w.Icon
		name = r.w.Name
		suffix = "  (root)"
		path = r.w.Path
	} else {
		iconKey = r.wt.Icon
		name = worktreeDisplay(r.wt)
		path = r.wt.Path
	}
	icon := StyleIcon.Render(detect.Icon(detect.Type(iconKey)))
	nameStyle := StyleName
	if selected {
		nameStyle = StyleNameSel
	}
	return fmt.Sprintf("%s· %s %s%s  %s",
		indent, icon, nameStyle.Render(name), StylePath.Render(suffix), StylePath.Render(path))
}

// worktreeDisplay returns wt.Name if set, else the basename of the path
// rendered with a subtle "unaliased" hint.
func worktreeDisplay(wt *model.Worktree) string {
	if wt.Name != "" {
		return wt.Name
	}
	return filepath.Base(wt.Path)
}
