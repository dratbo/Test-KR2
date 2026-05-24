package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/dratbo/satisfactory-task-manager/task-service/internal/config"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/handlers"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/middleware"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/repository"
	_ "github.com/lib/pq"
)

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

	taskRepo := repository.NewTaskRepository(db)
	taskHandler := handlers.NewTaskHandler(taskRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", taskHandler.CreateTask)
	mux.HandleFunc("GET /tasks", taskHandler.GetTasks)
	mux.HandleFunc("DELETE /tasks/{id}", taskHandler.DeleteTask)

	// Wrap with auth middleware
	authMiddleware := middleware.AuthMiddleware(cfg)
	protectedMux := authMiddleware(mux)

	log.Printf("Task service running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, protectedMux); err != nil {
		log.Fatal(err)
	}
}
