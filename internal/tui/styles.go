package tui

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var useStderrRendererOnce sync.Once

// useStderrRenderer points lipgloss's default renderer at stderr. The shell
// wrapper captures stdout (`out=$(ws …)`), so lipgloss's default, which probes
// stdout, drops to no-color. Called from the TUI entrypoints (not package
// init) because constructing a renderer can probe the terminal, which stalls
// in non-TTY shells for subcommands that never draw a TUI.
func useStderrRenderer() {
	useStderrRendererOnce.Do(func() {
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr))
	})
}

var (
	StyleGroup    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7"))
	StyleName     = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
	StyleNameSel  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f7768e"))
	StylePath     = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#9aa5ce"))
	StyleIcon     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	StyleFooter   = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Padding(0, 1)
	StyleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")).Padding(0, 1)
	StyleSearch   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	StyleCheckbox = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	StyleMissing  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	StyleLogo     = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Bold(true)
	StyleTag      = lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Italic(true)
)

// LogoANSIShadow is the large "ws" ASCII-art logo (ANSI Shadow figlet style).
const LogoANSIShadow = `██╗    ██╗███████╗
██║    ██║██╔════╝
██║ █╗ ██║███████╗
██║███╗██║╚════██║
╚███╔███╔╝███████║
 ╚══╝╚══╝ ╚══════╝`
