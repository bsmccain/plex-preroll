package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, tc := range []struct {
		path, want string
		fail       bool
	}{
		{"~", home, false},
		{"~/config.yaml", filepath.Join(home, "config.yaml"), false},
		{"config.yaml", "config.yaml", false},
		{filepath.Join(home, "config.yaml"), filepath.Join(home, "config.yaml"), false},
		{"~another/config.yaml", "", true},
	} {
		got, err := ExpandHomePath(tc.path)
		if (err != nil) != tc.fail || got != tc.want {
			t.Errorf("ExpandHomePath(%q) = %q, %v; want %q, error=%v", tc.path, got, err, tc.want, tc.fail)
		}
	}
	if os.PathSeparator == '\\' {
		got, err := ExpandHomePath(`~\config.yaml`)
		if err != nil || got != filepath.Join(home, "config.yaml") {
			t.Fatalf("Windows home path = %q, %v", got, err)
		}
	}
}
