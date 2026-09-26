package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func writeTestVideo(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("fake video data"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPrerollFileList(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "explicit.mp4")
	writeTestVideo(t, file)
	videos := filepath.Join(dir, "videos")
	if err := os.Mkdir(videos, 0755); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, name := range []string{
		"video.mp4", "video.mkv", "video.avi", "video.mov", "video.wmv",
		"video.ts", "video.m2ts", "video.flv", "video.asf", "video.3gp", "video.webm",
		"video.MP4", "video with spaces.mp4", "видео.mp4", "video&special.mp4",
	} {
		path := filepath.Join(videos, name)
		writeTestVideo(t, path)
		want = append(want, path)
	}
	writeTestVideo(t, filepath.Join(videos, "notes.txt"))
	nested := filepath.Join(videos, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestVideo(t, filepath.Join(nested, "nested.mp4"))
	want = append(want, file)
	sort.Strings(want)

	// Explicit files and directories can overlap. The result is sorted, unique,
	// and includes only immediate files from each configured directory.
	got, err := BuildPrerollFileList([]string{file, videos, file, want[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildPrerollFileList() = %v, want %v", got, want)
	}
}

func TestBuildPrerollFileListErrorsDiscardAllFiles(t *testing.T) {
	for _, tc := range []struct {
		name string
		path func(*testing.T, string) string
		want string
	}{
		{
			name: "relative path",
			path: func(t *testing.T, dir string) string { return "relative.mp4" },
			want: "must be absolute",
		},
		{
			name: "missing file",
			path: func(t *testing.T, dir string) string { return filepath.Join(dir, "missing.mp4") },
			want: "cannot access configured path",
		},
		{
			name: "explicit unsupported file",
			path: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "notes.txt")
				writeTestVideo(t, path)
				return path
			},
			want: "unsupported video extension",
		},
		{
			name: "empty directory",
			path: func(t *testing.T, dir string) string { return t.TempDir() },
			want: "contains no usable video files",
		},
		{
			name: "directory without videos",
			path: func(t *testing.T, dir string) string {
				path := t.TempDir()
				writeTestVideo(t, filepath.Join(path, "notes.txt"))
				return path
			},
			want: "contains no usable video files",
		},
		{
			name: "directory with only nested videos",
			path: func(t *testing.T, dir string) string {
				path := t.TempDir()
				nested := filepath.Join(path, "nested")
				if err := os.Mkdir(nested, 0755); err != nil {
					t.Fatal(err)
				}
				writeTestVideo(t, filepath.Join(nested, "video.mp4"))
				return path
			},
			want: "contains no usable video files",
		},
		{
			name: "semicolon in explicit filename",
			path: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "video;one.mp4")
				writeTestVideo(t, path)
				return path
			},
			want: "Plex list separator",
		},
		{
			name: "comma in directory video filename",
			path: func(t *testing.T, dir string) string {
				path := t.TempDir()
				writeTestVideo(t, filepath.Join(path, "video,one.mp4"))
				return path
			},
			want: "Plex list separator",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			good := filepath.Join(dir, "good.mp4")
			writeTestVideo(t, good)
			got, err := BuildPrerollFileList([]string{good, tc.path(t, dir)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("BuildPrerollFileList() error = %v, want %q", err, tc.want)
			}
			if got != nil {
				t.Fatalf("expected no partial file list on error, got %v", got)
			}
		})
	}
}

func TestBuildPrerollFileListUnreadablePaths(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "file"
		if directory {
			name = "directory"
		}
		t.Run(name, func(t *testing.T) {
			path := t.TempDir()
			if !directory {
				path = filepath.Join(path, "video.mp4")
				writeTestVideo(t, path)
			}
			if err := os.Chmod(path, 0000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(path, 0755) })
			if f, err := os.Open(path); err == nil {
				f.Close()
				t.Skip("current user can open paths with no permission bits")
			}
			if got, err := BuildPrerollFileList([]string{path}); err == nil || got != nil {
				t.Fatalf("unreadable path returned %v, %v; want no files and an error", got, err)
			}
		})
	}
}

func TestBuildPrerollFileListSymlinks(t *testing.T) {
	for _, targetKind := range []string{"file", "missing", "directory"} {
		t.Run(targetKind, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(t.TempDir(), "target.mp4")
			switch targetKind {
			case "file":
				writeTestVideo(t, target)
			case "directory":
				if err := os.Mkdir(target, 0755); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(dir, "video.mp4")
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("cannot create symlink: %v", err)
			}
			got, err := BuildPrerollFileList([]string{dir})
			if targetKind == "file" {
				if err != nil || !reflect.DeepEqual(got, []string{link}) {
					t.Fatalf("valid symlink returned %v, %v", got, err)
				}
			} else if err == nil || got != nil {
				t.Fatalf("invalid video symlink returned %v, %v; want no files and an error", got, err)
			}
		})
	}
}

func TestBuildPrerollFileListEmptyInput(t *testing.T) {
	for _, paths := range [][]string{nil, {}} {
		got, err := BuildPrerollFileList(paths)
		if err != nil || len(got) != 0 {
			t.Fatalf("empty input returned %v, %v; want empty list and no error", got, err)
		}
	}
}

func TestIsVideoFile(t *testing.T) {
	for _, filename := range []string{"video.mp4", "video.MKV", "video.webm", "video.old.mp4"} {
		if !isVideoFile(filename) {
			t.Errorf("supported video %q was rejected", filename)
		}
	}
	for _, filename := range []string{"notes.txt", "image.jpg", "audio.mp3", "video", "", "."} {
		if isVideoFile(filename) {
			t.Errorf("unsupported video %q was accepted", filename)
		}
	}
}
