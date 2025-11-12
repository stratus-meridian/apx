package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// HandleStatus handles GET /status/{request_id}
func (m *Matcher) HandleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	requestID := vars["request_id"]

	if requestID == "" {
		m.logger.Error("request_id not provided in URL")
		http.Error(w, `{"error":"request_id is required"}`, http.StatusBadRequest)
		return
	}

	// Get status from store
	statusRecord, err := m.statusStore.Get(ctx, requestID)
	if err != nil {
		m.logger.Error("failed to get status",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		http.Error(w, `{"error":"status not found"}`, http.StatusNotFound)
		return
	}

	// Return status as JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(statusRecord); err != nil {
		m.logger.Error("failed to encode status response",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
	}
}
