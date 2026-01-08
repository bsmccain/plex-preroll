package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdatePreroll(t *testing.T) {
	tests := []struct {
		name           string
		videos         []string
		serverStatus   int
		expectError    bool
		expectedQuery  string
		expectedToken  string
	}{
		{
			name:          "single video",
			videos:        []string{"/path/to/video.mp4"},
			serverStatus:  http.StatusOK,
			expectError:   false,
			expectedQuery: "/path/to/video.mp4",
			expectedToken: "test-token",
		},
		{
			name:          "multiple videos",
			videos:        []string{"/path/to/video1.mp4", "/path/to/video2.mkv", "/path/to/video3.avi"},
			serverStatus:  http.StatusOK,
			expectError:   false,
			expectedQuery: "/path/to/video1.mp4;/path/to/video2.mkv;/path/to/video3.avi",
			expectedToken: "test-token",
		},
		{
			name:          "empty video list",
			videos:        []string{},
			serverStatus:  http.StatusOK,
			expectError:   false,
			expectedQuery: "",
			expectedToken: "test-token",
		},
		{
			name:          "server returns 401 unauthorized",
			videos:        []string{"/path/to/video.mp4"},
			serverStatus:  http.StatusUnauthorized,
			expectError:   true,
			expectedQuery: "/path/to/video.mp4",
			expectedToken: "test-token",
		},
		{
			name:          "server returns 500 internal error",
			videos:        []string{"/path/to/video.mp4"},
			serverStatus:  http.StatusInternalServerError,
			expectError:   true,
			expectedQuery: "/path/to/video.mp4",
			expectedToken: "test-token",
		},
		{
			name:          "server returns 404 not found",
			videos:        []string{"/path/to/video.mp4"},
			serverStatus:  http.StatusNotFound,
			expectError:   true,
			expectedQuery: "/path/to/video.mp4",
			expectedToken: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMethod string
			var receivedPath string
			var receivedQuery string
			var receivedToken string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				receivedPath = r.URL.Path
				receivedQuery = r.URL.Query().Get("CinemaTrailersPrerollID")
				receivedToken = r.Header.Get("X-Plex-Token")
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			err := UpdatePreroll(server.URL, tt.expectedToken, tt.videos)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if receivedMethod != "PUT" {
				t.Errorf("expected PUT method, got %s", receivedMethod)
			}

			if receivedPath != "/:/prefs" {
				t.Errorf("expected path /:/prefs, got %s", receivedPath)
			}

			if receivedQuery != tt.expectedQuery {
				t.Errorf("expected query %q, got %q", tt.expectedQuery, receivedQuery)
			}

			if receivedToken != tt.expectedToken {
				t.Errorf("expected token %q, got %q", tt.expectedToken, receivedToken)
			}
		})
	}
}

func TestUpdatePrerollInvalidURL(t *testing.T) {
	err := UpdatePreroll("://invalid-url", "token", []string{"/video.mp4"})
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse server URL") {
		t.Errorf("expected error to contain 'failed to parse server URL', got: %v", err)
	}
}

func TestUpdatePrerollConnectionRefused(t *testing.T) {
	// Use a port that is not listening
	err := UpdatePreroll("http://127.0.0.1:1", "token", []string{"/video.mp4"})
	if err == nil {
		t.Error("expected error for connection refused, got nil")
	}
	if !strings.Contains(err.Error(), "failed to send request to Plex") {
		t.Errorf("expected error to contain 'failed to send request to Plex', got: %v", err)
	}
}

func TestUpdatePrerollURLWithTrailingSlash(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with trailing slash in URL
	err := UpdatePreroll(server.URL+"/", "token", []string{"/video.mp4"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// JoinPath should handle trailing slashes correctly
	if receivedPath != "/:/prefs" {
		t.Errorf("expected path /:/prefs, got %s", receivedPath)
	}
}

func TestUpdatePrerollSpecialCharactersInPath(t *testing.T) {
	var receivedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("CinemaTrailersPrerollID")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Video paths with special characters
	videos := []string{
		"/path/to/video with spaces.mp4",
		"/path/to/video&special.mkv",
		"/path/to/видео.mp4", // Unicode
	}

	err := UpdatePreroll(server.URL, "token", videos)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := strings.Join(videos, ";")
	if receivedQuery != expected {
		t.Errorf("expected query %q, got %q", expected, receivedQuery)
	}
}
