// Package telegram talks to the Telegram Bot API over plain HTTP.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	pollTimeout    = 30 // seconds, long polling
	pollRetryDelay = 5 * time.Second
)

// Client calls the Bot API for one bot.
type Client struct {
	token    string
	username string
	baseURL  string
	http     *http.Client
}

// Update is the part of an incoming message the bot needs.
type Update struct {
	ID     int64
	ChatID int64
	Text   string
}

// NewClient returns a client for the bot. baseURL is usually https://api.telegram.org.
func NewClient(token, username, baseURL string) *Client {
	return &Client{
		token:    token,
		username: username,
		baseURL:  baseURL,
		http:     &http.Client{Timeout: (pollTimeout + 15) * time.Second},
	}
}

// BotUsername is the public name of the bot, without the @.
func (c *Client) BotUsername() string { return c.username }

// SendMessage posts text to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]any{"chat_id": chatID, "text": text}
	return c.call(ctx, "sendMessage", body, nil)
}

// getUpdates returns messages after offset, waiting up to pollTimeout seconds.
func (c *Client) getUpdates(ctx context.Context, offset int64) ([]Update, error) {
	body := map[string]any{
		"offset":          offset,
		"timeout":         pollTimeout,
		"allowed_updates": []string{"message"},
	}
	var raw []struct {
		UpdateID int64 `json:"update_id"`
		Message  *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
	}
	if err := c.call(ctx, "getUpdates", body, &raw); err != nil {
		return nil, err
	}
	updates := make([]Update, 0, len(raw))
	for _, u := range raw {
		up := Update{ID: u.UpdateID}
		if u.Message != nil {
			up.ChatID = u.Message.Chat.ID
			up.Text = u.Message.Text
		}
		updates = append(updates, up)
	}
	return updates, nil
}

// Poll receives messages until ctx ends and passes each text message to handle.
// Handler errors are logged and do not stop polling.
func (c *Client) Poll(ctx context.Context, logger *slog.Logger, handle func(ctx context.Context, chatID int64, text string) error) {
	var offset int64
	for ctx.Err() == nil {
		updates, err := c.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("telegram polling failed, retrying", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollRetryDelay):
			}
			continue
		}
		for _, u := range updates {
			offset = u.ID + 1
			if u.Text == "" {
				continue
			}
			if err := handle(ctx, u.ChatID, u.Text); err != nil {
				logger.Error("telegram update not handled", "error", err)
			}
		}
	}
}

// call posts a Bot API method and decodes its result into out when out is not nil.
// The bot token is part of the request URL, so transport errors are returned
// without the URL.
func (c *Client) call(ctx context.Context, method string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram %s: encode body: %w", method, err)
	}
	endpoint := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram %s: build request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer res.Body.Close()

	var env struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&env); err != nil {
		return fmt.Errorf("telegram %s: status %d, unreadable response", method, res.StatusCode)
	}
	if !env.OK {
		return fmt.Errorf("telegram %s: status %d: %s", method, res.StatusCode, env.Description)
	}
	if out != nil {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return fmt.Errorf("telegram %s: decode result: %w", method, err)
		}
	}
	return nil
}
