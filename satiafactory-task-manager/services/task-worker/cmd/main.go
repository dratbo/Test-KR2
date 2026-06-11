package main

import (
	"database/sql"
	"log"

	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/cache"
	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/config"
	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/consumer"
	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/repository"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connect:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("database ping:", err)
	}

	taskCache := cache.New(cfg.RedisURL, cfg.RedisCacheTTL)
	taskRepo := repository.New(db)
	worker := consumer.New(taskRepo, taskCache)

	log.Println("task-worker starting")
	if err := worker.Run(cfg.RabbitMQURL); err != nil {
		log.Fatal(err)
	}
}
