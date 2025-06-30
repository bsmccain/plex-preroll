package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the structure of the config.yaml file
type Config struct {
	Seasons map[string]Season `yaml:"seasons"`
	Default []string          `yaml:"default"`
}

// Season represents a single season in the config
type Season struct {
	From  string   `yaml:"from"`
	To    string   `yaml:"to"`
	Files []string `yaml:"files"`
}

// Function to read and validate date input
func getDateInput(prompt string, reader *bufio.Reader) string {
	dateRegex := regexp.MustCompile(`^\d{2}-\d{2}$`)
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if dateRegex.MatchString(input) {
			// Further validation could be added here (e.g., valid month/day ranges)
			return input
		}
		fmt.Println("Invalid date format. Please use MM-DD (e.g., 10-31).")
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Plex Preroll Setup")
	fmt.Println("------------------")

	// .env file setup
	fmt.Println("\nLet's set up your .env file.")

	fmt.Print("Enter your PLEX_TOKEN: ")
	plexToken, _ := reader.ReadString('\n')
	plexToken = strings.TrimSpace(plexToken)

	fmt.Print("Enter your PLEX_SERVER_URL (default: http://localhost:32400): ")
	plexServerURL, _ := reader.ReadString('\n')
	plexServerURL = strings.TrimSpace(plexServerURL)
	if plexServerURL == "" {
		plexServerURL = "http://localhost:32400"
	}

	userConfigDir, err := os.UserConfigDir()
	defaultConfigPath := ""
	if err == nil {
		defaultConfigPath = filepath.Join(userConfigDir, "plex-preroll", "config.yaml")
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err == nil {
			defaultConfigPath = filepath.Join(userHomeDir, ".config", "plex-preroll", "config.yaml")
		} else {
			defaultConfigPath = "config.yaml" // Fallback if all else fails
		}
	}

	fmt.Printf("Enter path for PLEX_PREROLL_CFG (config.yaml) (default: %s): ", defaultConfigPath)
	plexPrerollCfg, _ := reader.ReadString('\n')
	plexPrerollCfg = strings.TrimSpace(plexPrerollCfg)
	if plexPrerollCfg == "" {
		plexPrerollCfg = defaultConfigPath
	}

	envContent := fmt.Sprintf("PLEX_TOKEN=%s\nPLEX_SERVER_URL=%s\nPLEX_PREROLL_CFG=%s\n", plexToken, plexServerURL, plexPrerollCfg)

	defaultEnvPath := ""
	if err == nil { // userConfigDir is valid
		defaultEnvPath = filepath.Join(userConfigDir, "plex-preroll", ".env")
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err == nil {
			defaultEnvPath = filepath.Join(userHomeDir, ".config", "plex-preroll", ".env")
		} else {
			defaultEnvPath = ".env"
		}
	}

	fmt.Printf("Enter path to save .env file (default: %s): ", defaultEnvPath)
	envPath, _ := reader.ReadString('\n')
	envPath = strings.TrimSpace(envPath)
	if envPath == "" {
		envPath = defaultEnvPath
	}

	err = os.MkdirAll(filepath.Dir(envPath), 0755)
	if err != nil {
		fmt.Printf("Error creating directory for .env file: %v\n", err)
		return
	}
	err = os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		fmt.Printf("Error writing .env file: %v\n", err)
		return
	}
	fmt.Printf(".env file created at %s\n", envPath)

	// config.yaml setup
	fmt.Println("\nNow let's set up your config.yaml file.")
	config := Config{
		Seasons: make(map[string]Season),
		Default: []string{},
	}

	for {
		fmt.Print("Add a season? (y/n): ")
		addSeasonInput, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(addSeasonInput)) != "y" {
			break
		}

		fmt.Print("Enter season name (e.g., halloween, christmas): ")
		seasonName, _ := reader.ReadString('\n')
		seasonName = strings.TrimSpace(seasonName)
		if seasonName == "" {
			fmt.Println("Season name cannot be empty. Please try again.")
			continue
		}

		seasonFrom := getDateInput("Enter season start date (MM-DD): ", reader)
		seasonTo := getDateInput("Enter season end date (MM-DD): ", reader)

		var files []string
		for {
			fmt.Print("Enter a preroll file or directory path for this season (or leave blank to finish): ")
			filePath, _ := reader.ReadString('\n')
			filePath = strings.TrimSpace(filePath)
			if filePath == "" {
				break
			}
			files = append(files, filePath)
		}
		config.Seasons[seasonName] = Season{From: seasonFrom, To: seasonTo, Files: files}
	}

	fmt.Println("\nLet's set up default prerolls (used when no season matches).")
	for {
		fmt.Print("Enter a default preroll file or directory path (or leave blank to finish): ")
		filePath, _ := reader.ReadString('\n')
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			break
		}
		config.Default = append(config.Default, filePath)
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Printf("Error marshalling YAML: %v\n", err)
		return
	}

	err = os.MkdirAll(filepath.Dir(plexPrerollCfg), 0755)
	if err != nil {
		fmt.Printf("Error creating directory for config.yaml: %v\n", err)
		return
	}
	err = os.WriteFile(plexPrerollCfg, yamlData, 0644)
	if err != nil {
		fmt.Printf("Error writing config.yaml: %v\n", err)
		return
	}
	fmt.Printf("config.yaml file created at %s\n", plexPrerollCfg)
	fmt.Println("\nSetup complete!")
}
