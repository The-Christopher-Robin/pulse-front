package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/analytics"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/catalog"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/experiments"
)

type handlers struct {
	catalog     *catalog.Service
	experiments *experiments.Service
	writer      *analytics.Writer
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) listProducts(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	products, err := h.catalog.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

func (h *handlers) getProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.catalog.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handlers) listExperiments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"experiments": h.experiments.Registry().All(),
	})
}

func (h *handlers) getAssignments(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r.Context())
	if uid == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}
	assignments, err := h.experiments.AssignAll(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":     uid,
		"assignments": assignments,
	})
}

type trackPayload struct {
	EventType  string                 `json:"event_type"`
	TargetID   string                 `json:"target_id"`
	Properties map[string]interface{} `json:"properties"`
}

func (h *handlers) trackEvent(w http.ResponseWriter, r *http.Request) {
	var p trackPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	uid := userIDFrom(r.Context())
	if uid == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}
	if p.EventType == "" {
		writeError(w, http.StatusBadRequest, "event_type required")
		return
	}
	err := h.writer.TrackEvent(analytics.Event{
		UserID:     uid,
		EventType:  p.EventType,
		TargetID:   p.TargetID,
		Properties: p.Properties,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *handlers) conversionReport(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	convEvent := r.URL.Query().Get("event")
	if convEvent == "" {
		convEvent = "purchase"
	}
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().Add(-7 * 24 * time.Hour)
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}
	rows, err := h.writer.ConversionByVariant(r.Context(), key, convEvent, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"experiment_key": key,
		"event":          convEvent,
		"since":          since.Format(time.RFC3339),
		"rows":           rows,
	})
}
