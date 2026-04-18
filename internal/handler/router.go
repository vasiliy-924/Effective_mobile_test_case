package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	httpSwagger "github.com/swaggo/http-swagger"
)

// RateLimitRouterConfig toggles request throttling for /api/v1 (health and swagger stay unlimited).
type RateLimitRouterConfig struct {
	Enabled      bool
	MaxRequests  int           // requests allowed per WindowLength per client key (real IP)
	WindowLength time.Duration // sliding window length
}

// NewRouter wires HTTP routes and middleware.
func NewRouter(log *slog.Logger, h *Handler, rl RateLimitRouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger(log))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		if rl.Enabled && rl.MaxRequests > 0 && rl.WindowLength > 0 {
			r.Use(httprate.Limit(rl.MaxRequests, rl.WindowLength,
				httprate.WithKeyFuncs(httprate.KeyByRealIP),
				httprate.WithLimitHandler(rateLimitJSON),
			))
		}
		r.Get("/subscriptions/cost", h.Cost)
		r.Post("/subscriptions", h.Create)
		r.Get("/subscriptions", h.List)
		r.Get("/subscriptions/{id}", h.Get)
		r.Put("/subscriptions/{id}", h.Update)
		r.Delete("/subscriptions/{id}", h.Delete)
	})

	return r
}

func rateLimitJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(errResp{Error: "rate limit exceeded"})
}
