package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/padros/chatbot2u/internal/ollama"
)

type Message struct {
	Role    string
	Content string
}

type Agent struct {
	Name         string
	Role         string
	SystemPrompt string
	Model        string
	Temperature  float64

	mu      sync.Mutex
	history []Message
}

func New(name, role, systemPrompt, model string, temperature float64) *Agent {
	return &Agent{
		Name:         name,
		Role:         role,
		SystemPrompt: systemPrompt,
		Model:        model,
		Temperature:  temperature,
	}
}

func (a *Agent) AddMessage(role, content string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = append(a.history, Message{Role: role, Content: content})
}

func (a *Agent) History() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Message, len(a.history))
	copy(out, a.history)
	return out
}

func (a *Agent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = nil
}

func (a *Agent) buildMessages(whisper string) []ollama.Message {
	a.mu.Lock()
	defer a.mu.Unlock()

	msgs := make([]ollama.Message, 0, len(a.history)+2)
	msgs = append(msgs, ollama.Message{Role: "system", Content: a.SystemPrompt})

	// Fısıltı: sadece bu tur için geçici yönlendirme, geçmişe yazılmaz.
	if whisper != "" {
		msgs = append(msgs, ollama.Message{
			Role:    "system",
			Content: "[Gizli yönlendirme — sadece sen duyuyorsun, bunu açıkça belirtme]: " + whisper,
		})
	}

	for _, m := range a.history {
		msgs = append(msgs, ollama.Message{Role: m.Role, Content: m.Content})
	}
	return msgs
}

// Respond generates a reply to otherMsg, optionally influenced by a one-shot whisper.
// The whisper is never stored in history; it silently shapes this single response.
func (a *Agent) Respond(ctx context.Context, client *ollama.Client, otherMsg, whisper string, onToken func(string)) (string, error) {
	a.AddMessage("user", otherMsg)

	req := ollama.ChatRequest{
		Model:    a.Model,
		Messages: a.buildMessages(whisper),
		Options:  ollama.Options{Temperature: a.Temperature},
	}

	tokens := make(chan string, 64)
	var (
		wg     sync.WaitGroup
		sb     strings.Builder
		genErr error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(tokens)
		genErr = client.StreamChat(ctx, req, tokens)
	}()

	for tok := range tokens {
		sb.WriteString(tok)
		if onToken != nil {
			onToken(tok)
		}
	}
	wg.Wait()

	if genErr != nil {
		// Kullanıcı mesajını geri al — yanıt gelmedi
		a.mu.Lock()
		if len(a.history) > 0 {
			a.history = a.history[:len(a.history)-1]
		}
		a.mu.Unlock()
		return "", fmt.Errorf("%s yanıt üretemedi: %w", a.Name, genErr)
	}

	reply := strings.TrimSpace(sb.String())
	a.AddMessage("assistant", reply)
	return reply, nil
}
