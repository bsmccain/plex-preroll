package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
)

var version = "dev"

func main() {
	dryRun := flag.Bool("dry-run", false, "dry run")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("plex-preroll %s\n", version)
		return
	}

	// Determine potential .env paths
	envPaths := []string{".env"}
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		envPaths = append(envPaths, filepath.Join(userConfigDir, "plex-preroll", ".env"))
	}
	userHomeDir, err := os.UserHomeDir()
	if err == nil {
		envPaths = append(envPaths, filepath.Join(userHomeDir, ".config", "plex-preroll", ".env"))
	}

	loadedEnv := false
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err == nil {
				log.Printf("Loaded .env from %s", path)
				loadedEnv = true
				break
			}
		}
	}
	if !loadedEnv {
		log.Println("No .env file found in preferred locations; relying on system env vars if set.")
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
		defaultCfgPaths := []string{}
		if userConfigDir != "" {
			defaultCfgPaths = append(defaultCfgPaths, filepath.Join(userConfigDir, "plex-preroll", "config.yaml"))
		}
		if userHomeDir != "" {
			defaultCfgPaths = append(defaultCfgPaths, filepath.Join(userHomeDir, ".config", "plex-preroll", "config.yaml"))
		}

		for _, potentialPath := range defaultCfgPaths {
			if _, err := os.Stat(potentialPath); err == nil {
				cfgPath = potentialPath
				log.Printf("Using config file: %s", cfgPath)
				break
			}
		}
		if cfgPath == "" {
			log.Fatalf("No config file found in standard locations.")
		}
	}

	cfg, err := internal.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", cfgPath, err)
	}

	today := time.Now()
	prerolls, err := internal.BuildPrerollFileList(cfg.GetPrerolls(today))
	if err != nil {
		log.Fatalf("failed to build preroll file list: %v", err)
	}

	log.Printf("Preroll list for %s:", today.Format("2006-01-02"))
	for _, f := range prerolls {
		log.Printf("  - %s", f)
	}

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
