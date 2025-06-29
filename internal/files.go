package internal

import (
	"log"
	"os"
	"path/filepath"
	"sort"
)

func BuildPrerollFileList(paths []string) ([]string, error) {
	files := []string{}

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			log.Printf("path %s is not absolute, skipping", path)
			continue
		}

		fi, err := os.Stat(path)
		if err != nil {
			log.Printf("failed to stat path %s: %v", path, err)
			continue
		}

		if fi.IsDir() {
			dirFiles, err := os.ReadDir(path)
			if err != nil {
				log.Printf("failed to read directory %s: %v", path, err)
				continue
			}
			for _, dirFile := range dirFiles {
				if !dirFile.IsDir() {
					files = append(files, filepath.Join(path, dirFile.Name()))
				}
			}
		} else {
			files = append(files, path)
		}
	}

	sort.Strings(files)
	return files, nil
}