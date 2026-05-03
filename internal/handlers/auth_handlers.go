package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/security"
)

const loginCSRFCookieName = "northgate_login_csrf"

type LoginPageData struct {
	Error     string
	CSRFToken string
}

type AuthHandler struct {
	DB           *sql.DB
	Sessions     *auth.SessionManager
	CSRF         *csrf.Manager
	LoginLimiter *security.LoginLimiter
}

func NewAuthHandler(db *sql.DB, sessions *auth.SessionManager, csrfManager *csrf.Manager, loginLimiter *security.LoginLimiter) *AuthHandler {
	return &AuthHandler{
		DB:           db,
		Sessions:     sessions,
		CSRF:         csrfManager,
		LoginLimiter: loginLimiter,
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

	token, err := h.createLoginCSRFToken(w)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	RenderTemplate(w, "login.html", LoginPageData{
		CSRFToken: token,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.validateLoginCSRFToken(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	if len(username) > 30 || len(password) > 200 {
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	if h.LoginLimiter.IsLocked(username) {
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	user, err := auth.AuthenticateUser(h.DB, username, password)
	if err != nil {
		h.LoginLimiter.RegisterFailure(username)
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	h.LoginLimiter.RegisterSuccess(username)

	if err := h.Sessions.Create(w, user); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := h.Sessions.Get(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !h.validateSessionCSRFToken(r, session.ID) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	h.CSRF.Delete(session.ID)
	h.Sessions.Destroy(w, r)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) createLoginCSRFToken(w http.ResponseWriter) (string, error) {
	token, err := h.CSRF.Generate("login")
	if err != nil {
		return "", err
	}

	cookie := &http.Cookie{
		Name:     loginCSRFCookieName,
		Value:    "login",
		Path:     "/login",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	}

	http.SetCookie(w, cookie)

	return token, nil
}

func (h *AuthHandler) validateLoginCSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(loginCSRFCookieName)
	if err != nil {
		return false
	}

	return h.CSRF.Validate(r, cookie.Value)
}

func (h *AuthHandler) renderLoginWithError(w http.ResponseWriter, message string) {
	token, err := h.createLoginCSRFToken(w)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	RenderTemplate(w, "login.html", LoginPageData{
		Error:     message,
		CSRFToken: token,
	})
}

func (h *AuthHandler) validateSessionCSRFToken(r *http.Request, sessionID string) bool {
	return h.CSRF.Validate(r, sessionID)
}
