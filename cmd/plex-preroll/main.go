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
	envPaths := []string{".env"} // Current directory first
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		envPaths = append(envPaths, filepath.Join(userConfigDir, "plex-preroll", ".env"))
	}
	userHomeDir, err := os.UserHomeDir()
	if err == nil {
		envPaths = append(envPaths, filepath.Join(userHomeDir, ".config", "plex-preroll", ".env")) // Legacy path
	}

	loadedEnv := false
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil { // Check if file exists before trying to load
			if err := godotenv.Load(path); err == nil {
				log.Printf("Loaded .env from %s", path)
				loadedEnv = true
				break
			} else {
				log.Printf("Warning: Error loading .env file from %s: %v", path, err)
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
		// Determine default config.yaml path by checking preferred locations
		defaultCfgPaths := []string{}
		if userConfigDir != "" { // Check if os.UserConfigDir() was successful
			defaultCfgPaths = append(defaultCfgPaths, filepath.Join(userConfigDir, "plex-preroll", "config.yaml"))
		}
		if userHomeDir != "" { // Check if os.UserHomeDir() was successful
			defaultCfgPaths = append(defaultCfgPaths, filepath.Join(userHomeDir, ".config", "plex-preroll", "config.yaml")) // Legacy
		}

		foundCfgPath := false
		for _, potentialPath := range defaultCfgPaths {
			if _, err := os.Stat(potentialPath); err == nil {
				cfgPath = potentialPath
				log.Printf("Using config file: %s", cfgPath)
				foundCfgPath = true
				break
			}
		}
		if !foundCfgPath {
			// If no config file found in preferred locations, fallback to the legacy path as the one to "fail" on.
			// This maintains the old behavior of erroring if the legacy path doesn't exist and no other is found.
			if userHomeDir != "" {
				cfgPath = filepath.Join(userHomeDir, ".config", "plex-preroll", "config.yaml")
			} else if userConfigDir != "" {
				// Should not happen if UserHomeDir failed, but as a last resort.
				cfgPath = filepath.Join(userConfigDir, "plex-preroll", "config.yaml")
			} else {
				// Extremely unlikely scenario, but necessary for variable initialization.
				log.Fatalf("Cannot determine home or user config directory to find default config.yaml.")
			}
			// Let LoadConfig handle the error if this default path also doesn't exist.
			log.Printf("No config file found in standard locations. Attempting to use default/legacy path: %s", cfgPath)
		}
	} else {
		log.Printf("Using config file from PLEX_PREROLL_CFG: %s", cfgPath)
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

	// Deduplicate the list of prerolls.
	// BuildPrerollFileList already sorts the list, which helps make deduplication straightforward
	// and keeps the order consistent if that matters.
	seen := make(map[string]bool)
	deduplicatedPrerolls := []string{}
	for _, preroll := range prerolls {
		if _, found := seen[preroll]; !found {
			seen[preroll] = true
			deduplicatedPrerolls = append(deduplicatedPrerolls, preroll)
		}
	}

	log.Printf("Final preroll list for %s after deduplication:", today.Format("2006-01-02"))
	if len(deduplicatedPrerolls) == 0 {
		log.Println("  (No prerolls active for today)")
	} else {
		for _, f := range deduplicatedPrerolls {
			log.Printf("  - %s", f)
		}
	}


	if *dryRun {
		log.Println("Dry run, skipping update")
		return
	}

	err = internal.UpdatePreroll(serverURL, token, deduplicatedPrerolls)
	if err != nil {
		log.Fatalf("failed to update preroll: %v", err)
	}
		return
	}

	// This redundant UpdatePreroll call and return was due to a merge error. Removing it.
	// err = internal.UpdatePreroll(serverURL, token, prerolls)
	// if err != nil {
	// 	log.Fatalf("failed to update preroll: %v", err)
	// }

	log.Println("Preroll updated successfully")
}
