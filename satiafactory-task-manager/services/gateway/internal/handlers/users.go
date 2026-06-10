package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

type UsersHandler struct {
	userClient  *clients.UserClient
	searchTmpl  *template.Template
	favoriteTmpl *template.Template
}

func NewUsersHandler(userClient *clients.UserClient) (*UsersHandler, error) {
	searchTmpl, err := template.ParseFiles("templates/users_search.html")
	if err != nil {
		return nil, err
	}
	favoriteTmpl, err := template.ParseFiles("templates/users_favorites.html")
	if err != nil {
		return nil, err
	}
	return &UsersHandler{
		userClient:   userClient,
		searchTmpl:   searchTmpl,
		favoriteTmpl: favoriteTmpl,
	}, nil
}

func (h *UsersHandler) token(r *http.Request) (string, error) {
	c, err := r.Cookie("token")
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func (h *UsersHandler) Favorites(w http.ResponseWriter, r *http.Request) {
	token, err := h.token(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	users, err := h.userClient.ListFavorites(token)
	if err != nil {
		http.Error(w, "Failed to load favorites", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.favoriteTmpl.Execute(w, users)
}

func (h *UsersHandler) Search(w http.ResponseWriter, r *http.Request) {
	token, err := h.token(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	q := r.URL.Query().Get("q")
	if len(q) < 1 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<p class="hint">Введите имя игрока…</p>`))
		return
	}
	users, err := h.userClient.SearchUsers(token, q)
	if err != nil {
		http.Error(w, "Ошибка поиска", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.searchTmpl.Execute(w, struct {
		Query string
		Users []clients.UserSearchRow
	}{Query: q, Users: users})
}

func (h *UsersHandler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	token, err := h.token(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	action := r.URL.Query().Get("action")
	if action == "remove" {
		_ = h.userClient.RemoveFavorite(token, id)
	} else {
		_ = h.userClient.AddFavorite(token, id)
	}
	q := r.URL.Query().Get("q")
	if q != "" {
		r2 := r.Clone(r.Context())
		qv := r2.URL.Query()
		qv.Set("q", q)
		r2.URL.RawQuery = qv.Encode()
		h.Search(w, r2)
		return
	}
	h.Favorites(w, r)
}
