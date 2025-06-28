package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
)

func main() {
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

	prerolls := cfg.GetPrerolls(time.Now())
	for _, preroll := range prerolls {
		fmt.Println(preroll)
	}
}