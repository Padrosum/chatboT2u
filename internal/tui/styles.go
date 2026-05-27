package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBotA    = lipgloss.Color("#7C3AED") // mor
	colorBotB    = lipgloss.Color("#059669") // yeşil
	colorSystem  = lipgloss.Color("#6B7280") // gri
	colorAccent  = lipgloss.Color("#F59E0B") // turuncu
	colorError   = lipgloss.Color("#EF4444") // kırmızı
	colorBorder  = lipgloss.Color("#374151")
	colorBg      = lipgloss.Color("#111827")
	colorText    = lipgloss.Color("#F9FAFB")
	colorMuted   = lipgloss.Color("#9CA3AF")

	styleBotAHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBotA).
			PaddingLeft(1)

	styleBotBHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBotB).
			PaddingLeft(1)

	styleBotAMsg = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2).
			PaddingRight(1)

	styleBotBMsg = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2).
			PaddingRight(1)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	stylePanelActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
			Background(colorBorder).
			Foreground(colorText).
			Padding(0, 1).
			Bold(true)

	styleButton = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(colorText).
			Padding(0, 2).
			MarginRight(1)

	styleButtonActive = lipgloss.NewStyle().
				Background(colorAccent).
				Foreground(lipgloss.Color("#000000")).
				Padding(0, 2).
				MarginRight(1).
				Bold(true)

	styleButtonBotA = lipgloss.NewStyle().
			Background(colorBotA).
			Foreground(colorText).
			Padding(0, 2).
			MarginRight(1).
			Bold(true)

	styleButtonBotB = lipgloss.NewStyle().
			Background(colorBotB).
			Foreground(colorText).
			Padding(0, 2).
			MarginRight(1).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true).
			PaddingLeft(1)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleThinking = lipgloss.NewStyle().
			Foreground(colorAccent).
			Italic(true).
			PaddingLeft(2)

	styleTitleBar = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Align(lipgloss.Center)
)
