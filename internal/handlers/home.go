package handlers

import (
	"net/http"

	"northgate-srms/internal/auth"
)

type HomePageData struct {
	IsAuthenticated bool
	Username        string
	Role            string
}

type HomeHandler struct {
	Sessions *auth.SessionManager
}

func NewHomeHandler(sessions *auth.SessionManager) *HomeHandler {
	return &HomeHandler{
		Sessions: sessions,
	}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := HomePageData{}

	if session, ok := h.Sessions.Get(r); ok {
		data.IsAuthenticated = true
		data.Username = session.User.Username
		data.Role = session.User.Role
	}
	RenderTemplate(w, "home.html", data)
}
