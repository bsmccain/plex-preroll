package internal

import (
	"path/filepath"
	"reflect"
	"sort"
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
