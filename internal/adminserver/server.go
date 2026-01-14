package adminserver

import (
	"log"
	"net/http"

	"github.com/AntiSlang/tracker/internal/config"
	"github.com/AntiSlang/tracker/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Server struct {
	cfg *config.Config
	db  *storage.Storage
}

func New(cfg *config.Config, db *storage.Storage) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
	}
}

func (s *Server) Run() error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Telegram-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {

		r.Get("/faculties", s.getFacultiesHandler)
		r.Get("/programs", s.getProgramsHandler)

		r.Route("/user", func(r chi.Router) {
			r.Post("/register", s.registerUserHandler)
			r.Get("/me", s.getUserProfileHandler)
		})

		r.Route("/subjects", func(r chi.Router) {
			r.Get("/", s.getSubjectsHandler)
			r.Post("/create", s.createSubjectHandler)

			r.Route("/{subject_id}", func(r chi.Router) {
				r.Get("/tables", s.getSubjectTablesHandler)
				r.Post("/assign", s.assignTableHandler)
			})
		})
	})

	log.Printf("Admin Gateway REST API is listening on %s", s.cfg.ServerPort)
	log.Printf("Swagger UI is available at https://%s:%s/swagger/index.html", s.cfg.ServerHost, s.cfg.ServerPort)
	return http.ListenAndServe(":"+s.cfg.ServerPort, r)
}
