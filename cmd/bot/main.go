package main

import (
	"log"

	"github.com/Omelyko/crypto-opportunities-bot/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.App.Environment == "development" {
		log.Printf("Config loaded:\n%s", cfg.SafeString())
	}
}
