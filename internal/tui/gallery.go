package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/padros/chatbot2u/internal/scenarios"
)

type GalleryModel struct {
	items    []scenarios.Scenario
	cursor   int
	selected int // -1 = atlandı
	width    int
	height   int
}

func NewGallery() *GalleryModel {
	return &GalleryModel{
		items:    scenarios.Presets,
		selected: -1,
	}
}

func (g *GalleryModel) Init() tea.Cmd { return nil }

func (g *GalleryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return g, tea.Quit

		case "up", "k":
			if g.cursor > 0 {
				g.cursor--
			}

		case "down", "j":
			if g.cursor < len(g.items)-1 {
				g.cursor++
			}

		case "enter":
			g.selected = g.cursor
			return g, tea.Quit

		case "n", "N", "esc":
			// Atla — selected kalır -1
			return g, tea.Quit
		}
	}
	return g, nil
}

func (g *GalleryModel) Done() bool {
	return g.selected >= 0
}

func (g *GalleryModel) Selected() *scenarios.Scenario {
	if g.selected < 0 || g.selected >= len(g.items) {
		return nil
	}
	return &g.items[g.selected]
}

func (g *GalleryModel) View() string {
	if g.width == 0 {
		return "Yükleniyor..."
	}

	title := styleTitleBar.Width(g.width).Render("◈ chatboT2u — Senaryo Galerisi ◈")

	listW := 36
	previewW := g.width - listW - 7
	if previewW < 20 {
		previewW = 20
	}
	panelH := g.height - 6
	if panelH < 8 {
		panelH = 8
	}

	list := g.renderList(listW, panelH)
	preview := g.renderPreview(previewW, panelH)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", preview)

	help := styleMuted.Width(g.width).Align(lipgloss.Center).Render(
		"↑/↓  j/k ▸ gezin  │  Enter ▸ seç  │  N / Esc ▸ özel kuruluma geç  │  Q ▸ çıkış",
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, "", columns, "", help)
}

func (g *GalleryModel) renderList(w, h int) string {
	var rows []string
	rows = append(rows, styleBotAHeader.Render("Hazır Senaryolar"))
	rows = append(rows, "")

	for i, sc := range g.items {
		var line string
		if i == g.cursor {
			line = styleButtonActive.Render(fmt.Sprintf("▶ %s", sc.Title))
			desc := styleThinking.Render("  " + sc.Description)
			rows = append(rows, line, desc, "")
		} else {
			line = styleMuted.Render(fmt.Sprintf("  %s", sc.Title))
			rows = append(rows, line, "")
		}
	}

	rows = append(rows, "")
	rows = append(rows, styleMuted.Render("─────────────────────"))
	rows = append(rows, styleMuted.Render("  [N] Özel kurulum →"))

	inner := strings.Join(rows, "\n")
	return stylePanel.Width(w).Height(h).Render(inner)
}

func (g *GalleryModel) renderPreview(w, h int) string {
	if len(g.items) == 0 {
		return stylePanel.Width(w).Height(h).Render(styleMuted.Render("Senaryo yok"))
	}

	sc := g.items[g.cursor]

	header := styleTitleBar.Width(w - 4).Render(sc.Title)
	desc := styleMuted.Render(sc.Description)
	seed := styleMuted.Italic(true).Render("\"" + sc.SeedMsg + "\"")

	divider := styleMuted.Render(strings.Repeat("─", w-6))

	botAHeader := styleBotAHeader.Render(fmt.Sprintf("Bot A — %s", sc.BotA.Name))
	botARole := styleMuted.Render(fmt.Sprintf("  Rol: %s  │  Model: %s  │  T=%.1f",
		sc.BotA.Role, sc.BotA.Model, sc.BotA.Temperature))
	botAPrompt := g.wrapPrompt(sc.BotA.SystemPrompt, w-6)

	botBHeader := styleBotBHeader.Render(fmt.Sprintf("Bot B — %s", sc.BotB.Name))
	botBRole := styleMuted.Render(fmt.Sprintf("  Rol: %s  │  Model: %s  │  T=%.1f",
		sc.BotB.Role, sc.BotB.Model, sc.BotB.Temperature))
	botBPrompt := g.wrapPrompt(sc.BotB.SystemPrompt, w-6)

	seedLabel := lipgloss.NewStyle().Foreground(colorAccent).Render("Başlangıç mesajı:")

	inner := strings.Join([]string{
		header,
		desc,
		"",
		seedLabel,
		seed,
		"",
		divider,
		"",
		botAHeader,
		botARole,
		botAPrompt,
		"",
		botBHeader,
		botBRole,
		botBPrompt,
	}, "\n")

	return stylePanel.Width(w).Height(h).Render(inner)
}

// wrapPrompt kısaltır ve görsel sınırlar içinde tutar.
func (g *GalleryModel) wrapPrompt(text string, maxW int) string {
	const maxLen = 200
	if len(text) > maxLen {
		text = text[:maxLen] + "…"
	}

	words := strings.Fields(text)
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len()+len(w)+1 > maxW {
			lines = append(lines, "  "+cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, "  "+cur.String())
	}

	return styleMuted.Render(strings.Join(lines, "\n"))
}
