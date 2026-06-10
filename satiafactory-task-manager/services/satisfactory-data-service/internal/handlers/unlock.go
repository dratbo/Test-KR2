package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/repository"
)

type UnlockHandler struct {
	repo *repository.UnlockRepository
}

func NewUnlockHandler(repo *repository.UnlockRepository) *UnlockHandler {
	return &UnlockHandler{repo: repo}
}

func (h *UnlockHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	idx, err := h.repo.GetIndex()
	if err != nil {
		http.Error(w, "failed to load unlock index", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(idx)
}
