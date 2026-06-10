package main

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"net/http"

	"github.com/dratbo/satisfactory-task-manager/user-service/internal/config"
	"github.com/dratbo/satisfactory-task-manager/user-service/internal/database"
	"github.com/dratbo/satisfactory-task-manager/user-service/internal/handlers"
	"github.com/dratbo/satisfactory-task-manager/user-service/internal/middleware"
	"github.com/dratbo/satisfactory-task-manager/user-service/internal/repository"
	_ "github.com/lib/pq"
)

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("→ %s %s", r.Method, r.URL.Path)
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				log.Printf("   Body: %s", string(bodyBytes))
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		next(w, r)
	}
}

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("database unreachable:", err)
	}
	database.RunMigrations(db, "migrations")

	userRepo := repository.NewUserRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	authHandler := handlers.NewAuthHandler(userRepo, cfg)
	usersHandler := handlers.NewUsersHandler(userRepo, favoriteRepo)

	withAuth := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(cfg.JWTSecret, h)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", loggingMiddleware(authHandler.Register))
	mux.HandleFunc("POST /api/login", loggingMiddleware(authHandler.Login))
	mux.HandleFunc("GET /api/users", usersHandler.ListUsers)

	mux.Handle("GET /api/users/search", withAuth(usersHandler.Search))
	mux.Handle("GET /api/users/favorites", withAuth(usersHandler.ListFavorites))
	mux.Handle("POST /api/users/favorites/{id}", withAuth(usersHandler.AddFavorite))
	mux.Handle("DELETE /api/users/favorites/{id}", withAuth(usersHandler.RemoveFavorite))

	log.Printf("User service running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}
