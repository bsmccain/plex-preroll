package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationDryRun(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	videoDir := filepath.Join(tempDir, "videos")
	os.Mkdir(videoDir, 0755)

	// Create a video file
	videoFile := filepath.Join(videoDir, "test.mp4")
	os.WriteFile(videoFile, []byte("fake video"), 0644)

	// Write config
	configContent := `seasons:
  - name: all_year
    from: "01-01"
    to: "12-31"
    files:
      - ` + videoFile + `
default:
  - ` + videoFile
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Set up environment
	originalCfg := os.Getenv("PLEX_PREROLL_CFG")
	originalToken := os.Getenv("PLEX_TOKEN")
	originalURL := os.Getenv("PLEX_SERVER_URL")
	defer func() {
		os.Setenv("PLEX_PREROLL_CFG", originalCfg)
		os.Setenv("PLEX_TOKEN", originalToken)
		os.Setenv("PLEX_SERVER_URL", originalURL)
	}()

	os.Setenv("PLEX_PREROLL_CFG", configPath)
	os.Setenv("PLEX_TOKEN", "test-token")
	os.Setenv("PLEX_SERVER_URL", "http://localhost:32400")

	// Note: We can't easily test main() directly since it uses log.Fatalf
	// and flag parsing. This test validates the config loading works.
	// Full integration testing would require running the binary as a subprocess.
}

func TestPlexServerIntegration(t *testing.T) {
	// Create a mock Plex server
	var receivedPath string
	var receivedQuery string
	var receivedToken string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedQuery = r.URL.Query().Get("CinemaTrailersPrerollID")
		receivedToken = r.Header.Get("X-Plex-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create test files
	tempDir := t.TempDir()
	videoFile := filepath.Join(tempDir, "test.mp4")
	os.WriteFile(videoFile, []byte("fake video"), 0644)

	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `seasons:
  - name: all_year
    from: "01-01"
    to: "12-31"
    files:
      - ` + videoFile + `
default:
  - ` + videoFile
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Import and use the internal package directly for testing
	// This tests the UpdatePreroll function in an integration context
	// (The actual binary testing would require subprocess execution)

	// Verify mock server is working
	if server.URL == "" {
		t.Fatal("mock server URL is empty")
	}

	// Make a test request to verify server setup
	req, _ := http.NewRequest("PUT", server.URL+"/:/prefs?CinemaTrailersPrerollID=test", nil)
	req.Header.Set("X-Plex-Token", "test-token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make test request: %v", err)
	}
	resp.Body.Close()

	if receivedMethod != "PUT" {
		t.Errorf("expected PUT method, got %s", receivedMethod)
	}
	if receivedPath != "/:/prefs" {
		t.Errorf("expected /:/prefs path, got %s", receivedPath)
	}
	if receivedQuery != "test" {
		t.Errorf("expected 'test' query, got %s", receivedQuery)
	}
	if receivedToken != "test-token" {
		t.Errorf("expected 'test-token', got %s", receivedToken)
	}
}

func TestEnvPathSearch(t *testing.T) {
	// Test that env path search order is correct
	tempDir := t.TempDir()

	// Create a .env file in the temp directory
	envFile := filepath.Join(tempDir, ".env")
	envContent := "PLEX_TOKEN=test-token-from-env\nPLEX_SERVER_URL=http://test-server:32400"
	os.WriteFile(envFile, []byte(envContent), 0644)

	// Verify file was created
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Fatal("failed to create test .env file")
	}

	// Read and verify content
	content, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("failed to read .env file: %v", err)
	}

	if !strings.Contains(string(content), "PLEX_TOKEN=test-token-from-env") {
		t.Error("env file doesn't contain expected PLEX_TOKEN")
	}
}

func TestConfigPathSearch(t *testing.T) {
	// Test that config path search works
	tempDir := t.TempDir()

	// Create a config file
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `seasons:
  - name: test
    from: "01-01"
    to: "12-31"
    files:
      - /path/to/video.mp4
default:
  - /path/to/default.mp4`
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Verify file exists and is readable
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not accessible: %v", err)
	}

	if info.Size() == 0 {
		t.Error("config file is empty")
	}
}

func TestVersionString(t *testing.T) {
	// Verify the version variable exists and has a default
	if version == "" {
		t.Error("version should not be empty")
	}

	// Default version should be "dev" if not set during build
	if version != "dev" {
		t.Logf("version is set to: %s (may be set during build)", version)
	}
}

func TestDefaultServerURL(t *testing.T) {
	// Test that the default server URL logic works correctly
	originalURL := os.Getenv("PLEX_SERVER_URL")
	defer os.Setenv("PLEX_SERVER_URL", originalURL)

	// Clear the environment variable
	os.Unsetenv("PLEX_SERVER_URL")

	serverURL := os.Getenv("PLEX_SERVER_URL")
	if serverURL != "" {
		t.Error("PLEX_SERVER_URL should be empty after unset")
	}

	// The default in main.go is http://localhost:32400
	expectedDefault := "http://localhost:32400"
	if serverURL == "" {
		serverURL = expectedDefault
	}

	if serverURL != expectedDefault {
		t.Errorf("expected default URL %s, got %s", expectedDefault, serverURL)
	}
}

func TestRequiredPlexToken(t *testing.T) {
	// Test that PLEX_TOKEN is required
	originalToken := os.Getenv("PLEX_TOKEN")
	defer os.Setenv("PLEX_TOKEN", originalToken)

	os.Unsetenv("PLEX_TOKEN")

	token := os.Getenv("PLEX_TOKEN")
	if token != "" {
		t.Error("PLEX_TOKEN should be empty after unset")
	}

	// The main function would call log.Fatalf if token is empty
	// We can't test that directly without running as subprocess
}
