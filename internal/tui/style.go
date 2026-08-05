package tui

import (
	"hash/fnv"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin-Mocha-ish palette, matching artifacts/tui.html.
const (
	colorBorder = lipgloss.Color("#cba6f7")
	colorDim    = lipgloss.Color("#6c7086")
	colorOK     = lipgloss.Color("#a6e3a1")
	colorError  = lipgloss.Color("#f38ba8")
	colorInfo   = lipgloss.Color("#f9e2af")
	colorJoin   = lipgloss.Color("#a6e3a1")
	colorLeave  = lipgloss.Color("#9399b2")
	colorBadge  = lipgloss.Color("#11111b")
)

// userPalette rotates sender name colors, chosen for contrast against a dark background.
var userPalette = []lipgloss.Color{
	"#cba6f7", // mauve
	"#89dceb", // sky
	"#fab387", // peach
	"#94e2d5", // teal
	"#eba0ac", // maroon
	"#f9e2af", // yellow
	"#a6e3a1", // green
	"#89b4fa", // blue
}

var (
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().Foreground(colorDim)

	badgeStyle = lipgloss.NewStyle().
			Background(colorBorder).
			Foreground(colorBadge).
			Bold(true).
			Padding(0, 1)

	promptStyle      = lipgloss.NewStyle().Foreground(colorBorder).Bold(true)
	placeholderStyle = lipgloss.NewStyle().Foreground(colorDim)

	okStyle    = lipgloss.NewStyle().Foreground(colorOK)
	errorStyle = lipgloss.NewStyle().Foreground(colorError)
	infoStyle  = lipgloss.NewStyle().Foreground(colorInfo)
	dimStyle   = lipgloss.NewStyle().Foreground(colorDim)
	joinStyle  = lipgloss.NewStyle().Foreground(colorJoin)
	leaveStyle = lipgloss.NewStyle().Foreground(colorLeave)
)

// userStyle returns a bold, colored style for name, stable across the session.
func userStyle(name string) lipgloss.Style {
	h := fnv.New32a()
	h.Write([]byte(name))
	color := userPalette[h.Sum32()%uint32(len(userPalette))]
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}
