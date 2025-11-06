package main

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

type outputEvent struct {
	Type     string                 `json:"type"`
	Progress *progressPayload       `json:"progress,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

type progressPayload struct {
	Current        int     `json:"current"`
	Total          int     `json:"total"`
	Percent        float64 `json:"percent"`
	Message        string  `json:"message"`
	PagesCompleted *int    `json:"pages_completed,omitempty"`
	PagesTotal     *int    `json:"pages_total,omitempty"`
}

type eventWriter struct {
	enc *json.Encoder
	w   *bufio.Writer
	mu  sync.Mutex
}

func newEventWriter(writer io.Writer) *eventWriter {
	buf := bufio.NewWriter(writer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return &eventWriter{enc: enc, w: buf}
}

func (e *eventWriter) flush() {
	_ = e.w.Flush()
}

func (e *eventWriter) Progress(payload progressPayload) {
	total := payload.Total
	current := payload.Current
	percent := payload.Percent
	if percent <= 0.0 && total > 0 {
		percent = float64(current) / float64(total) * 100.0
	}
	if percent > 100.0 {
		percent = 100.0
	}
	payload.Percent = percent
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type:     "progress",
		Progress: &payload,
	})
	e.flush()
}

func (e *eventWriter) Result(payload map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type:    "result",
		Payload: payload,
	})
	e.flush()
}

func (e *eventWriter) Error(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type:  "error",
		Error: err.Error(),
	})
	e.flush()
}

