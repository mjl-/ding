package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SSE is a real-time streaming updates API using server-sent event, available at /events.
// Query string parameter "password" is required.
// You'll receive the following events with a HTTP GET request to `/events`, encoded as JSON:
// - `repo`, repository was updated or created
// - `removeRepo`, repository was removed
// - `build`, build was updated or created
// - `removeBuild`, build was removed
// - `output`, new lines of output from a command for an active build
//
// These types are described below, with an _event_-prefix. E.g. type _EventRepo_ describes the `repo` event.
type SSE struct {
}

// ExampleSSE is a no-op.
// This function only serves to include documentation for the server-sent event types.
func (SSE) ExampleSSE() (repo EventRepo, removeRepo EventRemoveRepo, build EventBuild, removeBuild EventRemoveBuild, output EventOutput) {
	return
}

type eventWorker struct {
	events chan []byte
}

func serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("internal error: ResponseWriter not a http.Flusher")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	password := r.FormValue("password")
	if password != config.Password {
		http.Error(w, "bad auth", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, err := w.Write([]byte(": keepalive\n\n"))
	if err != nil {
		return
	}
	flusher.Flush()

	ew := &eventWorker{make(chan []byte, 48)}
	register <- ew
	defer func() {
		unregister <- ew
	}()

	ctx := r.Context()
	for {
		select {
		case msg := <-ew.events:
			_, err = w.Write(msg)
			flusher.Flush()
			if err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

var (
	register   = make(chan *eventWorker, 1)
	unregister = make(chan *eventWorker)
	events     = make(chan eventStringer, 10)
)

func init() {
	go eventMux()
}

func eventMux() {
	go func() {
		for {
			time.Sleep(120 * time.Second)
			events <- nil
		}
	}()

	workers := []*eventWorker{}
	for {
		select {
		case ew := <-register:
			workers = append(workers, ew)
		case ew := <-unregister:
			nworkers := []*eventWorker{}
			for _, x := range workers {
				if x != ew {
					nworkers = append(nworkers, x)
				}
			}
			workers = nworkers
		case ev := <-events:
			var buf []byte
			if ev == nil {
				buf = []byte(": keepalive\n\n")
			} else {
				event, evbuf, err := ev.eventString()
				if err != nil {
					slog.Error("sse: marshalling event", "err", err)
					continue
				}
				buf = []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, evbuf))
			}
			for _, w := range workers {
				select {
				case w.events <- buf:
				default:
					slog.Debug("sse: dropping event, client cannot keep up...")
				}
			}
		}
	}
}
