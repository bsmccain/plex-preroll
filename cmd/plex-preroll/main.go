package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("plex-preroll", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dryRun := flags.Bool("dry-run", false, "validate and list prerolls without updating Plex")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *showVersion {
		fmt.Fprintf(stdout, "plex-preroll %s\n", version)
		return nil
	}
	logger := log.New(stderr, "", log.LstdFlags)

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
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect .env file %s: %w", path, err)
		}
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("load .env file %s: %w", path, err)
		}
		logger.Printf("Loaded .env from %s", path)
		loadedEnv = true
		break
	}
	if !loadedEnv {
		logger.Println("No .env file found in preferred locations; relying on system env vars if set.")
	}

	serverURL := os.Getenv("PLEX_SERVER_URL")
	if serverURL == "" {
		logger.Println("PLEX_SERVER_URL is not set, using localhost:32400")
		serverURL = "http://localhost:32400"
	}

	token := os.Getenv("PLEX_TOKEN")
	if token == "" {
		return fmt.Errorf("PLEX_TOKEN is not set")
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
			if _, err := os.Stat(potentialPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return fmt.Errorf("inspect config file %s: %w", potentialPath, err)
			}
			cfgPath = potentialPath
			logger.Printf("Using config file: %s", cfgPath)
			break
		}
		if cfgPath == "" {
			return fmt.Errorf("no config file found in standard locations")
		}
	}

	cfgPath, err = internal.ExpandHomePath(cfgPath)
	if err != nil {
		return err
	}
	cfg, err := internal.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load config from %s: %w", cfgPath, err)
	}

	today := time.Now()
	prerolls, err := internal.BuildPrerollFileList(cfg.GetPrerolls(today))
	if err != nil {
		return fmt.Errorf("build preroll file list: %w", err)
	}

	logger.Printf("Preroll list for %s:", today.Format("2006-01-02"))
	for _, f := range prerolls {
		logger.Printf("  - %s", f)
	}

	if *dryRun {
		logger.Println("Dry run, skipping update")
		return nil
	}

	err = internal.UpdatePreroll(serverURL, token, prerolls)
	if err != nil {
		return fmt.Errorf("update preroll: %w", err)
	}

	logger.Println("Preroll updated successfully")
	return nil
}
