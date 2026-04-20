package tui

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var useStderrRendererOnce sync.Once

// useStderrRenderer points lipgloss at stderr. The shell wrapper captures
// stdout via `out=$(ws …)`, so lipgloss's boot-time probe of stdout lands on
// NoColor and bakes that into every style built against the default renderer
// — styles must be rebuilt against stderr for colors to survive. Not called
// from package init because NewRenderer probes the terminal, which stalls
// non-TTY subcommands that never draw a TUI.
func useStderrRenderer() {
	useStderrRendererOnce.Do(func() {
		r := lipgloss.NewRenderer(os.Stderr)
		lipgloss.SetDefaultRenderer(r)
		buildStyles(r)
	})
}

func buildStyles(r *lipgloss.Renderer) {
	StyleGroup = r.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7"))
	StyleName = r.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
	StyleNameSel = r.NewStyle().Bold(true).Foreground(lipgloss.Color("#f7768e"))
	StylePath = r.NewStyle().Faint(true).Foreground(lipgloss.Color("#9aa5ce"))
	StyleIcon = r.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	StyleFooter = r.NewStyle().Foreground(lipgloss.Color("#565f89")).Padding(0, 1)
	StyleTitle = r.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")).Padding(0, 1)
	StyleSearch = r.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	StyleCheckbox = r.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	StyleMissing = r.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	StyleLogo = r.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Bold(true)
	StyleTag = r.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Italic(true)
}

var (
	StyleGroup    lipgloss.Style
	StyleName     lipgloss.Style
	StyleNameSel  lipgloss.Style
	StylePath     lipgloss.Style
	StyleIcon     lipgloss.Style
	StyleFooter   lipgloss.Style
	StyleTitle    lipgloss.Style
	StyleSearch   lipgloss.Style
	StyleCheckbox lipgloss.Style
	StyleMissing  lipgloss.Style
	StyleLogo     lipgloss.Style
	StyleTag      lipgloss.Style
)

func init() {
	buildStyles(lipgloss.DefaultRenderer())
}

// LogoANSIShadow is the large "ws" ASCII-art logo (ANSI Shadow figlet style).
const LogoANSIShadow = `██╗    ██╗███████╗
██║    ██║██╔════╝
██║ █╗ ██║███████╗
██║███╗██║╚════██║
╚███╔███╔╝███████║
 ╚══╝╚══╝ ╚══════╝`
