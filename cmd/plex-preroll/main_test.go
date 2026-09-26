package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bsmccain/plex-preroll/internal"
	"gopkg.in/yaml.v3"
)

func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	for _, key := range []string{"PLEX_TOKEN", "PLEX_SERVER_URL", "PLEX_PREROLL_CFG"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	return dir
}

func writeConfig(t *testing.T, path string, files []string) {
	t.Helper()
	data, err := yaml.Marshal(internal.Config{Default: files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRunUpdatesPlexAndDryRun(t *testing.T) {
	dir := isolate(t)
	video := filepath.Join(dir, "intro.mp4")
	if err := os.WriteFile(video, []byte("test video"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.yaml")
	writeConfig(t, cfg, []string{video})
	t.Setenv("PLEX_TOKEN", "test-token")
	t.Setenv("PLEX_PREROLL_CFG", cfg)
	type request struct{ method, path, token, files string }
	requests := make(chan request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- request{r.Method, r.URL.Path, r.Header.Get("X-Plex-Token"), r.URL.Query().Get("CinemaTrailersPrerollID")}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("PLEX_SERVER_URL", server.URL)
	var output bytes.Buffer
	if err := run([]string{"--dry-run"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), video) {
		t.Fatalf("missing video in dry run: %s", &output)
	}
	select {
	case r := <-requests:
		t.Fatalf("dry run made request: %+v", r)
	default:
	}
	if err := run(nil, &output, &output); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-requests:
		want := request{"PUT", "/:/prefs", "test-token", video}
		if got != want {
			t.Fatalf("request = %+v, want %+v", got, want)
		}
	default:
		t.Fatal("no update sent")
	}
}

func TestRunPreservesSettingsOnMissingFiles(t *testing.T) {
	dir := isolate(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer server.Close()
	t.Setenv("PLEX_TOKEN", "test-token")
	t.Setenv("PLEX_SERVER_URL", server.URL)
	cfg := filepath.Join(dir, "config.yaml")
	t.Setenv("PLEX_PREROLL_CFG", cfg)
	writeConfig(t, cfg, []string{filepath.Join(dir, "missing.mp4")})
	var output bytes.Buffer
	if err := run(nil, &output, &output); err == nil {
		t.Fatal("missing file must fail")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("sent %d requests on file error", got)
	}
	writeConfig(t, cfg, []string{})
	if err := run(nil, &output, &output); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("explicit empty config sent %d requests, want 1", got)
	}
}

func TestRunEnvironmentSearchAndTilde(t *testing.T) {
	dir := isolate(t)
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	userEnvDir := filepath.Join(userConfigDir, "plex-preroll")
	if err := os.MkdirAll(userEnvDir, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "prerolls.yaml")
	writeConfig(t, config, []string{})
	// The current-directory file wins over the invalid user configuration file.
	if err := os.WriteFile(filepath.Join(userEnvDir, ".env"), []byte("INVALID LINE !\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("PLEX_TOKEN=test-token\nPLEX_PREROLL_CFG=~/prerolls.yaml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"--dry-run"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Loaded .env from .env") {
		t.Fatalf("wrong dotenv search order: %s", &output)
	}
	if err := os.Remove(".env"); err != nil {
		t.Fatal(err)
	}
	// An existing malformed file must report an error, even with inherited env values.
	if err := run([]string{"--dry-run"}, &output, &output); err == nil || !strings.Contains(err.Error(), "load .env") {
		t.Fatalf("malformed dotenv error = %v", err)
	}
}

func TestRunDefaultConfigSearch(t *testing.T) {
	isolate(t)
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	writeConfig(t, filepath.Join(dir, "plex-preroll", "config.yaml"), []string{})
	t.Setenv("PLEX_TOKEN", "test-token")
	var output bytes.Buffer
	if err := run([]string{"--dry-run"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Using config file:") {
		t.Fatalf("default config not selected: %s", &output)
	}
}

func TestRunValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		token string
		want  string
	}{
		{"missing token", nil, "", "PLEX_TOKEN"},
		{"missing config", nil, "test-token", "no config file"},
		{"unknown option", []string{"--unknown"}, "", "flag provided but not defined"},
		{"unexpected argument", []string{"extra"}, "", "unexpected arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolate(t)
			if tc.token != "" {
				t.Setenv("PLEX_TOKEN", tc.token)
			}
			var output bytes.Buffer
			err := run(tc.args, &output, &output)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
	for _, arg := range []string{"--version", "--help"} {
		t.Run(arg, func(t *testing.T) {
			isolate(t)
			var output bytes.Buffer
			if err := run([]string{arg}, &output, &output); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "plex-preroll") {
				t.Fatalf("unexpected output: %s", &output)
			}
		})
	}
}

func TestCommandSubprocess(t *testing.T) {
	// Build the real command once, then verify its process exit status and flags.
	binary := filepath.Join(t.TempDir(), "plex-preroll")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	dir := isolate(t)
	writeConfig(t, filepath.Join(dir, "config.yaml"), []string{})
	for _, tc := range []struct {
		name  string
		args  []string
		token string
		fail  bool
		want  string
	}{
		{"version", []string{"--version"}, "", false, "plex-preroll dev"},
		{"missing token", nil, "", true, "PLEX_TOKEN"},
		{"dry run", []string{"--dry-run"}, "test-token", false, "Dry run, skipping update"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, tc.args...)
			cmd.Env = append(os.Environ(), "PLEX_TOKEN="+tc.token, "PLEX_PREROLL_CFG="+filepath.Join(dir, "config.yaml"))
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			if (err != nil) != tc.fail {
				t.Fatalf("exit: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("output %q missing %q", output, tc.want)
			}
		})
	}
}
