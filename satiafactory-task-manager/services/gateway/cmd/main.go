package main

import (
	"log"
	"net/http"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/config"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/handlers"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/metrics"
)

func main() {
	cfg := config.Load()

	userClient := clients.NewUserClient(cfg.UserServiceURL)
	taskClient := clients.NewTaskClient(cfg.TaskServiceURL)
	dataClient := clients.NewDataClient(cfg.DataServiceURL)

	authHandler := handlers.NewAuthHandler(userClient)
	taskHandler, err := handlers.NewTaskHandler(taskClient, userClient, dataClient)
	if err != nil {
		log.Fatal("Failed to init task handler:", err)
	}
	recipeHandler, err := handlers.NewRecipeHandler(dataClient)
	if err != nil {
		log.Fatal("Failed to init recipe handler:", err)
	}
	usersHandler, err := handlers.NewUsersHandler(userClient)
	if err != nil {
		log.Fatal("Failed to init users handler:", err)
	}
	iconHandler := handlers.NewIconHandler(dataClient)

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authHandler.RegisterPage(w, r)
		} else if r.Method == http.MethodPost {
			authHandler.Register(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authHandler.LoginPage(w, r)
		} else if r.Method == http.MethodPost {
			authHandler.Login(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/logout", authHandler.Logout)

	mux.HandleFunc("/", taskHandler.Index)
	mux.HandleFunc("/my-tasks", taskHandler.MyTasks)
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			taskHandler.GetTasks(w, r)
		} else if r.Method == http.MethodPost {
			taskHandler.CreateTask(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/delete/", taskHandler.DeleteTask)
	mux.HandleFunc("/tasks/detail/", taskHandler.GetTaskDetail)
	mux.HandleFunc("/tasks/production/", taskHandler.GetTaskProduction)
	mux.HandleFunc("/tasks/take/", taskHandler.TakeTask)
	mux.HandleFunc("/tasks/status/", taskHandler.UpdateTaskStatus)
	mux.HandleFunc("/tasks/assign/", taskHandler.AssignTask)
	mux.HandleFunc("/tasks/tier/", taskHandler.UpdateHubTier)
	mux.HandleFunc("/icons/", iconHandler.Serve)
	mux.HandleFunc("/recipes/search", recipeHandler.Search)
	mux.HandleFunc("/recipes/preview", recipeHandler.Preview)
	mux.HandleFunc("/recipes/chain", recipeHandler.Chain)
	mux.HandleFunc("/recipes/chain-reverse", recipeHandler.ChainReverse)
	mux.HandleFunc("/users/favorites", usersHandler.Favorites)
	mux.HandleFunc("/users/search", usersHandler.Search)
	mux.HandleFunc("/users/favorite/toggle", usersHandler.ToggleFavorite)

	root := http.NewServeMux()
	root.Handle("GET /metrics", metrics.Handler())
	root.Handle("/", metrics.Middleware(mux))

	log.Printf("Gateway running on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, root))
}
