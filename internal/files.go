package internal

import (
	"log"
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
}

func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return videoExtensions[ext]
}

func BuildPrerollFileList(paths []string) ([]string, error) {
	files := []string{}

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			log.Printf("Path %q is not absolute. Skipping.", path)
			continue
		}

		fi, err := os.Stat(path)
		if err != nil {
			log.Printf("Failed to stat configured path %q. Skipping.", path)
			continue
		}

		if fi.IsDir() {
			dirFiles, err := os.ReadDir(path)
			if err != nil {
				log.Printf("Failed to read directory %q. Skipping.", path)
				continue
			}
			for _, dirFile := range dirFiles {
				if !dirFile.IsDir() {
					fullPath := filepath.Join(path, dirFile.Name())
					if isVideoFile(dirFile.Name()) {
						files = append(files, fullPath)
					}
				}
			}
		} else {
			if isVideoFile(path) {
				files = append(files, path)
			} else {
				log.Printf("Specified file %q is not a recognized video file. Skipping.", path)
			}
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}