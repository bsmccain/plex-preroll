package internal

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestUpdatePreroll(t *testing.T) {
	tests := []struct {
		name          string
		videos        []string
		serverStatus  int
		expectError   bool
		expectedQuery string
		expectedToken string
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

func TestUpdatePrerollRejectsNonHTTPURLs(t *testing.T) {
	for _, serverURL := range []string{
		"", "localhost:32400", "/plex", "ftp://localhost", "http:///plex", "http://:32400",
		"http://user:password@localhost", "http://localhost/#fragment",
	} {
		t.Run(serverURL, func(t *testing.T) {
			if err := UpdatePreroll(serverURL, "token", nil); err == nil {
				t.Fatal("expected invalid URL to fail")
			}
		})
	}
}

func TestUpdatePrerollRejectsRedirects(t *testing.T) {
	for _, status := range []int{
		http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var targetRequests atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targetRequests.Add(1)
				if r.Header.Get("X-Plex-Token") != "" {
					t.Error("Plex token reached the redirect target")
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer target.Close()
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL, status)
			}))
			defer source.Close()

			if err := UpdatePreroll(source.URL, "secret-token", []string{"/video.mp4"}); err == nil {
				t.Fatal("expected redirect to fail")
			}
			if got := targetRequests.Load(); got != 0 {
				t.Fatalf("redirect target received %d requests, want zero", got)
			}
		})
	}
}

func TestUpdatePrerollTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	// Exercise the same request path with a short deadline to avoid waiting for
	// the production 30-second timeout during every test run.
	err := updatePreroll(server.URL, "token", nil, 25*time.Millisecond)
	var networkError net.Error
	if !errors.As(err, &networkError) || !networkError.Timeout() {
		t.Fatalf("expected request timeout, got %v", err)
	}
}
