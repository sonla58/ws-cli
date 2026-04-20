package tui

import "github.com/charmbracelet/lipgloss"

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
