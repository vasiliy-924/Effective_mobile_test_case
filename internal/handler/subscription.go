package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wassiliy/subscriptions-service/internal/apperrors"
	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/service"
)

type Handler struct {
	svc SubscriptionService
}

func New(svc SubscriptionService) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary Create subscription
// @Description Creates a new subscription record
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param body body domain.CreateSubscription true "Payload"
// @Success 201 {object} domain.Subscription
// @Failure 400 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body domain.CreateSubscription
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.svc.Create(r.Context(), body)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// Get godoc
// @Summary Get subscription by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Subscription UUID"
// @Success 200 {object} domain.Subscription
// @Failure 400 {object} errResp
// @Failure 404 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	out, err := h.svc.Get(r.Context(), id)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Update godoc
// @Summary Update subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Subscription UUID"
// @Param body body domain.CreateSubscription true "Payload"
// @Success 200 {object} domain.Subscription
// @Failure 400 {object} errResp
// @Failure 404 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body domain.CreateSubscription
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.svc.Update(r.Context(), id, body)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete godoc
// @Summary Delete subscription
// @Tags subscriptions
// @Param id path string true "Subscription UUID"
// @Success 204
// @Failure 400 {object} errResp
// @Failure 404 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		mapErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// List godoc
// @Summary List subscriptions
// @Description Optional filters by user_id and exact service_name; pagination via limit/offset
// @Tags subscriptions
// @Produce json
// @Param user_id query string false "User UUID"
// @Param service_name query string false "Exact service name"
// @Param limit query int false "Page size (default 50, max 500)"
// @Param offset query int false "Offset"
// @Success 200 {array} domain.Subscription
// @Failure 400 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := service.ListInput{
		Limit:  parseIntDefault(q.Get("limit"), 50),
		Offset: parseIntDefault(q.Get("offset"), 0),
	}
	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		in.UserID = &uid
	}
	if v := q.Get("service_name"); v != "" {
		in.ServiceName = &v
	}
	out, err := h.svc.List(r.Context(), in)
	if err != nil {
		mapErr(w, err)
		return
	}
	if out == nil {
		out = []domain.Subscription{}
	}
	writeJSON(w, http.StatusOK, out)
}

// Cost godoc
// @Summary Total subscription cost for period
// @Description Sums monthly price for each calendar month in [from, to] where a subscription is active. Optional filters.
// @Tags subscriptions
// @Produce json
// @Param from query string true "Start month MM-YYYY"
// @Param to query string true "End month MM-YYYY"
// @Param user_id query string false "User UUID"
// @Param service_name query string false "Exact service name"
// @Success 200 {object} domain.CostReport
// @Failure 400 {object} errResp
// @Failure 500 {object} errResp
// @Router /subscriptions/cost [get]
func (h *Handler) Cost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "from and to are required (MM-YYYY)")
		return
	}
	in := service.CostInput{From: from, To: to}
	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		in.UserID = &uid
	}
	if v := q.Get("service_name"); v != "" {
		in.ServiceName = &v
	}
	out, err := h.svc.Cost(r.Context(), in)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type errResp struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errResp{Error: msg})
}

func mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, apperrors.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
