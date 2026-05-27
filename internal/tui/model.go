package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/padros/chatbot2u/internal/agent"
	"github.com/padros/chatbot2u/internal/config"
	"github.com/padros/chatbot2u/internal/ollama"
)

type appState int

const (
	stateIdle appState = iota
	stateGenerating
	stateWhisper // kullanıcı fısıltı yazıyor
)

type doneMsg struct {
	botIndex int
	reply    string
	err      error
}

type autoTickMsg struct{}

type Model struct {
	cfg    *config.AppConfig
	client *ollama.Client
	botA   *agent.Agent
	botB   *agent.Agent
	vpA    viewport.Model
	vpB    viewport.Model
	width  int
	height int
	state  appState
	autoOn bool
	turn   int // 0 = botA, 1 = botB

	linesA []string
	linesB []string

	statusMsg string
	err       error
	cancelGen context.CancelFunc
	seedMsg   string

	// Fısıltı
	whisperTarget int        // 0=A, 1=B
	whisperInput  textinput.Model
	whispers      [2]string  // her bot için bekleyen fısıltı
}

func New(cfg *config.AppConfig, seedMsg string) *Model {
	client := ollama.NewClient(cfg.Ollama.BaseURL, cfg.Ollama.Timeout)
	botA := agent.New(cfg.BotA.Name, cfg.BotA.Role, cfg.BotA.SystemPrompt, cfg.BotA.Model, cfg.BotA.Temperature)
	botB := agent.New(cfg.BotB.Name, cfg.BotB.Role, cfg.BotB.SystemPrompt, cfg.BotB.Model, cfg.BotB.Temperature)

	wi := textinput.New()
	wi.Placeholder = "bot için gizli yönlendirme yaz..."
	wi.CharLimit = 512
	wi.Width = 60

	return &Model{
		cfg:          cfg,
		client:       client,
		botA:         botA,
		botB:         botB,
		seedMsg:      seedMsg,
		turn:         0,
		statusMsg:    "Hazır — [A]/[B] konuştur, [Space] sıradakini, [W] fısıltı",
		whisperInput: wi,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initViewports()

	case tea.KeyMsg:
		if cmd := m.handleKey(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case doneMsg:
		m.state = stateIdle
		m.cancelGen = nil
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Hata: %s", msg.err)
		} else {
			m.err = nil
			m.rebuildLines(msg.botIndex)
			m.rebuildLines(1 - msg.botIndex)
			bot := m.botByIndex(msg.botIndex)
			next := m.botByIndex(1 - msg.botIndex)
			m.statusMsg = fmt.Sprintf("%s yanıtladı → Sıra: %s", bot.Name, next.Name)
			m.turn = 1 - msg.botIndex
			if m.autoOn {
				cmds = append(cmds, m.autoTickCmd())
			}
		}

	case autoTickMsg:
		if m.autoOn && m.state == stateIdle {
			if cmd := m.speakCmd(m.turn); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Viewport güncellemesi
	var cmd tea.Cmd
	m.vpA, cmd = m.vpA.Update(msg)
	cmds = append(cmds, cmd)
	m.vpB, cmd = m.vpB.Update(msg)
	cmds = append(cmds, cmd)

	// Fısıltı input güncellemesi
	if m.state == stateWhisper {
		m.whisperInput, cmd = m.whisperInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Fısıltı modunda ayrı davran
	if m.state == stateWhisper {
		return m.handleWhisperKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.cancelGen != nil {
			m.cancelGen()
		}
		return tea.Quit

	case "a", "A":
		if m.state == stateIdle {
			return m.speakCmd(0)
		}

	case "b", "B":
		if m.state == stateIdle {
			return m.speakCmd(1)
		}

	case " ":
		if m.state == stateIdle {
			return m.speakCmd(m.turn)
		}

	case "w", "W":
		if m.state == stateIdle {
			m.enterWhisperMode()
		}

	case "t", "T":
		m.autoOn = !m.autoOn
		if m.autoOn {
			m.statusMsg = "Otomatik mod AÇIK — [T] kapat"
			if m.state == stateIdle {
				return m.autoTickCmd()
			}
		} else {
			m.statusMsg = "Otomatik mod KAPALI"
		}

	case "s", "S":
		if m.cancelGen != nil {
			m.cancelGen()
		}
		m.autoOn = false
		m.state = stateIdle
		m.statusMsg = "Durduruldu"

	case "c", "C":
		if m.cancelGen != nil {
			m.cancelGen()
			m.cancelGen = nil
		}
		m.botA.ClearHistory()
		m.botB.ClearHistory()
		m.linesA, m.linesB = nil, nil
		m.whispers = [2]string{}
		m.state = stateIdle
		m.autoOn = false
		m.turn = 0
		m.err = nil
		m.statusMsg = "Geçmiş ve fısıltılar temizlendi"
		m.refreshViewport(0)
		m.refreshViewport(1)
	}
	return nil
}

func (m *Model) handleWhisperKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.exitWhisperMode(false)

	case "enter":
		m.exitWhisperMode(true)

	case "tab":
		// Hedef botu değiştir
		m.whisperTarget = 1 - m.whisperTarget
		name := m.botByIndex(m.whisperTarget).Name
		m.statusMsg = fmt.Sprintf("Fısıltı hedefi → %s", name)
	}
	return nil
}

func (m *Model) enterWhisperMode() {
	m.state = stateWhisper
	m.whisperTarget = m.turn
	m.whisperInput.SetValue("")
	m.whisperInput.Focus()
	target := m.botByIndex(m.whisperTarget).Name
	m.statusMsg = fmt.Sprintf("Fısıltı: %s için yaz, Enter onayla, Esc iptal, Tab hedef değiştir", target)
}

func (m *Model) exitWhisperMode(confirm bool) {
	m.whisperInput.Blur()
	text := strings.TrimSpace(m.whisperInput.Value())
	if confirm && text != "" {
		m.whispers[m.whisperTarget] = text
		name := m.botByIndex(m.whisperTarget).Name
		m.statusMsg = fmt.Sprintf("Fısıltı hazır → %s (sonraki konuşmasında etkili olur)", name)
	} else {
		m.statusMsg = "Fısıltı iptal edildi"
	}
	m.state = stateIdle
}

func (m *Model) speakCmd(botIndex int) tea.Cmd {
	if m.state != stateIdle {
		return nil
	}

	// Karşı botun geçmişinden son mesajı al
	history := m.botByIndex(1 - botIndex).History()
	var prompt string
	if len(history) > 0 {
		prompt = history[len(history)-1].Content
	}
	if prompt == "" {
		prompt = m.seedMsg
	}
	if prompt == "" {
		prompt = "Merhaba, konuşmaya başlayalım."
	}

	// Fısıltıyı kap ve temizle (yarış koşulunu önlemek için şimdi al)
	whisper := m.whispers[botIndex]
	m.whispers[botIndex] = ""

	m.state = stateGenerating
	m.turn = botIndex
	bot := m.botByIndex(botIndex)
	whisperNote := ""
	if whisper != "" {
		whisperNote = fmt.Sprintf(" 〔fısıltı aktif〕")
	}
	m.statusMsg = fmt.Sprintf("%s düşünüyor...%s", bot.Name, whisperNote)
	m.rebuildLines(botIndex) // "düşünüyor" göstergesi için

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelGen = cancel
	finalPrompt := prompt
	finalWhisper := whisper

	return func() tea.Msg {
		reply, err := bot.Respond(ctx, m.client, finalPrompt, finalWhisper, nil)
		return doneMsg{botIndex: botIndex, reply: reply, err: err}
	}
}

func (m *Model) autoTickCmd() tea.Cmd {
	d := time.Duration(m.cfg.AutoDelay) * time.Second
	return tea.Tick(d, func(time.Time) tea.Msg { return autoTickMsg{} })
}

func (m *Model) botByIndex(i int) *agent.Agent {
	if i == 0 {
		return m.botA
	}
	return m.botB
}

func (m *Model) initViewports() {
	pw := (m.width-3)/2 - 4
	ph := m.height - 7 // title + status + buttons + whisper satırı
	if pw < 10 {
		pw = 10
	}
	if ph < 4 {
		ph = 4
	}
	m.vpA = viewport.New(pw, ph-2)
	m.vpB = viewport.New(pw, ph-2)
	m.rebuildLines(0)
	m.rebuildLines(1)
}

func (m *Model) rebuildLines(botIndex int) {
	history := m.botByIndex(botIndex).History()
	var lines []string

	for _, msg := range history {
		if msg.Role == "system" {
			continue
		}
		var nameStyle lipgloss.Style
		var name string
		if msg.Role == "assistant" {
			if botIndex == 0 {
				nameStyle = styleBotAHeader
				name = "▶ " + m.botA.Name
			} else {
				nameStyle = styleBotBHeader
				name = "▶ " + m.botB.Name
			}
		} else {
			if botIndex == 0 {
				nameStyle = styleBotBHeader
				name = "◀ " + m.botB.Name
			} else {
				nameStyle = styleBotAHeader
				name = "◀ " + m.botA.Name
			}
		}
		lines = append(lines, nameStyle.Render(name))
		lines = append(lines, styleBotAMsg.Render(msg.Content))
		lines = append(lines, "")
	}

	if m.state == stateGenerating && m.turn == botIndex {
		lines = append(lines, styleThinking.Render("⟳ düşünüyor..."))
	}

	if botIndex == 0 {
		m.linesA = lines
	} else {
		m.linesB = lines
	}
	m.refreshViewport(botIndex)
}

func (m *Model) refreshViewport(botIndex int) {
	var lines []string
	if botIndex == 0 {
		lines = m.linesA
	} else {
		lines = m.linesB
	}
	content := strings.Join(lines, "\n")
	if botIndex == 0 {
		m.vpA.SetContent(content)
		m.vpA.GotoBottom()
	} else {
		m.vpB.SetContent(content)
		m.vpB.GotoBottom()
	}
}

// ── Görünüm ──────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	if m.width == 0 {
		return "Yükleniyor..."
	}

	titleText := fmt.Sprintf("◈ chatboT2u ◈   %s (%s)  ↔  %s (%s)",
		m.botA.Name, m.botA.Role, m.botB.Name, m.botB.Role)
	titleBar := styleTitleBar.Width(m.width).Render(titleText)

	panelW := (m.width - 3) / 2
	panelH := m.height - 7
	if panelH < 4 {
		panelH = 4
	}

	psA, psB := stylePanel, stylePanel
	if m.state == stateGenerating {
		if m.turn == 0 {
			psA = stylePanelActive
		} else {
			psB = stylePanelActive
		}
	}

	headerA := styleBotAHeader.Render(fmt.Sprintf("Bot A: %s  [%s]  T=%.1f%s",
		m.botA.Name, m.botA.Model, m.botA.Temperature, m.whisperBadge(0)))
	headerB := styleBotBHeader.Render(fmt.Sprintf("Bot B: %s  [%s]  T=%.1f%s",
		m.botB.Name, m.botB.Model, m.botB.Temperature, m.whisperBadge(1)))

	innerW := panelW - 4
	if innerW < 4 {
		innerW = 4
	}
	m.vpA.Width = innerW
	m.vpB.Width = innerW

	contentA := lipgloss.JoinVertical(lipgloss.Left, headerA, m.vpA.View())
	contentB := lipgloss.JoinVertical(lipgloss.Left, headerB, m.vpB.View())

	panelA := psA.Width(panelW - 2).Height(panelH).Render(contentA)
	panelB := psB.Width(panelW - 2).Height(panelH).Render(contentB)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, panelA, " ", panelB)

	// Fısıltı satırı — her zaman ayrılmış alan, mod dışında boş
	whisperBar := m.renderWhisperBar()

	// Durum çubuğu
	spin := "○"
	if m.state == stateGenerating {
		spin = "●"
	}
	autoLabel := styleMuted.Render("oto:kapalı")
	if m.autoOn {
		autoLabel = styleButtonActive.Render(" oto:açık ")
	}
	statusLine := fmt.Sprintf(" %s %s  │ %s │ sıra: %s │ A=%d B=%d",
		spin, m.statusMsg, autoLabel,
		m.botByIndex(m.turn).Name,
		len(m.botA.History()), len(m.botB.History()))
	if m.err != nil {
		statusLine = styleError.Render("✗ "+m.err.Error()) + "  " + statusLine
	}
	statusBar := styleStatusBar.Width(m.width).Render(statusLine)

	// Buton satırı
	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		styleButtonBotA.Render("[A] Bot A"),
		styleButtonBotB.Render("[B] Bot B"),
		styleButton.Render("[Space] Sıradaki"),
		styleButton.Render("[W] Fısıltı"),
		styleButton.Render("[T] Oto"),
		styleButton.Render("[S] Dur"),
		styleButton.Render("[C] Temizle"),
		styleButton.Render("[Q] Çıkış"),
	)
	buttonBar := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(buttons)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		panels,
		whisperBar,
		statusBar,
		buttonBar,
	)
}

func (m *Model) whisperBadge(botIndex int) string {
	if m.whispers[botIndex] != "" {
		return styleThinking.Render(" 〔fısıltı〕")
	}
	return ""
}

func (m *Model) renderWhisperBar() string {
	if m.state != stateWhisper {
		// Fısıltı modu kapalı — pasif ipucu
		hint := styleMuted.Render("  [W] ile bota gizli fısıltı gönder")
		return lipgloss.NewStyle().Width(m.width).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorBorder).
			PaddingLeft(1).
			Render(hint)
	}

	targetBot := m.botByIndex(m.whisperTarget)
	targetStyle := styleBotAHeader
	if m.whisperTarget == 1 {
		targetStyle = styleBotBHeader
	}

	label := targetStyle.Render(fmt.Sprintf("〔Fısıltı → %s〕", targetBot.Name))
	hint := styleMuted.Render("  Enter onayla  │  Tab hedef değiştir  │  Esc iptal")

	content := lipgloss.JoinHorizontal(lipgloss.Center,
		label, "  ",
		m.whisperInput.View(),
		"  ",
		hint,
	)

	return lipgloss.NewStyle().Width(m.width).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		PaddingLeft(1).
		Render(content)
}
