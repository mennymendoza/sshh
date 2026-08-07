package tui

import (
	"hash/fnv"

	"github.com/charmbracelet/lipgloss"
)

const (
	colorBorder = lipgloss.Color("#bb9af7")
	colorDim    = lipgloss.Color("#565f89")
	colorOK     = lipgloss.Color("#9ece6a")
	colorError  = lipgloss.Color("#f7768e")
	colorInfo   = lipgloss.Color("#e0af68")
	colorJoin   = lipgloss.Color("#9ece6a")
	colorLeave  = lipgloss.Color("#565f89")
	colorBadge  = lipgloss.Color("#1a1b26")
)

var userPalette = []lipgloss.Color{
	"#7aa2f7",
	"#7dcfff",
	"#bb9af7",
	"#9ece6a",
	"#e0af68",
	"#ff9e64",
	"#f7768e",
	"#73daca",
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

func userStyle(name string) lipgloss.Style {
	h := fnv.New32a()
	h.Write([]byte(name))
	color := userPalette[h.Sum32()%uint32(len(userPalette))]
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}
