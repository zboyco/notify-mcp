package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zboyco/notify-mcp/internal/config"
)

// SendMessage posts a message to the configured Telegram chat.
func SendMessage(ctx context.Context, cfg config.TelegramConfig, message string) error {
	base := strings.TrimRight(cfg.APIBaseURL, "/")
	url := fmt.Sprintf("%s/bot%s/sendMessage", base, cfg.Token)

	payload := map[string]any{
		"chat_id": cfg.ChatID,
		"text":    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram responded with %s", resp.Status)
	}
	return nil
}
