package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestBuildPrerollFileList(t *testing.T) {
    tempDir := t.TempDir()

    // Create a single valid file
    file1 := filepath.Join(tempDir, "one.mp4")
    os.WriteFile(file1, []byte("fake video data"), 0644)

    // Create a directory with a file in it
    subdir := filepath.Join(tempDir, "holiday")
    os.Mkdir(subdir, 0755)
    file2 := filepath.Join(subdir, "two.mp4")
    os.WriteFile(file2, []byte("another fake video"), 0644)

    // Nonexistent and relative paths
    missing := filepath.Join(tempDir, "missing.mp4")
    relative := "not/absolute.mp4"

    inputs := []string{file1, subdir, missing, relative}
	sort.Strings(inputs)

    got, err := BuildPrerollFileList(inputs)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    want := []string{
        file1,
        filepath.Join(subdir, "two.mp4"),
    }
	sort.Strings(want)

    if !reflect.DeepEqual(got, want) {
        t.Errorf("expected %v, got %v", want, got)
    }
}
