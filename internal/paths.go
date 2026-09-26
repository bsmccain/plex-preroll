package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHomePath expands ~ and ~/ at the start of a configuration path.
// Named users such as ~alice are not supported.
func ExpandHomePath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	if path != "~" && !strings.HasPrefix(path, "~/") && !(os.PathSeparator == '\\' && strings.HasPrefix(path, `~\`)) {
		return "", fmt.Errorf("unsupported home directory path %q; use ~ or an absolute path", path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
