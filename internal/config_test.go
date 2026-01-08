package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	path := filepath.Join("testdata", "prerolls_test.yaml")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		name     string
		date     string
		expected []string
	}{
		{
			name:     "summer season",
			date:     "2025-06-28",
			expected: []string{"beach_day.mp4", "ice_cream.mp4"},
		},
		{
			name:     "thanksgiving season",
			date:     "2025-11-15",
			expected: []string{"turkeys.mp4", "mashed_potatoes.mp4"},
		},
		{
			name:     "christmas season",
			date:     "2025-12-25",
			expected: []string{"jingle_bells.mp4", "santa_hat.mp4"},
		},
		{
			name:     "christmas season (overlapping with thanksgiving)",
			date:     "2025-11-25",
			expected: []string{"jingle_bells.mp4", "santa_hat.mp4", "turkeys.mp4", "mashed_potatoes.mp4"},
		},
		{
			name:     "christmas season (early january)",
			date:     "2025-01-03",
			expected: []string{"jingle_bells.mp4", "santa_hat.mp4"},
		},
		{
			name:     "default (spring)",
			date:     "2025-04-15",
			expected: []string{"generic_intro.mp4", "plex_logo_spin.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the date
			parsedDate, err := time.Parse("2006-01-02", tt.date)
			if err != nil {
				t.Fatalf("failed to parse date %s: %v", tt.date, err)
			}

			prerolls := cfg.GetPrerolls(parsedDate)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(prerolls, tt.expected) {
				t.Errorf("expected prerolls to be %v, got %v", tt.expected, prerolls)
			}
		})
	}
}

func TestNormalizeMMDD(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "already normalized",
			input:       "01-15",
			expected:    "01-15",
			expectError: false,
		},
		{
			name:        "single digit month",
			input:       "1-15",
			expected:    "01-15",
			expectError: false,
		},
		{
			name:        "single digit day",
			input:       "12-5",
			expected:    "12-05",
			expectError: false,
		},
		{
			name:        "both single digit",
			input:       "1-1",
			expected:    "01-01",
			expectError: false,
		},
		{
			name:        "december 31",
			input:       "12-31",
			expected:    "12-31",
			expectError: false,
		},
		{
			name:        "invalid month 13",
			input:       "13-01",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid month 0",
			input:       "0-15",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid day 32",
			input:       "01-32",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid day 0",
			input:       "01-0",
			expected:    "",
			expectError: true,
		},
		{
			name:        "missing separator",
			input:       "0115",
			expected:    "",
			expectError: true,
		},
		{
			name:        "wrong separator",
			input:       "01/15",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "non-numeric month",
			input:       "ab-15",
			expected:    "",
			expectError: true,
		},
		{
			name:        "non-numeric day",
			input:       "01-cd",
			expected:    "",
			expectError: true,
		},
		{
			name:        "extra parts",
			input:       "01-15-2025",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeMMDD(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid.yaml")
		content := `seasons:
  - name: test
    from: "01-01
    to: "01-31"` // Missing closing quote
		os.WriteFile(configPath, []byte(content), 0644)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})

	t.Run("invalid from date", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		content := `seasons:
  - name: test
    from: "invalid"
    to: "01-31"
    files:
      - test.mp4`
		os.WriteFile(configPath, []byte(content), 0644)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid from date, got nil")
		}
		if !strings.Contains(err.Error(), "invalid 'from' date") {
			t.Errorf("expected error message about 'from' date, got: %v", err)
		}
	})

	t.Run("invalid to date", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		content := `seasons:
  - name: test
    from: "01-01"
    to: "13-45"
    files:
      - test.mp4`
		os.WriteFile(configPath, []byte(content), 0644)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid to date, got nil")
		}
		if !strings.Contains(err.Error(), "invalid 'to' date") {
			t.Errorf("expected error message about 'to' date, got: %v", err)
		}
	})
}

func TestLoadConfigValid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := `seasons:
  - name: summer
    from: "6-1"
    to: "8-31"
    files:
      - summer1.mp4
      - summer2.mp4
  - name: winter
    from: "12-1"
    to: "2-28"
    files:
      - winter.mp4
default:
  - default.mp4`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Seasons) != 2 {
		t.Errorf("expected 2 seasons, got %d", len(cfg.Seasons))
	}

	// Verify date normalization
	if cfg.Seasons[0].From != "06-01" {
		t.Errorf("expected normalized from date 06-01, got %s", cfg.Seasons[0].From)
	}
	if cfg.Seasons[0].To != "08-31" {
		t.Errorf("expected normalized to date 08-31, got %s", cfg.Seasons[0].To)
	}

	if len(cfg.Default) != 1 {
		t.Errorf("expected 1 default file, got %d", len(cfg.Default))
	}
}

func TestGetPrerollsBoundaryDates(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := `seasons:
  - name: summer
    from: "06-01"
    to: "08-31"
    files:
      - summer.mp4
default:
  - default.mp4`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		name     string
		date     string
		expected []string
	}{
		{
			name:     "day before season start",
			date:     "2025-05-31",
			expected: []string{"default.mp4"},
		},
		{
			name:     "first day of season",
			date:     "2025-06-01",
			expected: []string{"summer.mp4"},
		},
		{
			name:     "last day of season",
			date:     "2025-08-31",
			expected: []string{"summer.mp4"},
		},
		{
			name:     "day after season end",
			date:     "2025-09-01",
			expected: []string{"default.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedDate, _ := time.Parse("2006-01-02", tt.date)
			prerolls := cfg.GetPrerolls(parsedDate)
			if !reflect.DeepEqual(prerolls, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, prerolls)
			}
		})
	}
}

func TestGetPrerollsYearCrossingBoundaries(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := `seasons:
  - name: holiday
    from: "12-20"
    to: "01-10"
    files:
      - holiday.mp4
default:
  - default.mp4`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		name     string
		date     string
		expected []string
	}{
		{
			name:     "before year-crossing season",
			date:     "2025-12-19",
			expected: []string{"default.mp4"},
		},
		{
			name:     "first day of year-crossing season",
			date:     "2025-12-20",
			expected: []string{"holiday.mp4"},
		},
		{
			name:     "new years eve",
			date:     "2025-12-31",
			expected: []string{"holiday.mp4"},
		},
		{
			name:     "new years day",
			date:     "2025-01-01",
			expected: []string{"holiday.mp4"},
		},
		{
			name:     "last day of year-crossing season",
			date:     "2025-01-10",
			expected: []string{"holiday.mp4"},
		},
		{
			name:     "day after year-crossing season ends",
			date:     "2025-01-11",
			expected: []string{"default.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedDate, _ := time.Parse("2006-01-02", tt.date)
			prerolls := cfg.GetPrerolls(parsedDate)
			if !reflect.DeepEqual(prerolls, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, prerolls)
			}
		})
	}
}

func TestGetPrerollsEmptyConfig(t *testing.T) {
	cfg := &Config{
		Seasons: []Season{},
		Default: []string{"fallback.mp4"},
	}

	parsedDate, _ := time.Parse("2006-01-02", "2025-06-15")
	prerolls := cfg.GetPrerolls(parsedDate)

	expected := []string{"fallback.mp4"}
	if !reflect.DeepEqual(prerolls, expected) {
		t.Errorf("expected %v, got %v", expected, prerolls)
	}
}

func TestGetPrerollsNoDefaultNoMatch(t *testing.T) {
	cfg := &Config{
		Seasons: []Season{
			{
				Name:  "summer",
				From:  "06-01",
				To:    "08-31",
				Files: []string{"summer.mp4"},
			},
		},
		Default: nil,
	}

	parsedDate, _ := time.Parse("2006-01-02", "2025-01-15")
	prerolls := cfg.GetPrerolls(parsedDate)

	if len(prerolls) != 0 {
		t.Errorf("expected empty slice, got %v", prerolls)
	}
}

func TestGetPrerollsMultipleOverlapping(t *testing.T) {
	cfg := &Config{
		Seasons: []Season{
			{
				Name:  "season1",
				From:  "06-01",
				To:    "06-30",
				Files: []string{"a.mp4", "b.mp4"},
			},
			{
				Name:  "season2",
				From:  "06-15",
				To:    "07-15",
				Files: []string{"c.mp4", "d.mp4"},
			},
			{
				Name:  "season3",
				From:  "06-20",
				To:    "06-25",
				Files: []string{"e.mp4"},
			},
		},
		Default: []string{"default.mp4"},
	}

	parsedDate, _ := time.Parse("2006-01-02", "2025-06-22")
	prerolls := cfg.GetPrerolls(parsedDate)

	expected := []string{"a.mp4", "b.mp4", "c.mp4", "d.mp4", "e.mp4"}
	sort.Strings(expected)

	if !reflect.DeepEqual(prerolls, expected) {
		t.Errorf("expected %v, got %v", expected, prerolls)
	}
}

func TestGetPrerollsSortedOutput(t *testing.T) {
	cfg := &Config{
		Seasons: []Season{
			{
				Name:  "test",
				From:  "01-01",
				To:    "12-31",
				Files: []string{"z.mp4", "a.mp4", "m.mp4"},
			},
		},
		Default: []string{},
	}

	parsedDate, _ := time.Parse("2006-01-02", "2025-06-15")
	prerolls := cfg.GetPrerolls(parsedDate)

	expected := []string{"a.mp4", "m.mp4", "z.mp4"}
	if !reflect.DeepEqual(prerolls, expected) {
		t.Errorf("expected sorted output %v, got %v", expected, prerolls)
	}
}
