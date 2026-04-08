package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/haroldcamargo/english/backend/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewRouter(cfg config.Config, database *mongo.Database) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://english.haroldsoftware.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authMW := NewAuthMiddleware(cfg)
	api := NewAPI(cfg, database)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", api.Register)
		r.Post("/auth/login", api.Login)

		r.Group(func(r chi.Router) {
			r.Use(authMW.RequireAuth)
			r.Get("/me", api.Me)
			r.Post("/vocab", api.QuickAddVocab)
			r.Put("/vocab/{id}", api.UpdateVocab)
			r.Get("/vocab/recent", api.RecentVocab)
			r.Post("/practice/next", api.PracticeNext)
			r.Post("/practice/answer", api.PracticeAnswer)
		})
	})

	return r
}
