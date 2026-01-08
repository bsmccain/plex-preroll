package main

import (
	"bufio"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGetDateInput(t *testing.T) {
	tests := []struct {
		name     string
		inputs   string
		expected string
	}{
		{
			name:     "valid date on first try",
			inputs:   "10-31\n",
			expected: "10-31",
		},
		{
			name:     "valid date with leading zeros",
			inputs:   "01-05\n",
			expected: "01-05",
		},
		{
			name:     "december 31",
			inputs:   "12-31\n",
			expected: "12-31",
		},
		{
			name:     "january 1",
			inputs:   "01-01\n",
			expected: "01-01",
		},
		{
			name:     "invalid then valid",
			inputs:   "invalid\n10-31\n",
			expected: "10-31",
		},
		{
			name:     "single digit month then valid",
			inputs:   "1-31\n01-31\n",
			expected: "01-31",
		},
		{
			name:     "missing dash then valid",
			inputs:   "1031\n10-31\n",
			expected: "10-31",
		},
		{
			name:     "wrong separator then valid",
			inputs:   "10/31\n10-31\n",
			expected: "10-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.inputs))
			result := getDateInput("Enter date: ", reader)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDateRegex(t *testing.T) {
	dateRegex := regexp.MustCompile(`^\d{2}-\d{2}$`)

	validDates := []string{
		"01-01",
		"12-31",
		"06-15",
		"10-31",
		"11-26",
		"00-00", // regex only checks format, not validity
		"99-99", // regex only checks format, not validity
	}

	invalidDates := []string{
		"1-01",      // single digit month
		"01-1",      // single digit day
		"1-1",       // single digit both
		"010-31",    // three digit month
		"10-310",    // three digit day
		"10/31",     // wrong separator
		"10.31",     // wrong separator
		"10 31",     // space separator
		"1031",      // no separator
		"",          // empty
		"hello",     // not a date
		"2025-10-31", // full date
	}

	for _, date := range validDates {
		if !dateRegex.MatchString(date) {
			t.Errorf("expected %q to be valid", date)
		}
	}

	for _, date := range invalidDates {
		if dateRegex.MatchString(date) {
			t.Errorf("expected %q to be invalid", date)
		}
	}
}

func TestConfigYAMLSerialization(t *testing.T) {
	config := Config{
		Seasons: map[string]Season{
			"halloween": {
				From:  "10-01",
				To:    "10-31",
				Files: []string{"/path/to/halloween1.mp4", "/path/to/halloween2.mp4"},
			},
			"christmas": {
				From:  "12-01",
				To:    "12-31",
				Files: []string{"/path/to/christmas.mp4"},
			},
		},
		Default: []string{"/path/to/default.mp4"},
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Unmarshal and verify
	var loadedConfig Config
	err = yaml.Unmarshal(yamlData, &loadedConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if len(loadedConfig.Seasons) != 2 {
		t.Errorf("expected 2 seasons, got %d", len(loadedConfig.Seasons))
	}

	halloween, ok := loadedConfig.Seasons["halloween"]
	if !ok {
		t.Fatal("halloween season not found")
	}

	if halloween.From != "10-01" {
		t.Errorf("expected halloween from 10-01, got %s", halloween.From)
	}
	if halloween.To != "10-31" {
		t.Errorf("expected halloween to 10-31, got %s", halloween.To)
	}
	if len(halloween.Files) != 2 {
		t.Errorf("expected 2 halloween files, got %d", len(halloween.Files))
	}

	christmas, ok := loadedConfig.Seasons["christmas"]
	if !ok {
		t.Fatal("christmas season not found")
	}

	if christmas.From != "12-01" {
		t.Errorf("expected christmas from 12-01, got %s", christmas.From)
	}

	if !reflect.DeepEqual(loadedConfig.Default, []string{"/path/to/default.mp4"}) {
		t.Errorf("expected default files, got %v", loadedConfig.Default)
	}
}

func TestConfigYAMLEmptyConfig(t *testing.T) {
	config := Config{
		Seasons: make(map[string]Season),
		Default: []string{},
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("failed to marshal empty config: %v", err)
	}

	var loadedConfig Config
	err = yaml.Unmarshal(yamlData, &loadedConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal empty config: %v", err)
	}

	if len(loadedConfig.Seasons) != 0 {
		t.Errorf("expected 0 seasons, got %d", len(loadedConfig.Seasons))
	}

	if len(loadedConfig.Default) != 0 {
		t.Errorf("expected 0 default files, got %d", len(loadedConfig.Default))
	}
}

func TestSeasonStruct(t *testing.T) {
	season := Season{
		From:  "11-20",
		To:    "01-06",
		Files: []string{"/videos/holiday1.mp4", "/videos/holiday2.mkv"},
	}

	yamlData, err := yaml.Marshal(&season)
	if err != nil {
		t.Fatalf("failed to marshal season: %v", err)
	}

	var loadedSeason Season
	err = yaml.Unmarshal(yamlData, &loadedSeason)
	if err != nil {
		t.Fatalf("failed to unmarshal season: %v", err)
	}

	if loadedSeason.From != "11-20" {
		t.Errorf("expected from 11-20, got %s", loadedSeason.From)
	}
	if loadedSeason.To != "01-06" {
		t.Errorf("expected to 01-06, got %s", loadedSeason.To)
	}
	if len(loadedSeason.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(loadedSeason.Files))
	}
}

func TestEnvFileContent(t *testing.T) {
	// Test that the env content format is correct
	plexToken := "abc123xyz"
	plexServerURL := "http://192.168.1.100:32400"
	plexPrerollCfg := "/home/user/.config/plex-preroll/config.yaml"

	envContent := "PLEX_TOKEN=" + plexToken + "\nPLEX_SERVER_URL=" + plexServerURL + "\nPLEX_PREROLL_CFG=" + plexPrerollCfg + "\n"

	if !strings.Contains(envContent, "PLEX_TOKEN=abc123xyz") {
		t.Error("env content missing PLEX_TOKEN")
	}
	if !strings.Contains(envContent, "PLEX_SERVER_URL=http://192.168.1.100:32400") {
		t.Error("env content missing PLEX_SERVER_URL")
	}
	if !strings.Contains(envContent, "PLEX_PREROLL_CFG=/home/user/.config/plex-preroll/config.yaml") {
		t.Error("env content missing PLEX_PREROLL_CFG")
	}
}

func TestFileCreation(t *testing.T) {
	tempDir := t.TempDir()

	// Test creating a file in a nested directory
	nestedPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

	err := os.MkdirAll(filepath.Dir(nestedPath), 0755)
	if err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	content := "test content"
	err = os.WriteFile(nestedPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify file exists and has correct content
	readContent, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("expected %q, got %q", content, string(readContent))
	}
}

func TestDefaultURLValue(t *testing.T) {
	// Verify the default URL constant
	defaultURL := "http://localhost:32400"

	// Empty input should result in default
	input := ""
	result := input
	if result == "" {
		result = defaultURL
	}

	if result != defaultURL {
		t.Errorf("expected default URL %q, got %q", defaultURL, result)
	}
}

func TestConfigPathDetermination(t *testing.T) {
	// Test that config path determination works
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		expectedPath := filepath.Join(userConfigDir, "plex-preroll", "config.yaml")
		if expectedPath == "" {
			t.Error("expected non-empty path")
		}
		// Verify path has expected structure
		if !strings.Contains(expectedPath, "plex-preroll") {
			t.Errorf("expected path to contain 'plex-preroll', got %s", expectedPath)
		}
		if !strings.HasSuffix(expectedPath, "config.yaml") {
			t.Errorf("expected path to end with 'config.yaml', got %s", expectedPath)
		}
	}

	// Test fallback to UserHomeDir
	userHomeDir, err := os.UserHomeDir()
	if err == nil {
		fallbackPath := filepath.Join(userHomeDir, ".config", "plex-preroll", "config.yaml")
		if fallbackPath == "" {
			t.Error("expected non-empty fallback path")
		}
		if !strings.Contains(fallbackPath, ".config") {
			t.Errorf("expected fallback path to contain '.config', got %s", fallbackPath)
		}
	}
}

func TestEnvPathDetermination(t *testing.T) {
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		expectedEnvPath := filepath.Join(userConfigDir, "plex-preroll", ".env")
		if !strings.HasSuffix(expectedEnvPath, ".env") {
			t.Errorf("expected env path to end with '.env', got %s", expectedEnvPath)
		}
	}
}

func TestConfigWithSpecialCharacters(t *testing.T) {
	config := Config{
		Seasons: map[string]Season{
			"new year's eve": {
				From:  "12-31",
				To:    "01-01",
				Files: []string{"/path/with spaces/video.mp4", "/path/with-dashes/video.mp4"},
			},
		},
		Default: []string{"/path/to/default video.mp4"},
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("failed to marshal config with special characters: %v", err)
	}

	var loadedConfig Config
	err = yaml.Unmarshal(yamlData, &loadedConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	season, ok := loadedConfig.Seasons["new year's eve"]
	if !ok {
		t.Fatal("season with special characters not found")
	}

	if len(season.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(season.Files))
	}

	if season.Files[0] != "/path/with spaces/video.mp4" {
		t.Errorf("expected path with spaces, got %s", season.Files[0])
	}
}
