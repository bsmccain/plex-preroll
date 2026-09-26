package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var videoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".ts":   true,
	".m2ts": true,
	".flv":  true,
	".asf":  true,
	".3gp":  true,
	".webm": true,
}

func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return videoExtensions[ext]
}

// BuildPrerollFileList resolves every configured path before an update is sent.
// Invalid paths return no files so callers cannot send a partial or empty update.
// An empty input is allowed when the user intends to clear the preroll setting.
func BuildPrerollFileList(paths []string) ([]string, error) {
	files := []string{}

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("configured path %q must be absolute", path)
		}
		if strings.ContainsAny(path, ",;") {
			return nil, fmt.Errorf("configured path %q contains a Plex list separator (comma or semicolon)", path)
		}

		fi, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("cannot access configured path %q: %w", path, err)
		}

		if fi.IsDir() {
			dirFiles, err := os.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("cannot read configured directory %q: %w", path, err)
			}
			found := false
			for _, dirFile := range dirFiles {
				if dirFile.IsDir() || !isVideoFile(dirFile.Name()) {
					continue
				}
				fullPath := filepath.Join(path, dirFile.Name())
				if err := validateVideoFile(fullPath); err != nil {
					return nil, err
				}
				files = append(files, fullPath)
				found = true
			}
			if !found {
				return nil, fmt.Errorf("configured directory %q contains no usable video files", path)
			}
		} else {
			if err := validateVideoFile(path); err != nil {
				return nil, err
			}
			files = append(files, path)
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}

func validateVideoFile(path string) error {
	if !isVideoFile(path) {
		return fmt.Errorf("configured file %q has an unsupported video extension", path)
	}
	if strings.ContainsAny(path, ",;") {
		return fmt.Errorf("video path %q contains a Plex list separator (comma or semicolon)", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access video file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("video path %q is not a regular file", path)
	}
	// Plex opens the video under its own account; this process only needs to
	// resolve its path and confirm that it is a regular file.
	return nil
}
