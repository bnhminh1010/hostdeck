package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DiscordConfig struct {
	WebhookURL string
}

type DiscordSender struct {
	endpoint string
	client   *http.Client
}

func NewDiscordSender(config DiscordConfig, client *http.Client) (*DiscordSender, error) {
	rawURL := strings.TrimSpace(config.WebhookURL)
	if rawURL == "" {
		return nil, errors.New("alerts: discord webhook URL is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("alerts: discord webhook URL must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return nil, errors.New("alerts: discord webhook URL must not contain credentials")
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &DiscordSender{endpoint: parsed.String(), client: client}, nil
}

func (s *DiscordSender) Send(ctx context.Context, delivery Delivery) error {
	if !validSingleLineText(delivery.Title, 512) {
		return errors.New("alerts: discord title must be single-line UTF-8 and at most 512 bytes")
	}

	color := 3447003 // Neutral Blue (#3498db)
	switch delivery.Severity {
	case SeverityCritical:
		color = 14689052 // Vermilion Red (#e0231c)
	case SeverityWarning:
		color = 16098827 // Amber Yellow (#f59e0b)
	}

	embed := map[string]any{
		"title":       delivery.Title,
		"description": delivery.Message,
		"color":       color,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"footer": map[string]string{
			"text": "HostDeck Operations Console",
		},
	}

	payloadMap := map[string]any{
		"embeds": []any{embed},
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("alerts: marshal discord payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("alerts: create discord request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("alerts: send discord request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("alerts: discord returned %d: %s", response.StatusCode, message)
}
