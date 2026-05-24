package indexer

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func NewHTTPHandler(broadcaster *Broadcaster) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		id, events := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(id)

		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				encoded, err := json.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\n", event.Type)
				fmt.Fprintf(w, "data: %s\n\n", encoded)
				flusher.Flush()
			}
		}
	})
	return mux
}
