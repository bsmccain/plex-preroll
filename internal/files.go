package internal

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var videoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
	".mpg":  true,
	".mpeg": true,
	".m4v":  true,
	".ts":   true,
}

func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return videoExtensions[ext]
}

func BuildPrerollFileList(paths []string) ([]string, error) {
	files := []string{}

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			log.Printf("Warning: Path %q is not absolute, skipping.", path)
			continue
		}

		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Warning: Configured path %q does not exist, skipping.", path)
				continue // Skip non-existent paths
			}
			// For other stat errors (e.g. permission denied) on a configured path:
			log.Printf("Error: Failed to stat configured path %q: %v. This path will be skipped.", path, err)
			// Decide if this should be a fatal error or just skip this path.
			// For now, let's skip, but this could be made stricter.
			// If it's a directory and unstattable, we can't list its contents.
			// If it's a file and unstattable, we can't use it.
			// To make it stricter for directories:
			// if fi != nil && fi.IsDir() { // Check fi nil in case stat failed entirely
			//    return nil, fmt.Errorf("failed to stat configured directory %q: %w", path, err)
			// }
			continue // Default: skip if error other than NotExist
		}

		if fi.IsDir() {
			dirFiles, err := os.ReadDir(path)
			if err != nil {
				// This is a more critical error for a directory.
				// If we can't read a configured directory, we should probably error out.
				log.Printf("Error: Failed to read configured directory %q: %v.", path, err)
				return nil, fmt.Errorf("failed to read configured directory %q: %w", path, err)
			}
			for _, dirFile := range dirFiles {
				if !dirFile.IsDir() {
					fullPath := filepath.Join(path, dirFile.Name())
					if isVideoFile(dirFile.Name()) {
						files = append(files, fullPath)
					} else {
						log.Printf("Info: Skipping non-video file %q in directory %q.", dirFile.Name(), path)
					}
				}
			}
		} else { // It's a file
			if isVideoFile(path) {
				files = append(files, path)
			} else {
				log.Printf("Warning: Specified file %q is not a recognized video file, skipping.", path)
			}
		}
	}

	sort.Strings(files) // Sort to ensure consistent order
	return files, nil
}