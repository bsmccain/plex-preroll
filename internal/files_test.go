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

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// Supported video extensions
		{name: "mp4", filename: "video.mp4", expected: true},
		{name: "mkv", filename: "video.mkv", expected: true},
		{name: "avi", filename: "video.avi", expected: true},
		{name: "mov", filename: "video.mov", expected: true},
		{name: "wmv", filename: "video.wmv", expected: true},
		{name: "ts", filename: "video.ts", expected: true},
		{name: "m2ts", filename: "video.m2ts", expected: true},
		{name: "flv", filename: "video.flv", expected: true},
		{name: "asf", filename: "video.asf", expected: true},
		{name: "3gp", filename: "video.3gp", expected: true},

		// Case insensitive
		{name: "MP4 uppercase", filename: "video.MP4", expected: true},
		{name: "MKV uppercase", filename: "video.MKV", expected: true},
		{name: "Mp4 mixed case", filename: "video.Mp4", expected: true},
		{name: "AVI uppercase", filename: "VIDEO.AVI", expected: true},

		// Non-video extensions
		{name: "txt", filename: "notes.txt", expected: false},
		{name: "jpg", filename: "image.jpg", expected: false},
		{name: "png", filename: "image.png", expected: false},
		{name: "pdf", filename: "document.pdf", expected: false},
		{name: "mp3", filename: "audio.mp3", expected: false},
		{name: "wav", filename: "audio.wav", expected: false},
		{name: "exe", filename: "program.exe", expected: false},
		{name: "zip", filename: "archive.zip", expected: false},

		// Edge cases
		{name: "no extension", filename: "videofile", expected: false},
		{name: "hidden file", filename: ".mp4", expected: true},
		{name: "double extension", filename: "video.old.mp4", expected: true},
		{name: "empty string", filename: "", expected: false},
		{name: "just dot", filename: ".", expected: false},
		{name: "path with mp4", filename: "/path/to/video.mp4", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVideoFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isVideoFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestBuildPrerollFileListAllVideoExtensions(t *testing.T) {
	tempDir := t.TempDir()

	// Create files for all supported video extensions
	extensions := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".ts", ".m2ts", ".flv", ".asf", ".3gp"}
	var expectedFiles []string

	for _, ext := range extensions {
		filename := "video" + ext
		fullPath := filepath.Join(tempDir, filename)
		os.WriteFile(fullPath, []byte("fake video"), 0644)
		expectedFiles = append(expectedFiles, fullPath)
	}

	// Also create non-video files that should be ignored
	nonVideoFiles := []string{"notes.txt", "image.jpg", "audio.mp3"}
	for _, filename := range nonVideoFiles {
		fullPath := filepath.Join(tempDir, filename)
		os.WriteFile(fullPath, []byte("not a video"), 0644)
	}

	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(expectedFiles)
	sort.Strings(got)

	if !reflect.DeepEqual(got, expectedFiles) {
		t.Errorf("expected %v, got %v", expectedFiles, got)
	}
}

func TestBuildPrerollFileListDeduplication(t *testing.T) {
	tempDir := t.TempDir()

	file := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(file, []byte("fake video"), 0644)

	// Pass the same file multiple times
	inputs := []string{file, file, file}

	got, err := BuildPrerollFileList(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("expected 1 file after deduplication, got %d: %v", len(got), got)
	}
}

func TestBuildPrerollFileListNestedDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create a nested directory structure
	subdir := filepath.Join(tempDir, "subdir")
	nestedDir := filepath.Join(subdir, "nested")
	os.MkdirAll(nestedDir, 0755)

	// Create video files at different levels
	topLevelFile := filepath.Join(tempDir, "top.mp4")
	subdirFile := filepath.Join(subdir, "sub.mp4")
	nestedFile := filepath.Join(nestedDir, "nested.mp4")

	os.WriteFile(topLevelFile, []byte("fake video"), 0644)
	os.WriteFile(subdirFile, []byte("fake video"), 0644)
	os.WriteFile(nestedFile, []byte("fake video"), 0644)

	// When passing tempDir, only top-level files should be returned (not recursive)
	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the top-level file should be included (not files in subdirectories)
	expected := []string{topLevelFile}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected only top-level files %v, got %v", expected, got)
	}
}

func TestBuildPrerollFileListEmptyInput(t *testing.T) {
	got, err := BuildPrerollFileList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", got)
	}
}

func TestBuildPrerollFileListNilInput(t *testing.T) {
	got, err := BuildPrerollFileList(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %v", got)
	}
}

func TestBuildPrerollFileListSortedOutput(t *testing.T) {
	tempDir := t.TempDir()

	// Create files in non-alphabetical order
	files := []string{"z_video.mp4", "a_video.mp4", "m_video.mp4"}
	for _, f := range files {
		fullPath := filepath.Join(tempDir, f)
		os.WriteFile(fullPath, []byte("fake video"), 0644)
	}

	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output is sorted
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("output not sorted: %v comes before %v", got[i-1], got[i])
		}
	}
}

func TestBuildPrerollFileListRelativePath(t *testing.T) {
	// Relative paths should be skipped
	inputs := []string{"relative/path/video.mp4", "./another/video.mkv", "simple.mp4"}

	got, err := BuildPrerollFileList(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice when all paths are relative, got %v", got)
	}
}

func TestBuildPrerollFileListMixedAbsoluteRelative(t *testing.T) {
	tempDir := t.TempDir()

	// Create an absolute path file
	absFile := filepath.Join(tempDir, "absolute.mp4")
	os.WriteFile(absFile, []byte("fake video"), 0644)

	inputs := []string{
		absFile,             // Should be included
		"relative/path.mp4", // Should be skipped
		"./another.mkv",     // Should be skipped
	}

	got, err := BuildPrerollFileList(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{absFile}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestBuildPrerollFileListSpecialCharactersInPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with special characters in names
	specialNames := []string{
		"video with spaces.mp4",
		"video_with_underscores.mp4",
		"video-with-dashes.mp4",
		"video.multiple.dots.mp4",
	}

	var expectedFiles []string
	for _, name := range specialNames {
		fullPath := filepath.Join(tempDir, name)
		os.WriteFile(fullPath, []byte("fake video"), 0644)
		expectedFiles = append(expectedFiles, fullPath)
	}

	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(expectedFiles)
	if !reflect.DeepEqual(got, expectedFiles) {
		t.Errorf("expected %v, got %v", expectedFiles, got)
	}
}

func TestBuildPrerollFileListDirectoryWithOnlyNonVideoFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create only non-video files
	nonVideoFiles := []string{"document.txt", "image.jpg", "audio.mp3", "archive.zip"}
	for _, f := range nonVideoFiles {
		fullPath := filepath.Join(tempDir, f)
		os.WriteFile(fullPath, []byte("not a video"), 0644)
	}

	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice for directory with only non-video files, got %v", got)
	}
}

func TestBuildPrerollFileListDirectoryContainingSubdirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a subdirectory (not a file)
	subdir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subdir, 0755)

	// Create a video file in the subdirectory
	subdirFile := filepath.Join(subdir, "nested.mp4")
	os.WriteFile(subdirFile, []byte("fake video"), 0644)

	// When listing tempDir, subdirectories should be skipped (files inside them are not included)
	got, err := BuildPrerollFileList([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The subdirectory itself should be skipped, so no files should be returned
	if len(got) != 0 {
		t.Errorf("expected empty slice (subdirs should be skipped), got %v", got)
	}
}

func TestBuildPrerollFileListExplicitFileAndDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory with files
	subdir := filepath.Join(tempDir, "videos")
	os.Mkdir(subdir, 0755)
	dirFile := filepath.Join(subdir, "dir_video.mp4")
	os.WriteFile(dirFile, []byte("fake video"), 0644)

	// Create an explicit file
	explicitFile := filepath.Join(tempDir, "explicit.mp4")
	os.WriteFile(explicitFile, []byte("fake video"), 0644)

	inputs := []string{explicitFile, subdir}

	got, err := BuildPrerollFileList(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{dirFile, explicitFile}
	sort.Strings(expected)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
