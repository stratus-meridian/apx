package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apx/router/pkg/status"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// SSEStreamer handles Server-Sent Events streaming
type SSEStreamer struct {
	statusStore status.Store
	logger      *zap.Logger
}

// NewSSEStreamer creates a new SSE streamer
func NewSSEStreamer(statusStore status.Store, logger *zap.Logger) *SSEStreamer {
	return &SSEStreamer{
		statusStore: statusStore,
		logger:      logger,
	}
}

// SSEEvent represents a single SSE event
type SSEEvent struct {
	ID    string `json:"-"`        // Event ID (for resume)
	Type  string `json:"type"`     // Event type
	Data  any    `json:"data"`     // Event data
	Retry int    `json:"-"`        // Retry timeout in ms
}

// HandleStream handles GET /stream/{request_id}
// Implements Server-Sent Events (SSE) for real-time status updates
func (s *SSEStreamer) HandleStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	requestID := vars["request_id"]

	if requestID == "" {
		s.logger.Error("request_id not provided in URL")
		http.Error(w, `{"error":"request_id is required"}`, http.StatusBadRequest)
		return
	}

	// Check if request exists
	statusRecord, err := s.statusStore.Get(ctx, requestID)
	if err != nil {
		s.logger.Error("failed to get status",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		http.Error(w, `{"error":"status not found"}`, http.StatusNotFound)
		return
	}

	// Get Last-Event-ID header for resume support
	lastEventID := r.Header.Get("Last-Event-ID")
	lastSequence := 0
	if lastEventID != "" {
		if seq, err := strconv.Atoi(lastEventID); err == nil {
			lastSequence = seq
			s.logger.Info("resuming stream",
				zap.String("request_id", requestID),
				zap.Int("last_sequence", lastSequence),
			)
		}
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering in nginx

	// Create a flusher to send data immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logger.Error("streaming not supported")
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	s.logger.Info("starting SSE stream",
		zap.String("request_id", requestID),
		zap.String("tenant_id", statusRecord.TenantID),
	)

	// Send initial status event
	sequence := lastSequence + 1
	s.sendEvent(w, flusher, &SSEEvent{
		ID:    fmt.Sprintf("%d", sequence),
		Type:  "status",
		Data:  statusRecord,
		Retry: 1000, // Retry after 1 second
	})
	sequence++

	// Poll for status updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(5 * time.Minute) // 5 minute timeout
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("client disconnected",
				zap.String("request_id", requestID),
			)
			return

		case <-timeout.C:
			s.logger.Info("stream timeout",
				zap.String("request_id", requestID),
			)
			s.sendEvent(w, flusher, &SSEEvent{
				ID:   fmt.Sprintf("%d", sequence),
				Type: "timeout",
				Data: map[string]string{
					"message": "stream timeout - reconnect to continue",
				},
			})
			return

		case <-ticker.C:
			// Get updated status
			updatedStatus, err := s.statusStore.Get(ctx, requestID)
			if err != nil {
				s.logger.Error("failed to get status update",
					zap.String("request_id", requestID),
					zap.Error(err),
				)
				continue
			}

			// Send status update
			s.sendEvent(w, flusher, &SSEEvent{
				ID:   fmt.Sprintf("%d", sequence),
				Type: "status",
				Data: updatedStatus,
			})
			sequence++

			// If complete or failed, send final event and close
			if updatedStatus.Status == status.StatusComplete || updatedStatus.Status == status.StatusFailed {
				s.logger.Info("request completed, closing stream",
					zap.String("request_id", requestID),
					zap.String("final_status", string(updatedStatus.Status)),
				)

				// Send completion event
				s.sendEvent(w, flusher, &SSEEvent{
					ID:   fmt.Sprintf("%d", sequence),
					Type: "complete",
					Data: map[string]any{
						"status":       updatedStatus.Status,
						"result":       updatedStatus.Result,
						"error":        updatedStatus.Error,
						"completed_at": updatedStatus.CompletedAt,
					},
				})

				return
			}
		}
	}
}

// sendEvent sends an SSE event to the client
func (s *SSEStreamer) sendEvent(w http.ResponseWriter, flusher http.Flusher, event *SSEEvent) {
	// Write event ID (for resume support)
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}

	// Write event type
	if event.Type != "" {
		fmt.Fprintf(w, "event: %s\n", event.Type)
	}

	// Write retry timeout
	if event.Retry > 0 {
		fmt.Fprintf(w, "retry: %d\n", event.Retry)
	}

	// Write data as JSON
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		s.logger.Error("failed to marshal event data", zap.Error(err))
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", string(dataJSON))

	// Flush to send immediately
	flusher.Flush()
}
