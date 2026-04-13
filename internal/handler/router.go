package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter wires HTTP routes and middleware.
func NewRouter(log *slog.Logger, h *Handler) http.Handler {
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
		r.Get("/subscriptions/cost", h.Cost)
		r.Post("/subscriptions", h.Create)
		r.Get("/subscriptions", h.List)
		r.Get("/subscriptions/{id}", h.Get)
		r.Put("/subscriptions/{id}", h.Update)
		r.Delete("/subscriptions/{id}", h.Delete)
	})

	return r
}
