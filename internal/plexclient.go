package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const plexRequestTimeout = 30 * time.Second

func UpdatePreroll(serverURL, token string, videos []string) error {
	return updatePreroll(serverURL, token, videos, plexRequestTimeout)
}

func updatePreroll(serverURL, token string, videos []string, timeout time.Duration) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" {
		return fmt.Errorf("server URL must be an absolute HTTP or HTTPS URL")
	}
	if parsedURL.User != nil || parsedURL.Fragment != "" {
		return fmt.Errorf("server URL must not contain user credentials or a fragment")
	}

	parsedURL = parsedURL.JoinPath("/:/prefs")

	req, err := http.NewRequest("PUT", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("CinemaTrailersPrerollID", strings.Join(videos, ";"))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("X-Plex-Token", token)

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Custom authentication headers can be forwarded to a redirect target.
			// Return the redirect response without sending another request.
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("failed to send request to Plex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update preroll: Plex API returned status %s", resp.Status)
	}

	return nil
}
