package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// init honors SSHW_BACKGROUND=light|dark to force lipgloss's adaptive-color
// detection. Auto-detection (termenv OSC 11 query) fails on many setups —
// nested tmux, ssh-into-Linux from a light-themed Mac terminal, VS Code's
// integrated terminal, etc. — and a wrong guess makes the non-cursor rows
// unreadable (default fg ≈ background). Users hit this in dogfood and the
// only sane fix is letting them override.
func init() {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SSHW_BACKGROUND"))) {
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	}
}

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
	selAliasStyle    = lipgloss.NewStyle().Foreground(colorPrimary)
	norAliasStyle    = lipgloss.NewStyle().Foreground(colorDim)
)

// Health check indicators
var (
	colorHealthOK   = lipgloss.AdaptiveColor{Light: "34", Dark: "82"}
	colorHealthFail = lipgloss.AdaptiveColor{Light: "160", Dark: "204"}
)

var (
	healthOKStyle       = lipgloss.NewStyle().Foreground(colorHealthOK).Bold(true)
	healthFailStyle     = lipgloss.NewStyle().Foreground(colorHealthFail).Bold(true)
	healthCheckingStyle = lipgloss.NewStyle().Foreground(colorDim)
)

// Selected list row background (full-width bar). The Light value used to
// be 254 (near-white), which was effectively invisible on default light
// terminal backgrounds. Switched to a saturated blue tint that holds
// contrast on both light and dark themes; inner foreground styles render
// on top so column colors still come through.
var selRowStyle = lipgloss.NewStyle().
	Background(lipgloss.AdaptiveColor{Light: "153", Dark: "24"})

// Batch execution styles (prompt, results, detail)
var (
	batchMarkOnStyle   = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	batchMarkOffStyle  = lipgloss.NewStyle().Foreground(colorDim)
	batchPromptStyle   = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	batchHintStyle     = lipgloss.NewStyle().Foreground(colorDim)
	batchExitOKStyle   = lipgloss.NewStyle().Foreground(colorHealthOK).Bold(true)
	batchExitFailStyle = lipgloss.NewStyle().Foreground(colorHealthFail).Bold(true)
	batchSectionStyle  = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	batchMetaStyle     = lipgloss.NewStyle().Foreground(colorDim)
	batchCmdStyle      = lipgloss.NewStyle().Foreground(colorText).Bold(true)

	// dangerHighlightStyle marks the dangerous substring inside a command.
	dangerHighlightStyle = lipgloss.NewStyle().
				Foreground(colorHealthFail).
				Bold(true).
				Underline(true)

	// Detail-view tab bar.
	detailTabActiveStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Underline(true)
	detailTabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorDim)
)
