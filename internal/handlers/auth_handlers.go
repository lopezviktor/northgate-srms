package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"northgate-srms/internal/auth"
)

type LoginPageData struct {
	Error string
}

type AuthHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
}

func NewAuthHandler(db *sql.DB, sessions *auth.SessionManager) *AuthHandler {
	return &AuthHandler{
		DB:       db,
		Sessions: sessions,
	}
}

func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := h.Sessions.Get(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	RenderTemplate(w, "login.html", LoginPageData{})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		RenderTemplate(w, "login.html", LoginPageData{
			Error: "Invalid username or password.",
		})
		return
	}

	if len(username) > 30 || len(password) > 200 {
		RenderTemplate(w, "login.html", LoginPageData{
			Error: "Invalid username or password.",
		})
		return
	}

	user, err := auth.AuthenticateUser(h.DB, username, password)
	if err != nil {
		RenderTemplate(w, "login.html", LoginPageData{
			Error: "Invalid username or password.",
		})
		return
	}

	if err := h.Sessions.Create(w, user); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
