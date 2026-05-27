# chatboT2u

İki yerel AI ajanının birbirleriyle konuştuğu, terminal tabanlı (TUI) bir uygulama.  
Ollama üzerindeki herhangi bir modeli kullanır. Tamamen yerel çalışır.

## Gereksinimler

- Go 1.21+
- [Ollama](https://ollama.com) çalışıyor olmalı (`ollama serve`)
- En az bir model indirilmiş olmalı (`ollama pull llama3.2`)

## Kurulum

```bash
git clone ...
cd chatboT2u
go build -o chatbot2u ./cmd/chatbot2u/
```

## Çalıştırma

```bash
# Kurulum ekranı ile başlat (varsayılan)
./chatbot2u

# Konfigürasyon dosyası belirterek
./chatbot2u -config configs/config.json

# Kurulum ekranını atla, doğrudan başlat
./chatbot2u -skip-setup

# Başlangıç mesajını flag ile ver
./chatbot2u -skip-setup -seed "Özgür irade var mı?"
```

## Klavye Kısayolları

| Tuş | Eylem |
|-----|-------|
| `A` | Bot A konuşsun |
| `B` | Bot B konuşsun |
| `Space` | Sıradaki bot konuşsun |
| `T` | Otomatik mod aç/kapat |
| `S` | Üretimi durdur |
| `C` | Konuşma geçmişini temizle |
| `Q` / `Ctrl+C` | Çıkış |

## Konfigürasyon (`configs/config.json`)

```json
{
  "bot_a": {
    "name": "Alfa",
    "role": "Felsefeci",
    "system_prompt": "...",
    "model": "llama3.2",
    "temperature": 0.8
  },
  "bot_b": {
    "name": "Beta",
    "role": "Bilim İnsanı",
    "system_prompt": "...",
    "model": "llama3.2",
    "temperature": 0.7
  },
  "ollama": {
    "base_url": "http://localhost:11434",
    "timeout_seconds": 120
  },
  "auto_delay_seconds": 3
}
```

## Proje Yapısı

```
chatboT2u/
├── cmd/chatbot2u/main.go      # Giriş noktası
├── internal/
│   ├── agent/agent.go         # Bot mantığı, geçmiş yönetimi
│   ├── ollama/client.go       # Ollama HTTP client (streaming)
│   ├── config/config.go       # Konfigürasyon yükleme/kaydetme
│   └── tui/
│       ├── model.go           # Ana TUI modeli (bubbletea)
│       ├── setup.go           # Kimlik kurulum ekranı
│       └── styles.go          # Lipgloss renk/stil tanımları
├── configs/config.json        # Örnek konfigürasyon
└── go.mod
```

## Ollama Model Önerileri

```bash
ollama pull llama3.2        # Genel amaç, hızlı
ollama pull qwen2.5         # Çok dilli, güçlü
ollama pull mistral         # Dengeli performans
ollama pull phi3            # Hafif, düşük bellek
```
