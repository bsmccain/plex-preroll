package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	serverURL := os.Getenv("PLEX_SERVER_URL")
	if serverURL == "" {
		log.Println("PLEX_SERVER_URL is not set, using localhost:32400")
		serverURL = "http://localhost:32400"
	}

	token := os.Getenv("PLEX_TOKEN")
	if token == "" {
		log.Fatalf("PLEX_TOKEN is not set")
	}

	cfgPath := os.Getenv("PLEX_PREROLL_CFG")
	if cfgPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %v", err)
		}
		cfgPath = filepath.Join(homeDir, ".plex-preroll", "config.yaml")
	}
	cfg, err := internal.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	today := time.Now()
	prerolls := cfg.GetPrerolls(today)
	log.Printf("Updating prerolls for %s: %v", today.Format("2006-01-02"), prerolls)
	err = internal.UpdatePreroll(serverURL, token, prerolls)
	if err != nil {
		log.Fatalf("failed to update preroll: %v", err)
	}

	log.Println("Preroll updated successfully")

}