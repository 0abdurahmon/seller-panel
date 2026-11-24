package telegram

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testToken = "123:SECRET-TOKEN"

// fakeBotAPI serves the two Bot API methods the client uses and records requests.
func fakeBotAPI(t *testing.T, updates []string) (*httptest.Server, *[]map[string]any, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var calls []map[string]any
	served := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/bot"+testToken+"/") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		mu.Lock()
		calls = append(calls, map[string]any{"method": strings.TrimPrefix(r.URL.Path, "/bot"+testToken+"/"), "body": req})
		mu.Unlock()

		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			if served {
				// Keep the poll loop idle after the scripted updates.
				<-r.Context().Done()
				return
			}
			served = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true,"result":[`+strings.Join(updates, ",")+`]}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"ok":false,"description":"Not Found"}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &mu
}

func TestSendMessagePostsToChat(t *testing.T) {
	srv, calls, mu := fakeBotAPI(t, nil)
	c := NewClient(testToken, "panel_bot", srv.URL)
	if err := c.SendMessage(context.Background(), 42, "hello"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 1 || (*calls)[0]["method"] != "sendMessage" {
		t.Fatalf("calls = %+v", *calls)
	}
	body := (*calls)[0]["body"].(map[string]any)
	if body["chat_id"] != float64(42) || body["text"] != "hello" {
		t.Fatalf("body = %+v", body)
	}
}

func TestErrorsNeverContainTheToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"ok":false,"description":"Unauthorized"}`)
	}))
	defer srv.Close()
	c := NewClient(testToken, "panel_bot", srv.URL)
	err := c.SendMessage(context.Background(), 1, "x")
	if err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), testToken) {
		t.Fatal("error leaks the bot token")
	}

	dead := NewClient(testToken, "panel_bot", "http://127.0.0.1:1")
	err = dead.SendMessage(context.Background(), 1, "x")
	if err == nil || strings.Contains(err.Error(), testToken) {
		t.Fatalf("transport error = %v", err)
	}
}

func TestPollPassesTextMessagesAndAdvancesOffset(t *testing.T) {
	updates := []string{
		`{"update_id":10,"message":{"chat":{"id":7},"text":"/start abc"}}`,
		`{"update_id":11,"edited_message":{"chat":{"id":7}}}`,
	}
	srv, calls, mu := fakeBotAPI(t, updates)
	c := NewClient(testToken, "panel_bot", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan string, 1)
	go c.Poll(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), func(_ context.Context, chatID int64, text string) error {
		got <- text
		cancel()
		return nil
	})
	if text := <-got; text != "/start abc" {
		t.Fatalf("handled %q", text)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) == 0 {
		t.Fatal("no getUpdates call")
	}
	if first := (*calls)[0]["body"].(map[string]any); first["offset"] != float64(0) || first["timeout"] != float64(pollTimeout) {
		t.Fatalf("first poll = %+v", first)
	}
}
