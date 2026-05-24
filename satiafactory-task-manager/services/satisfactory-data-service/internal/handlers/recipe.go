package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/repository"
)

type RecipeHandler struct {
	repo *repository.RecipeRepository
}

func NewRecipeHandler(repo *repository.RecipeRepository) *RecipeHandler {
	return &RecipeHandler{repo: repo}
}

func (h *RecipeHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}
