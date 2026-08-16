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

type TelegramConfig struct {
	BotToken string
	ChatID   string
	BaseURL  string // Optional, defaults to https://api.telegram.org
}

type TelegramSender struct {
	endpoint string
	chatID   string
	client   *http.Client
}

func NewTelegramSender(config TelegramConfig, client *http.Client) (*TelegramSender, error) {
	token := strings.TrimSpace(config.BotToken)
	if token == "" {
		return nil, errors.New("alerts: telegram bot token is required")
	}
	chatID := strings.TrimSpace(config.ChatID)
	if chatID == "" {
		return nil, errors.New("alerts: telegram chat ID is required")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("alerts: telegram base URL must be an absolute http(s) URL")
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(baseURL, "/"), token)
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramSender{endpoint: endpoint, chatID: chatID, client: client}, nil
}

func (s *TelegramSender) Send(ctx context.Context, delivery Delivery) error {
	if !validSingleLineText(delivery.Title, 512) {
		return errors.New("alerts: telegram title must be single-line UTF-8 and at most 512 bytes")
	}

	icon := "ℹ️"
	switch delivery.Severity {
	case SeverityCritical:
		icon = "🚨"
	case SeverityWarning:
		icon = "⚠️"
	}

	text := fmt.Sprintf("%s *[%s]* %s\n\n%s", icon, escapeTelegramMarkdown(string(delivery.Severity)), escapeTelegramMarkdown(delivery.Title), escapeTelegramMarkdown(delivery.Message))
	payloadMap := map[string]any{
		"chat_id":    s.chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("alerts: marshal telegram payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("alerts: create telegram request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("alerts: send telegram request: %w", err)
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
	return fmt.Errorf("alerts: telegram returned %d: %s", response.StatusCode, message)
}

func escapeTelegramMarkdown(text string) string {
	chars := `\_*[]()~` + "`" + `>#+-=|{}.!`
	var sb strings.Builder
	for _, r := range text {
		if strings.ContainsRune(chars, r) {
			sb.WriteRune('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
