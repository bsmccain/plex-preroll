package internal

import (
	"fmt"
	"net/http"
	"strings"
)

func UpdatePreroll(serverURL, token string, videos []string) error {
	serverURL = strings.TrimRight(serverURL, "/")
	req, err := http.NewRequest("PUT", serverURL+"/:/prefs", nil)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update preroll: %s", resp.Status)
	}

	return nil
}
