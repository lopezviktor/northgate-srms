package handlers

import (
	"net/http"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
)

type HomePageData struct {
	IsAuthenticated bool
	Username        string
	Role            string
	CSRFToken       string
}

type HomeHandler struct {
	Sessions *auth.SessionManager
	CSRF     *csrf.Manager
}

func NewHomeHandler(sessions *auth.SessionManager, csrfManager *csrf.Manager) *HomeHandler {
	return &HomeHandler{
		Sessions: sessions,
		CSRF:     csrfManager,
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
		token, err := h.CSRF.Generate(session.ID)

		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		data.IsAuthenticated = true
		data.Username = session.User.Username
		data.Role = session.User.Role
		data.CSRFToken = token
	}
	RenderTemplate(w, "home.html", data)
}
