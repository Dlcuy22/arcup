// Package webhook provides notification mechanisms for the arcup lifecycle.
//
// Key Components:
//   - DiscordNotifier: Implements Notifier for Discord webhooks
//   - NewDiscord(): Returns a new DiscordNotifier
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordNotifier struct {
	WebhookURL string
	client     *http.Client
}

func NewDiscord(url string) *DiscordNotifier {
	if url == "" {
		return nil
	}
	return &DiscordNotifier{
		WebhookURL: url,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// formatSize converts bytes to a human-readable string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func (d *DiscordNotifier) Notify(event Event) error {
	payload := d.buildPayload(event)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequest("POST", d.WebhookURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

func (d *DiscordNotifier) buildPayload(event Event) map[string]interface{} {
	var color int
	var title string
	var fields []map[string]interface{}

	switch event.Type {
	case EventStarted:
		color = 3447003 // Blue
		title = "[ARCUP] Backup Job Started"
	case EventArchived:
		color = 16776960 // Yellow
		title = "[ARCUP] Archive Created"
		fields = append(fields,
			map[string]interface{}{"name": "Archive", "value": event.Archive, "inline": false},
			map[string]interface{}{"name": "Time Taken", "value": event.Duration.String(), "inline": true},
		)
	case EventUploaded:
		color = 5763719 // Green
		title = "[ARCUP] Backup Uploaded Successfully"
		fields = append(fields,
			map[string]interface{}{"name": "Archive", "value": event.Archive, "inline": false},
			map[string]interface{}{"name": "Size", "value": formatSize(event.Size), "inline": true},
			map[string]interface{}{"name": "SHA256", "value": fmt.Sprintf("`%s`", event.Hash), "inline": false},
			map[string]interface{}{"name": "Upload Time", "value": event.Duration.String(), "inline": true},
		)
	case EventFailed:
		color = 15548997 // Red
		title = "[ARCUP] Backup Job Failed"
		fields = append(fields,
			map[string]interface{}{"name": "Error", "value": fmt.Sprintf("```\n%s\n```", event.Error.Error()), "inline": false},
		)
	}

	embed := map[string]interface{}{
		"title":     title,
		"color":     color,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"footer": map[string]interface{}{
			"text": "Powered by [arcup](<https://github.com/dlcuy22/arcup>)",
		},
	}
	if len(fields) > 0 {
		embed["fields"] = fields
	}

	return map[string]interface{}{
		"username": "Arcup Notifier",
		"embeds":   []map[string]interface{}{embed},
	}
}
