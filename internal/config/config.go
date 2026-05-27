package config

import (
	"encoding/json"
	"os"
)

type BotConfig struct {
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
}

type OllamaConfig struct {
	BaseURL string `json:"base_url"`
	Timeout int    `json:"timeout_seconds"`
}

type AppConfig struct {
	BotA      BotConfig    `json:"bot_a"`
	BotB      BotConfig    `json:"bot_b"`
	Ollama    OllamaConfig `json:"ollama"`
	AutoDelay int          `json:"auto_delay_seconds"`
}

func Default() *AppConfig {
	return &AppConfig{
		BotA: BotConfig{
			Name:         "Alfa",
			Role:         "Felsefeci",
			SystemPrompt: "Sen derin bir felsefeci yapay zekasın. Sorulara felsefi perspektiften yaklaşırsın, Socrates ve Platon'dan ilham alırsın. Her yanıtında soru sorarak diyalogu ilerletirsin.",
			Model:        "llama3.2",
			Temperature:  0.8,
		},
		BotB: BotConfig{
			Name:         "Beta",
			Role:         "Bilim İnsanı",
			SystemPrompt: "Sen ampirik bir bilim insanı yapay zekasın. Fikirlere bilimsel ve kanıta dayalı yaklaşırsın. Felsefi sorulara bile ölçülebilir, test edilebilir bir perspektiften cevap verirsin.",
			Model:        "llama3.2",
			Temperature:  0.7,
		},
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
			Timeout: 120,
		},
		AutoDelay: 3,
	}
}

func Load(path string) (*AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := Default()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(cfg *AppConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
