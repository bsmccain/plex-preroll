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
    os.WriteFile(file1, []byte("fake video data"), 0644) // Valid video file

    // Create a directory with mixed files
    subdir := filepath.Join(tempDir, "holiday")
    os.Mkdir(subdir, 0755)
    file2 := filepath.Join(subdir, "two.mkv") // Valid video file
    os.WriteFile(file2, []byte("another fake video"), 0644)
    file3 := filepath.Join(subdir, "notes.txt") // Non-video file
    os.WriteFile(file3, []byte("some notes"), 0644)
    file4 := filepath.Join(tempDir, "explicit_non_video.jpg") // Explicitly listed non-video file
    os.WriteFile(file4, []byte("fake image data"), 0644)


    // Nonexistent and relative paths
    missing := filepath.Join(tempDir, "missing.mp4")
    relative := "not/absolute.mp4"

    // Input paths for the test
    inputs := []string{
		file1,    // valid video file, explicitly listed
		subdir,   // directory with video and non-video files
		missing,  // should be skipped (and logged)
		relative, // should be skipped (and logged)
		file4,    // non-video file, explicitly listed, should be skipped
	}
	// sort.Strings(inputs) // Order doesn't strictly matter for input, but sorting expected output is key

    got, err := BuildPrerollFileList(inputs)
    if err != nil {
        t.Fatalf("BuildPrerollFileList returned an unexpected error: %v", err)
    }

    // Expected output: only valid, absolute video files
    want := []string{
        file1, // from explicit path
        file2, // from subdir
    }
	sort.Strings(want) // Sort for consistent comparison

    // Sort `got` as well, as the order from BuildPrerollFileList is sorted but good practice here.
    sort.Strings(got)

    if !reflect.DeepEqual(got, want) {
        t.Errorf("BuildPrerollFileList() got = %v, want %v", got, want)
    }

	// Test case with an explicitly listed non-video file
	nonVideoInputs := []string{file4}
	gotNonVideo, err := BuildPrerollFileList(nonVideoInputs)
	if err != nil {
		t.Fatalf("BuildPrerollFileList with non-video returned an unexpected error: %v", err)
	}
	if len(gotNonVideo) != 0 {
		t.Errorf("BuildPrerollFileList with non-video file: expected empty list, got %v", gotNonVideo)
	}

	// Test case with an empty directory
	emptySubdir := filepath.Join(tempDir, "empty_holiday")
	os.Mkdir(emptySubdir, 0755)
	emptyDirInputs := []string{emptySubdir}
	gotEmptyDir, err := BuildPrerollFileList(emptyDirInputs)
	if err != nil {
		t.Fatalf("BuildPrerollFileList with empty dir returned an unexpected error: %v", err)
	}
	if len(gotEmptyDir) != 0 {
		t.Errorf("BuildPrerollFileList with empty directory: expected empty list, got %v", gotEmptyDir)
	}
}
