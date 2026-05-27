package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/padros/chatbot2u/internal/config"
)

// Alan sırası: A-name → A-role → A-model → A-temp → A-prompt →
//              B-name → B-role → B-model → B-temp → B-prompt → seed
type setupField int

const (
	fAName setupField = iota
	fARole
	fAModel
	fATemp
	fAPrompt
	fBName
	fBRole
	fBModel
	fBTemp
	fBPrompt
	fSeed
	fCount
)

type SetupModel struct {
	cfg     *config.AppConfig
	shorts  []*textinput.Model // name, role, model, temp, seed (kısa alanlar)
	taA     textarea.Model
	taB     textarea.Model
	cursor  setupField
	done    bool
	width   int
	height  int
}

// shortIdx maps a setupField to its index in s.shorts slice (nil if textarea)
func shortIdx(f setupField) int {
	switch f {
	case fAName:
		return 0
	case fARole:
		return 1
	case fAModel:
		return 2
	case fATemp:
		return 3
	case fBName:
		return 4
	case fBRole:
		return 5
	case fBModel:
		return 6
	case fBTemp:
		return 7
	case fSeed:
		return 8
	}
	return -1
}

func isTextArea(f setupField) bool {
	return f == fAPrompt || f == fBPrompt
}

func NewSetup(cfg *config.AppConfig) *SetupModel {
	newTI := func(val, placeholder string, w int) *textinput.Model {
		t := textinput.New()
		t.CharLimit = 256
		t.Width = w
		t.Placeholder = placeholder
		t.SetValue(val)
		return &t
	}

	shorts := []*textinput.Model{
		newTI(cfg.BotA.Name, "bot ismi", 20),
		newTI(cfg.BotA.Role, "rol / persona", 20),
		newTI(cfg.BotA.Model, "llama3.2", 20),
		newTI(fmt.Sprintf("%.2f", cfg.BotA.Temperature), "0.00–2.00", 8),
		newTI(cfg.BotB.Name, "bot ismi", 20),
		newTI(cfg.BotB.Role, "rol / persona", 20),
		newTI(cfg.BotB.Model, "llama3.2", 20),
		newTI(fmt.Sprintf("%.2f", cfg.BotB.Temperature), "0.00–2.00", 8),
		newTI("Bilinç nedir? Öznel deneyim var olabilir mi?", "konuşmayı başlatacak mesaj", 60),
	}

	newTA := func(val string, w, h int) textarea.Model {
		ta := textarea.New()
		ta.SetWidth(w)
		ta.SetHeight(h)
		ta.CharLimit = 1024
		ta.ShowLineNumbers = false
		ta.SetValue(val)
		return ta
	}

	taA := newTA(cfg.BotA.SystemPrompt, 38, 4)
	taB := newTA(cfg.BotB.SystemPrompt, 38, 4)

	shorts[0].Focus() // ilk alan aktif

	return &SetupModel{
		cfg:    cfg,
		shorts: shorts,
		taA:    taA,
		taB:    taB,
	}
}

func (s *SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (s *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "esc":
			if isTextArea(s.cursor) {
				// textarea odaktan çık, bir sonraki alana geç
				s.advance()
				return s, nil
			}
			return s, tea.Quit

		case "tab":
			if s.cursor == fSeed {
				return s, s.start()
			}
			s.advance()
			return s, nil

		case "shift+tab":
			s.retreat()
			return s, nil

		case "ctrl+s":
			return s, s.start()

		case "enter":
			// Kısa alanlarda enter = ileri; textarea'da enter = satır sonu
			if !isTextArea(s.cursor) {
				if s.cursor == fSeed {
					return s, s.start()
				}
				s.advance()
				return s, nil
			}
		}
	}

	// Aktif bileşene mesajı ilet
	var cmd tea.Cmd
	switch {
	case isTextArea(s.cursor) && s.cursor == fAPrompt:
		s.taA, cmd = s.taA.Update(msg)
	case isTextArea(s.cursor) && s.cursor == fBPrompt:
		s.taB, cmd = s.taB.Update(msg)
	default:
		if idx := shortIdx(s.cursor); idx >= 0 {
			updated, c := s.shorts[idx].Update(msg)
			s.shorts[idx] = &updated
			cmd = c
		}
	}
	return s, cmd
}

// start finalizes config and signals bubbletea to exit.
func (s *SetupModel) start() tea.Cmd {
	s.applyToConfig()
	s.done = true
	return tea.Quit
}

func (s *SetupModel) advance() {
	s.blur(s.cursor)
	if s.cursor < fSeed {
		s.cursor++
		s.focus(s.cursor)
	}
	// fSeed'den sonrası yok; Tab son alanda tıklandıysa bir şey yapma.
	// Kullanıcı Enter veya Ctrl+S ile başlatır.
}

func (s *SetupModel) retreat() {
	if s.cursor == 0 {
		return
	}
	s.blur(s.cursor)
	s.cursor--
	s.focus(s.cursor)
}

func (s *SetupModel) focus(f setupField) {
	if isTextArea(f) {
		if f == fAPrompt {
			s.taA.Focus()
		} else {
			s.taB.Focus()
		}
	} else {
		if idx := shortIdx(f); idx >= 0 {
			s.shorts[idx].Focus()
		}
	}
}

func (s *SetupModel) blur(f setupField) {
	if isTextArea(f) {
		if f == fAPrompt {
			s.taA.Blur()
		} else {
			s.taB.Blur()
		}
	} else {
		if idx := shortIdx(f); idx >= 0 {
			s.shorts[idx].Blur()
		}
	}
}

func (s *SetupModel) applyToConfig() {
	v := func(idx int) string { return strings.TrimSpace(s.shorts[idx].Value()) }
	orDef := func(val, def string) string {
		if val == "" {
			return def
		}
		return val
	}
	parseTemp := func(val string, def float64) float64 {
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || f < 0 || f > 2 {
			return def
		}
		return f
	}

	s.cfg.BotA.Name = orDef(v(0), s.cfg.BotA.Name)
	s.cfg.BotA.Role = orDef(v(1), s.cfg.BotA.Role)
	s.cfg.BotA.Model = orDef(v(2), s.cfg.BotA.Model)
	s.cfg.BotA.Temperature = parseTemp(v(3), s.cfg.BotA.Temperature)
	s.cfg.BotA.SystemPrompt = orDef(strings.TrimSpace(s.taA.Value()), s.cfg.BotA.SystemPrompt)

	s.cfg.BotB.Name = orDef(v(4), s.cfg.BotB.Name)
	s.cfg.BotB.Role = orDef(v(5), s.cfg.BotB.Role)
	s.cfg.BotB.Model = orDef(v(6), s.cfg.BotB.Model)
	s.cfg.BotB.Temperature = parseTemp(v(7), s.cfg.BotB.Temperature)
	s.cfg.BotB.SystemPrompt = orDef(strings.TrimSpace(s.taB.Value()), s.cfg.BotB.SystemPrompt)
}

func (s *SetupModel) SeedMsg() string {
	return strings.TrimSpace(s.shorts[8].Value())
}

func (s *SetupModel) Done() bool { return s.done }

// ── Görünüm ─────────────────────────────────────────────────────────────────

func (s *SetupModel) View() string {
	if s.width == 0 {
		return "Yükleniyor..."
	}

	title := styleTitleBar.Width(s.width).Render("◈ chatboT2u — Bot Kimlik Kurulumu ◈")

	colW := (s.width - 6) / 2
	if colW < 36 {
		colW = 36
	}

	colA := s.renderBotCol("BOT A", colorBotA, fAName, fARole, fAModel, fATemp, fAPrompt, s.taA, colW)
	colB := s.renderBotCol("BOT B", colorBotB, fBName, fBRole, fBModel, fBTemp, fBPrompt, s.taB, colW)

	cols := lipgloss.JoinHorizontal(lipgloss.Top, colA, "  ", colB)

	// Başlangıç mesajı
	seedActive := s.cursor == fSeed
	seedLabel := s.fieldLabel("Başlangıç mesajı", seedActive, lipgloss.Color("#F59E0B"))
	seedBox := lipgloss.NewStyle().Width(s.width - 4).Render(
		seedLabel + "\n" + s.shorts[8].View(),
	)

	// Yardım satırı
	var hints []string
	if isTextArea(s.cursor) {
		hints = append(hints, "Enter ▸ satır sonu")
		hints = append(hints, "Esc/Tab ▸ ileri")
	} else {
		hints = append(hints, "Tab/Enter ▸ ileri")
	}
	hints = append(hints, "Shift+Tab ▸ geri", "Ctrl+S ▸ Başlat", "Esc ▸ çıkış")
	help := styleMuted.Width(s.width).Align(lipgloss.Center).Render(strings.Join(hints, "  │  "))

	// İlerleme göstergesi
	total := int(fCount)
	cur := int(s.cursor) + 1
	progress := styleMuted.Render(fmt.Sprintf("Alan %d/%d", cur, total))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		cols,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(seedBox),
		"",
		help,
		lipgloss.NewStyle().Width(s.width).Align(lipgloss.Right).PaddingRight(2).Render(progress),
	)
}

func (s *SetupModel) renderBotCol(
	title string,
	color lipgloss.Color,
	fName, fRole, fModel, fTemp, fPrompt setupField,
	ta textarea.Model,
	colW int,
) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(color).Render("── " + title + " ──")

	lbl := func(name string, f setupField) string {
		return s.fieldLabel(name, s.cursor == f, color)
	}

	fields := []string{
		header,
		lbl("İsim", fName),
		s.shorts[shortIdx(fName)].View(),
		lbl("Rol / Persona", fRole),
		s.shorts[shortIdx(fRole)].View(),
		lbl("Model", fModel),
		s.shorts[shortIdx(fModel)].View(),
		lbl("Sıcaklık (0–2)", fTemp),
		s.shorts[shortIdx(fTemp)].View(),
		lbl("System Prompt  (Esc veya Tab ▸ çıkış)", fPrompt),
		ta.View(),
	}

	inner := strings.Join(fields, "\n")

	borderStyle := stylePanel.Width(colW - 2)
	if title == "BOT A" && s.cursor >= fAName && s.cursor <= fAPrompt {
		borderStyle = stylePanelActive.Width(colW - 2)
	}
	if title == "BOT B" && s.cursor >= fBName && s.cursor <= fBPrompt {
		borderStyle = stylePanelActive.Width(colW - 2)
	}

	return borderStyle.Render(inner)
}

func (s *SetupModel) fieldLabel(name string, active bool, color lipgloss.Color) string {
	base := lipgloss.NewStyle().Foreground(color)
	if active {
		return base.Bold(true).Render("▶ " + name + ":")
	}
	return styleMuted.Render("  " + name + ":")
}
