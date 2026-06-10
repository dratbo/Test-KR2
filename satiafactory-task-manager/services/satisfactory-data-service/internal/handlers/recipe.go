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

func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	className := r.PathValue("className")
	if className == "" {
		http.Error(w, "missing className", http.StatusBadRequest)
		return
	}
	recipe, err := h.repo.GetByClassName(className)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) GetRecipesByProduct(w http.ResponseWriter, r *http.Request) {
	itemClass := r.PathValue("className")
	if itemClass == "" {
		http.Error(w, "missing className", http.StatusBadRequest)
		return
	}
	includeAlternates := r.URL.Query().Get("include_alternates") == "1" || r.URL.Query().Get("include_alternates") == "true"
	recipes, err := h.repo.GetByProduct(itemClass, includeAlternates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

func (h *RecipeHandler) HasRecipeForProduct(w http.ResponseWriter, r *http.Request) {
	itemClass := r.PathValue("className")
	if itemClass == "" {
		http.Error(w, "missing className", http.StatusBadRequest)
		return
	}
	exists, err := h.repo.HasRecipeForProduct(itemClass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"craftable": exists})
}

func (h *RecipeHandler) SearchRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	includeAlternates := r.URL.Query().Get("include_alternates") == "1" || r.URL.Query().Get("include_alternates") == "true"
	recipes, err := h.repo.Search(q, 40, includeAlternates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}
