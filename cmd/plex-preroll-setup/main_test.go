package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
)

func setupInput(configPath, envPath string, trailing ...string) string {
	lines := []string{"test-token", "", configPath, envPath, "n", ""}
	return strings.Join(append(lines, trailing...), "\n") + "\n"
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return // Windows controls file access through ACLs, not Unix mode bits.
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %04o, want %04o", path, got, want)
	}
}

func TestSetupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config with spaces # and $dollar.yaml")
	envPath := filepath.Join(dir, ".env")
	video := filepath.Join(dir, "holiday video.mp4")
	defaultVideo := filepath.Join(dir, "default.mp4")
	token := `test-token # with "quotes" and $DOLLAR`
	input := strings.Join([]string{
		"", token, "ftp://invalid", "https://user:password@example.com", "http://example.com#fragment", "",
		configPath, envPath, "maybe", "y", "", "christmas", "02-31", "12-25", "12-25",
		"relative.mp4", filepath.Join(dir, "comma,video.mp4"), video, "", "n", defaultVideo, "",
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := run(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	config, err := internal.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("setup output must load in the application: %v", err)
	}
	want := &internal.Config{
		Seasons: []internal.Season{{Name: "christmas", From: "12-25", To: "12-25", Files: []string{video}}},
		Default: []string{defaultVideo},
	}
	if !reflect.DeepEqual(config, want) {
		t.Errorf("config = %#v, want %#v", config, want)
	}
	env, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatal(err)
	}
	resolvedConfigPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		t.Fatal(err)
	}
	wantEnv := map[string]string{"PLEX_TOKEN": token, "PLEX_SERVER_URL": "http://localhost:32400", "PLEX_PREROLL_CFG": resolvedConfigPath}
	if !reflect.DeepEqual(env, wantEnv) {
		t.Errorf("dotenv round trip changed setup values: got %#v, want %#v", env, wantEnv)
	}
	if strings.Contains(output.String(), token) {
		t.Error("setup echoed the token")
	}
	assertMode(t, envPath, 0600)
}

func TestSetupRepromptsForNonRegularOutputPaths(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			invalidPath := filepath.Join(dir, "invalid")
			var err error
			if kind == "directory" {
				err = os.Mkdir(invalidPath, 0700)
			} else {
				err = os.Symlink(filepath.Join(dir, "missing-target"), invalidPath)
			}
			if err != nil {
				if runtime.GOOS == "windows" && kind == "symlink" {
					t.Skipf("symlinks unavailable: %v", err)
				}
				t.Fatal(err)
			}
			configPath := filepath.Join(dir, "config.yaml")
			envPath := filepath.Join(dir, ".env")
			video := filepath.Join(dir, "holiday.mp4")
			input := strings.Join([]string{
				"test-token", "", invalidPath, configPath, invalidPath, envPath,
				"y", "holiday", "12-25", "12-25", video, "", "n", "",
			}, "\n") + "\n"
			var output bytes.Buffer
			if err := run(strings.NewReader(input), &output); err != nil {
				t.Fatalf("setup did not recover from invalid output paths: %v", err)
			}
			config, err := internal.LoadConfig(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(config.Seasons) != 1 || config.Seasons[0].Name != "holiday" || !reflect.DeepEqual(config.Seasons[0].Files, []string{video}) {
				t.Errorf("setup lost season input after correcting output paths: %#v", config.Seasons)
			}
			warning := "is not a regular file. Enter a different path."
			transcript := output.String()
			if strings.Count(transcript, warning) != 2 || strings.LastIndex(transcript, warning) > strings.Index(transcript, "Add a season?") {
				t.Errorf("setup must correct both paths before collecting seasons: %s", transcript)
			}
			if _, err := godotenv.Read(envPath); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSetupEOFLeavesNoFiles(t *testing.T) {
	for _, stop := range []string{"token", "server", "paths", "season date", "default", "confirmation"} {
		t.Run(stop, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			envPath := filepath.Join(dir, ".env")
			inputs := map[string]string{
				"token": "", "server": "test-token\n", "paths": "test-token\n\n",
				"season date":  strings.Join([]string{"test-token", "", configPath, envPath, "y", "holiday", "bad-date", ""}, "\n"),
				"default":      strings.Join([]string{"test-token", "", configPath, envPath, "n", ""}, "\n"),
				"confirmation": setupInput(configPath, envPath),
			}
			if stop == "confirmation" {
				if err := os.WriteFile(envPath, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := run(strings.NewReader(inputs[stop]), io.Discard)
			if !errors.Is(err, io.EOF) {
				t.Fatalf("error = %v, want EOF", err)
			}
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Errorf("config was written before input completed: %v", err)
			}
			if stop == "confirmation" {
				data, err := os.ReadFile(envPath)
				if err != nil || string(data) != "original" {
					t.Fatalf("existing .env changed: %q, %v", data, err)
				}
			} else if _, err := os.Stat(envPath); !os.IsNotExist(err) {
				t.Errorf(".env was written before input completed: %v", err)
			}
		})
	}
}

func TestSetupOverwrite(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answers []string
		replace bool
	}{
		{"decline config", []string{"n"}, false},
		{"decline env", []string{"y", "n"}, false},
		{"confirm both", []string{"yes", "yes"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			envPath := filepath.Join(dir, ".env")
			for _, path := range []string{configPath, envPath} {
				if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := run(strings.NewReader(setupInput(configPath, envPath, tc.answers...)), io.Discard)
			if tc.replace {
				if err != nil {
					t.Fatal(err)
				}
				assertMode(t, envPath, 0600)
				if _, err := internal.LoadConfig(configPath); err != nil {
					t.Fatal(err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "cancelled") {
					t.Fatalf("error = %v, want cancelled", err)
				}
				for _, path := range []string{configPath, envPath} {
					data, err := os.ReadFile(path)
					if err != nil || string(data) != "original" {
						t.Errorf("existing %s changed: %q, %v", path, data, err)
					}
				}
			}
		})
	}
}

func assertNoSetupTemps(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(entry.Name(), ".plex-preroll-") {
			t.Errorf("setup left a temporary file at %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetupPreservesConfigWhenEnvCannotBeStaged(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("original config"), 0600); err != nil {
		t.Fatal(err)
	}
	// A regular file cannot hold the adjacent temporary .env. This filesystem
	// failure is deterministic even when tests run with elevated privileges.
	blockedParent := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blockedParent, []byte("original blocker"), 0600); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(blockedParent, ".env")
	err := saveSetupFiles(configPath, []byte("replacement config"), envPath, []byte("PLEX_TOKEN=secret"))
	if err == nil || !strings.Contains(err.Error(), "preparing .env") {
		t.Fatalf("error = %v, want .env staging failure", err)
	}
	config, err := os.ReadFile(configPath)
	if err != nil || string(config) != "original config" {
		t.Errorf("existing config changed after failed setup: %q, %v", config, err)
	}
	assertMode(t, configPath, 0600)
	assertNoSetupTemps(t, dir)
}

func TestSetupRollsBackConfigWhenEnvCannotBeReplaced(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing config=%t", existing), func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			if existing {
				if err := os.WriteFile(configPath, []byte("original config"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			// Both temporary outputs can be written, but the .env rename fails.
			envPath := filepath.Join(dir, ".env")
			if err := os.Mkdir(envPath, 0700); err != nil {
				t.Fatal(err)
			}
			markerPath := filepath.Join(envPath, "keep")
			if err := os.WriteFile(markerPath, []byte("untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			err := saveSetupFiles(configPath, []byte("replacement config"), envPath, []byte("PLEX_TOKEN=secret"))
			if err == nil || !strings.Contains(err.Error(), "writing .env") {
				t.Fatalf("error = %v, want .env replacement failure", err)
			}
			if existing {
				config, err := os.ReadFile(configPath)
				if err != nil || string(config) != "original config" {
					t.Errorf("failed setup did not restore original config: %q, %v", config, err)
				}
				assertMode(t, configPath, 0600)
			} else if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Errorf("failed setup left a new config: %v", err)
			}
			marker, err := os.ReadFile(markerPath)
			if err != nil || string(marker) != "untouched" {
				t.Errorf("failed setup changed .env target: %q, %v", marker, err)
			}
			assertNoSetupTemps(t, dir)
		})
	}
}

func TestSetupRejectsUnsafeOutputPaths(t *testing.T) {
	for _, kind := range []string{"same path", "case-only difference", "directory", "symlink", "hardlink", "parent symlink alias"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			envPath := filepath.Join(dir, ".env")
			var err error
			switch kind {
			case "same path":
				envPath = configPath
			case "case-only difference":
				envPath = filepath.Join(dir, "CONFIG.YAML")
			case "directory":
				err = os.Mkdir(envPath, 0700)
			case "symlink":
				err = os.Symlink(filepath.Join(dir, "target"), envPath)
			case "hardlink":
				if err = os.WriteFile(configPath, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
				err = os.Link(configPath, envPath)
			case "parent symlink alias":
				alias := filepath.Join(dir, "alias")
				err = os.Symlink(dir, alias)
				configPath = filepath.Join(dir, "new", "settings")
				envPath = filepath.Join(alias, "new", "settings")
			}
			if err != nil {
				if runtime.GOOS == "windows" && strings.Contains(kind, "link") {
					t.Skipf("links unavailable: %v", err)
				}
				t.Fatal(err)
			}
			err = run(strings.NewReader(setupInput(configPath, envPath, "y", "y")), io.Discard)
			if err == nil {
				t.Fatal("setup accepted unsafe output paths")
			}
			if kind == "hardlink" {
				data, err := os.ReadFile(configPath)
				if err != nil || string(data) != "original" {
					t.Fatalf("existing file changed: %q, %v", data, err)
				}
			} else if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Errorf("config was written on failure: %v", err)
			}
		})
	}
}

func TestSetupExpandsHomePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	input := strings.Join([]string{"test-token", "", "~/settings/config.yaml", "~/settings/.env", "n", "~/videos/default.mp4", ""}, "\n") + "\n"
	if err := run(strings.NewReader(input), io.Discard); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, "settings", "config.yaml")
	config, err := internal.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Default, []string{filepath.Join(home, "videos", "default.mp4")}) {
		t.Errorf("default paths = %v", config.Default)
	}
	env, err := godotenv.Read(filepath.Join(home, "settings", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	// macOS may resolve its temporary directory through /private.
	resolved, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if env["PLEX_PREROLL_CFG"] != resolved {
		t.Errorf("config path = %q, want %q", env["PLEX_PREROLL_CFG"], resolved)
	}
}

func TestSetupCommand(t *testing.T) {
	binDir := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	build := func(name, pkg string) string {
		t.Helper()
		binary := filepath.Join(binDir, name+suffix)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", binary, pkg)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building %s: %v\n%s", name, err, output)
		}
		return binary
	}
	setup := build("plex-preroll-setup", ".")
	app := build("plex-preroll", "../plex-preroll")

	t.Run("generated configuration works in main command", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		envPath := filepath.Join(dir, ".env")
		video := filepath.Join(dir, "default video.mp4")
		if err := os.WriteFile(video, []byte("test video"), 0600); err != nil {
			t.Fatal(err)
		}
		input := strings.Join([]string{"test-subprocess-token", "", configPath, envPath, "n", video, ""}, "\n") + "\n"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, setup)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(input)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup: %v\n%s", err, output)
		}
		if _, err := internal.LoadConfig(configPath); err != nil {
			t.Fatal(err)
		}
		assertMode(t, envPath, 0600)
		cmd = exec.CommandContext(ctx, app, "--dry-run")
		cmd.Dir = dir
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "PLEX_TOKEN=") && !strings.HasPrefix(entry, "PLEX_SERVER_URL=") && !strings.HasPrefix(entry, "PLEX_PREROLL_CFG=") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("main command rejected setup output: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), video) {
			t.Errorf("dry run omitted configured video: %s", output)
		}
	})
	t.Run("EOF exits nonzero", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, setup)
		cmd.Dir = t.TempDir()
		output, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatal("setup hung on EOF")
		}
		if err == nil || !strings.Contains(string(output), "EOF") {
			t.Fatalf("setup should report EOF with nonzero exit: %v\n%s", err, output)
		}
	})
}
