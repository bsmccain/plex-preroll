package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
)

func main() {
	home, err := os.UserHomeDir()
	paths := []string{".env", filepath.Join(home, ".config", "plex-preroll", ".env")}
	loaded := false
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from %s", path)
			loaded = true
			break
		}
	}
	if !loaded {
		log.Println("No .env file found in . or ~/.config/plex-preroll; relying on system env vars.")
	}

	dryRun := flag.Bool("dry-run", false, "dry run")
	flag.Parse()

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
		cfgPath = filepath.Join(homeDir, ".config", "plex-preroll", "config.yaml")
	}
	cfg, err := internal.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	today := time.Now()
	prerolls, err := internal.BuildPrerollFileList(cfg.GetPrerolls(today))
	if err != nil {
		log.Fatalf("failed to build preroll file list: %v", err)
	}
	log.Printf("Prerolls for %s: %v", today.Format("2006-01-02"), prerolls)

	if *dryRun {
		log.Println("Dry run, skipping update")
		return
	}

	err = internal.UpdatePreroll(serverURL, token, prerolls)
	if err != nil {
		log.Fatalf("failed to update preroll: %v", err)
	}

	log.Println("Preroll updated successfully")
}