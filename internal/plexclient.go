package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func UpdatePreroll(serverURL, token string, videos []string) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
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

	client := &http.Client{}
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
