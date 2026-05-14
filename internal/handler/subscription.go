package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sekigo/em-subscriptions/internal/model"
	"github.com/sekigo/em-subscriptions/internal/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
	log *slog.Logger
}

func New(svc *service.SubscriptionService, log *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc, log: log}
}

// Register mounts all CRUDL + total endpoints under r.
func (h *SubscriptionHandler) Register(r chi.Router) {
	r.Route("/subscriptions", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/total", h.total) // must be declared before /{id}
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})
}

// create godoc
// @Summary      Create subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        body body     SubscriptionRequest true "Subscription payload"
// @Success      201  {object} SubscriptionResponse
// @Failure      400  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Router       /subscriptions [post]
func (h *SubscriptionHandler) create(w http.ResponseWriter, r *http.Request) {
	var req SubscriptionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sub := req.toModel(uuid.Nil)

	if err := h.svc.Create(r.Context(), sub); err != nil {
		h.respondError(w, r, err)
		return
	}
	h.log.Info("subscription created", "id", sub.ID, "user_id", sub.UserID)
	writeJSON(w, http.StatusCreated, fromModel(sub))
}

// get godoc
// @Summary      Get subscription by ID
// @Tags         subscriptions
// @Produce      json
// @Param        id   path     string true "Subscription UUID"
// @Success      200  {object} SubscriptionResponse
// @Failure      400  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	sub, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, fromModel(sub))
}

// update godoc
// @Summary      Update subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id   path     string              true "Subscription UUID"
// @Param        body body     SubscriptionRequest true "Subscription payload"
// @Success      200  {object} SubscriptionResponse
// @Failure      400  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var req SubscriptionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sub := req.toModel(id)

	if err := h.svc.Update(r.Context(), sub); err != nil {
		h.respondError(w, r, err)
		return
	}
	h.log.Info("subscription updated", "id", sub.ID)
	writeJSON(w, http.StatusOK, fromModel(sub))
}

// delete godoc
// @Summary      Delete subscription
// @Tags         subscriptions
// @Param        id   path     string true "Subscription UUID"
// @Success      204
// @Failure      400  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.respondError(w, r, err)
		return
	}
	h.log.Info("subscription deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// list godoc
// @Summary      List subscriptions
// @Tags         subscriptions
// @Produce      json
// @Param        user_id      query    string false "Filter by user UUID"
// @Param        service_name query    string false "Filter by service name (exact match)"
// @Param        limit        query    int    false "Page size (default 50, max 200)"
// @Param        offset       query    int    false "Page offset (default 0)"
// @Success      200          {array}  SubscriptionResponse
// @Failure      400          {object} ErrorResponse
// @Router       /subscriptions [get]
func (h *SubscriptionHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := model.ListFilter{Limit: 50}

	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		f.UserID = &uid
	}
	if v := q.Get("service_name"); v != "" {
		f.ServiceName = &v
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid limit"))
			return
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid offset"))
			return
		}
		f.Offset = n
	}

	subs, err := h.svc.List(r.Context(), f)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	resp := make([]SubscriptionResponse, 0, len(subs))
	for i := range subs {
		resp = append(resp, fromModel(&subs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// total godoc
// @Summary      Total cost over a period
// @Description  Sums monthly price * number of months the subscription is active inside [period_from, period_to]. Both bounds are inclusive whole months.
// @Tags         subscriptions
// @Produce      json
// @Param        period_from  query    string true  "Start month, MM-YYYY" example(01-2025)
// @Param        period_to    query    string true  "End month, MM-YYYY"   example(12-2025)
// @Param        user_id      query    string false "Filter by user UUID"
// @Param        service_name query    string false "Filter by service name (exact match)"
// @Success      200          {object} TotalResponse
// @Failure      400          {object} ErrorResponse
// @Router       /subscriptions/total [get]
func (h *SubscriptionHandler) total(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	from, err := model.ParseMonthYear(q.Get("period_from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("period_from: "+err.Error()))
		return
	}
	to, err := model.ParseMonthYear(q.Get("period_to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("period_to: "+err.Error()))
		return
	}

	f := model.TotalFilter{PeriodFrom: from, PeriodTo: to}

	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		f.UserID = &uid
	}
	if v := q.Get("service_name"); v != "" {
		f.ServiceName = &v
	}

	total, err := h.svc.Total(r.Context(), f)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, TotalResponse{
		Total:       total,
		PeriodFrom:  from,
		PeriodTo:    to,
		UserID:      f.UserID,
		ServiceName: f.ServiceName,
	})
}

// --- helpers ------------------------------------------------------------

func decodeJSON(body io.ReadCloser, v any) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

func (h *SubscriptionHandler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err)
	default:
		h.log.Error("internal error", "err", err, "path", r.URL.Path, "method", r.Method)
		writeError(w, http.StatusInternalServerError, errors.New("internal error"))
	}
}
