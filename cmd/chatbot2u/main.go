package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/padros/chatbot2u/internal/config"
	"github.com/padros/chatbot2u/internal/ollama"
	"github.com/padros/chatbot2u/internal/tui"
)

func main() {
	cfgPath := flag.String("config", "configs/config.json", "Konfigürasyon dosyası yolu")
	skipSetup := flag.Bool("skip-setup", false, "Kurulum ekranını atla, doğrudan başlat")
	seed := flag.String("seed", "", "Başlangıç mesajı (kurulum ekranındaki alanı doldurur)")
	flag.Parse()

	// Konfigürasyon yükle
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		// Varsayılan konfigürasyonu kullan, örnek dosyayı kaydet
		cfg = config.Default()
		if saveErr := config.Save(cfg, *cfgPath); saveErr == nil {
			fmt.Fprintf(os.Stderr, "Bilgi: %s oluşturuldu (varsayılan konfigürasyon)\n", *cfgPath)
		}
	}

	// Ollama erişilebilirlik kontrolü
	client := ollama.NewClient(cfg.Ollama.BaseURL, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: Ollama'ya bağlanılamıyor: %v\n", err)
		fmt.Fprintf(os.Stderr, "Ollama'nın çalıştığından emin olun: ollama serve\n")
		os.Exit(1)
	}
	fmt.Println("Ollama bağlantısı başarılı.")

	seedMsg := *seed
	if !*skipSetup {
		// Senaryo galerisi
		gallery := tui.NewGallery()
		p := tea.NewProgram(gallery, tea.WithAltScreen())
		fm, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Galeri hatası: %v\n", err)
			os.Exit(1)
		}
		if gm, ok := fm.(*tui.GalleryModel); ok && gm.Done() {
			sc := gm.Selected()
			cfg.BotA = sc.BotA
			cfg.BotB = sc.BotB
			if seedMsg == "" {
				seedMsg = sc.SeedMsg
			}
		}

		// Kimlik kurulum ekranı
		setupModel := tui.NewSetup(cfg)
		p = tea.NewProgram(setupModel, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Kurulum hatası: %v\n", err)
			os.Exit(1)
		}
		sm, ok := finalModel.(*tui.SetupModel)
		if !ok || !sm.Done() {
			fmt.Println("Kurulum iptal edildi.")
			os.Exit(0)
		}
		if seedMsg == "" {
			seedMsg = sm.SeedMsg()
		}
	}

	// Ana TUI
	mainModel := tui.New(cfg, seedMsg)
	p := tea.NewProgram(mainModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI hatası: %v\n", err)
		os.Exit(1)
	}
}
