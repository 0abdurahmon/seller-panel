package realtime

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// heartbeatEvery keeps idle streams open through proxies; nginx reads time out after 30 s without data.
const heartbeatEvery = 25 * time.Second

// Stream writes the events of sub as server-sent events until the client leaves, the
// subscription ends, or the token expires (zero means no expiry).
func Stream(w http.ResponseWriter, r *http.Request, sub *Subscription, expires time.Time) {
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		return
	}

	heartbeat := time.NewTicker(heartbeatEvery)
	defer heartbeat.Stop()
	var expired <-chan time.Time
	if !expires.IsZero() {
		timer := time.NewTimer(time.Until(expires))
		defer timer.Stop()
		expired = timer.C
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-expired:
			return
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
		case msg, ok := <-sub.Messages():
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Name, msg.Data); err != nil {
				return
			}
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
