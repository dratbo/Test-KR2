package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/icons"
)

type IconHandler struct {
	dataClient *clients.DataClient
	httpClient *http.Client
}

func NewIconHandler(dataClient *clients.DataClient) *IconHandler {
	return &IconHandler{
		dataClient: dataClient,
		httpClient: &http.Client{},
	}
}

func (h *IconHandler) Serve(w http.ResponseWriter, r *http.Request) {
	className := strings.TrimPrefix(r.URL.Path, "/icons/")
	className = strings.TrimSuffix(className, ".png")
	if className == "" {
		http.Redirect(w, r, "/static/placeholder.svg", http.StatusFound)
		return
	}

	displayName := ""
	if item, err := h.dataClient.GetItem(className); err == nil && item != nil {
		displayName = item.DisplayName
	}

	for _, u := range icons.URLsForItem(displayName, className) {
		resp, err := h.httpClient.Get(u)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || len(body) == 0 {
			continue
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(body)
		return
	}

	http.Redirect(w, r, "/static/placeholder.svg", http.StatusFound)
}
