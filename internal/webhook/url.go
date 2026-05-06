// Package webhook provides notification mechanisms for the arcup lifecycle.
//
// Key Components:
//   - URLNotifier: Implements Notifier for custom URL webhooks
//   - NewURL(): Returns a new URLNotifier
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type URLNotifier struct {
	WebhookURL string
	client     *http.Client
}

func NewURL(url string) *URLNotifier {
	if url == "" {
		return nil
	}
	return &URLNotifier{
		WebhookURL: url,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (u *URLNotifier) Notify(event Event) error {
	payload := u.buildPayload(event)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal url payload: %w", err)
	}

	req, err := http.NewRequest("POST", u.WebhookURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Arcup-Webhook/1.0")

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("custom webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("custom webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

func (u *URLNotifier) buildPayload(event Event) map[string]interface{} {
	payload := map[string]interface{}{
		"event":     string(event.Type),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	switch event.Type {
	case EventArchived:
		payload["archive"] = event.Archive
		payload["duration_seconds"] = event.Duration.Seconds()
	case EventUploaded:
		payload["archive"] = event.Archive
		payload["size_bytes"] = event.Size
		payload["hash_sha256"] = event.Hash
		payload["duration_seconds"] = event.Duration.Seconds()
	case EventFailed:
		payload["error"] = event.Error.Error()
	}

	return payload
}
