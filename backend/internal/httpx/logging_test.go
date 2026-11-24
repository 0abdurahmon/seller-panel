package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingWritesOneLinePerRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping?secret=x", nil))

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON: %v: %q", err, buf.String())
	}
	if line["method"] != "GET" || line["path"] != "/ping" || line["status"] != float64(http.StatusTeapot) {
		t.Fatalf("log line = %v", line)
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Fatal("duration_ms missing")
	}
}
