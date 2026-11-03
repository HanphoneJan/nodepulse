package localapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/store"
)

func (s *Server) publishEvent(ev store.Event) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	for ch := range s.eventSubs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// handleEvents streams daemon events as Server-Sent Events so the desktop UI
// can show notifications without polling the whole history.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan store.Event, 8)
	s.eventMu.Lock()
	s.eventSubs[ch] = struct{}{}
	s.eventMu.Unlock()
	defer func() {
		s.eventMu.Lock()
		delete(s.eventSubs, ch)
		s.eventMu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
