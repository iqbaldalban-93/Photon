package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"app/libweb/configx"
	"app/libweb/events"
)

type AccountInfo struct {
	Account string `json:"account"`
	Epoch   uint64 `json:"epoch"`
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.Events == nil {
		jsonError(w, http.StatusNotFound, "events unavailable")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ch, cancel := s.Events.Subscribe()
	defer cancel()

	hello := AccountInfo{Account: configx.Get().Photos.Selected}
	if s.AccountEpoch != nil {
		hello.Epoch = s.AccountEpoch()
	}
	writeSSE(w, events.Event{Name: "hello", Data: mustJSON(hello)})
	flusher.Flush()

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			writeSSE(w, ev)
			flusher.Flush()
		case <-keepalive.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, ev events.Event) {
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Name, ev.Data)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}
