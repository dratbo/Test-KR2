package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dratbo/satisfactory-task-manager/user-service/internal/middleware"
	"github.com/dratbo/satisfactory-task-manager/user-service/internal/repository"
)

type UsersHandler struct {
	userRepo     *repository.UserRepository
	favoriteRepo *repository.FavoriteRepository
}

func NewUsersHandler(userRepo *repository.UserRepository, favoriteRepo *repository.FavoriteRepository) *UsersHandler {
	return &UsersHandler{userRepo: userRepo, favoriteRepo: favoriteRepo}
}

func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAll()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	type userBrief struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	out := make([]userBrief, 0, len(users))
	for _, u := range users {
		out = append(out, userBrief{ID: u.ID, Username: u.Username})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *UsersHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	q := r.URL.Query().Get("q")
	users, err := h.userRepo.Search(q)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	type userRow struct {
		ID         int64  `json:"id"`
		Username   string `json:"username"`
		IsFavorite bool   `json:"is_favorite"`
	}
	out := make([]userRow, 0, len(users))
	for _, u := range users {
		if u.ID == userID {
			continue
		}
		fav, _ := h.favoriteRepo.IsFavorite(userID, u.ID)
		out = append(out, userRow{ID: u.ID, Username: u.Username, IsFavorite: fav})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *UsersHandler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	users, err := h.favoriteRepo.List(userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	type userBrief struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	out := make([]userBrief, 0, len(users))
	for _, u := range users {
		out = append(out, userBrief{ID: u.ID, Username: u.Username})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *UsersHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	favID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || favID <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.favoriteRepo.Add(userID, favID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UsersHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	favID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || favID <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.favoriteRepo.Remove(userID, favID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
