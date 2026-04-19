package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/analytics"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/catalog"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/experiments"
)

type Deps struct {
	Catalog       *catalog.Service
	Experiments   *experiments.Service
	Writer        *analytics.Writer
	AllowedOrigin string
}

func NewRouter(d Deps) http.Handler {
	h := &handlers{
		catalog:     d.Catalog,
		experiments: d.Experiments,
		writer:      d.Writer,
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(corsMiddleware(d.AllowedOrigin))

	r.Get("/healthz", h.health)
	r.Get("/readyz", h.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(userIDMiddleware)

		r.Get("/products", h.listProducts)
		r.Get("/products/{id}", h.getProduct)

		r.Get("/experiments", h.listExperiments)
		r.Get("/experiments/{key}/report", h.conversionReport)

		r.Get("/assignments", h.getAssignments)

		r.Post("/events", h.trackEvent)
	})

	return r
}
