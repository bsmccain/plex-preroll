package internal

import (
	"fmt"
	"net/http"
	"net/url" // Added
	"strings"
)

func UpdatePreroll(serverURL, token string, videos []string) error {
	// Parse the base server URL
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
	}

	// Use JoinPath to correctly append the API path.
	// This handles cases where serverURL might or might not have a trailing slash,
	// or might have an existing path.
	// For example, if serverURL is "http://localhost:32400/plex",
	// JoinPath will correctly produce "http://localhost:32400/plex/:/prefs".
	// If serverURL is "http://localhost:32400",
	// it will produce "http://localhost:32400/:/prefs".
	parsedURL = parsedURL.JoinPath("/:/prefs")

	// Create the request with the new, correctly formed URL string
	req, err := http.NewRequest("PUT", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get query parameters object from the *request's* URL, not the base parsedURL
	q := req.URL.Query()
	q.Set("CinemaTrailersPrerollID", strings.Join(videos, ";"))
	req.URL.RawQuery = q.Encode() // Set the encoded query string back to the request's URL

	req.Header.Set("X-Plex-Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Wrap the error for more context if it's an HTTP error
		return fmt.Errorf("failed to send request to Plex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// It might be useful to include more info from the response, e.g., body, if permission issues occur
		return fmt.Errorf("failed to update preroll: Plex API returned status %s", resp.Status)
	}

	return nil
}
