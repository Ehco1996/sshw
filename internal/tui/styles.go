package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors for light/dark terminal support
var (
	colorPrimary = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}
	colorText    = lipgloss.AdaptiveColor{Light: "235", Dark: "252"}
	colorDim     = lipgloss.AdaptiveColor{Light: "244", Dark: "243"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "250", Dark: "238"}
	colorHost    = lipgloss.AdaptiveColor{Light: "30", Dark: "37"}
	colorUser    = lipgloss.AdaptiveColor{Light: "133", Dark: "176"}
	colorPort    = lipgloss.AdaptiveColor{Light: "244", Dark: "243"}
	colorDir     = lipgloss.AdaptiveColor{Light: "208", Dark: "214"}
)

// List item styles
var (
	cursorStyle        = lipgloss.NewStyle().Foreground(colorPrimary)
	selNameStyle       = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	selHostStyle       = lipgloss.NewStyle().Foreground(colorHost).Bold(true)
	selUserStyle       = lipgloss.NewStyle().Foreground(colorUser).Bold(true)
	selPortStyle       = lipgloss.NewStyle().Foreground(colorPort).Bold(true)
	selJumpStyle       = lipgloss.NewStyle().Foreground(colorDim).Bold(true)
	selDirBadgeStyle   = lipgloss.NewStyle().Foreground(colorDir).Bold(true)
	selDirPreviewStyle = lipgloss.NewStyle().Foreground(colorDim).Bold(true)
	norNameStyle       = lipgloss.NewStyle().Foreground(colorText)
	norHostStyle       = lipgloss.NewStyle().Foreground(colorHost)
	norUserStyle       = lipgloss.NewStyle().Foreground(colorUser)
	norPortStyle       = lipgloss.NewStyle().Foreground(colorPort)
	norJumpStyle       = lipgloss.NewStyle().Foreground(colorDim)
	norDirBadgeStyle   = lipgloss.NewStyle().Foreground(colorDir)
	norDirPreviewStyle = lipgloss.NewStyle().Foreground(colorDim)
)

// Header / footer
var (
	headerTitleStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	headerPathStyle  = lipgloss.NewStyle().Foreground(colorDim)
	headerSepStyle   = lipgloss.NewStyle().Foreground(colorPrimary)
	headerCountStyle = lipgloss.NewStyle().Foreground(colorDim)
	separatorStyle   = lipgloss.NewStyle().Foreground(colorSubtle)
	helpKeyStyle     = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	helpDescStyle    = lipgloss.NewStyle().Foreground(colorDim)
	breadcrumbStyle  = lipgloss.NewStyle().Foreground(colorDim)
)
