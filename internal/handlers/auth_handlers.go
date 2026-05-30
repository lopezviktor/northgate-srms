package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/security"
	"northgate-srms/internal/validation"
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

	loginCSRFCookie, err := r.Cookie(loginCSRFCookieName)
	if err != nil || !h.CSRF.Validate(r, loginCSRFCookie.Value) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	preSessionID := loginCSRFCookie.Value

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.CSRF.Delete(preSessionID)
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	if !validation.IsValidUsername(username) || len(password) > 200 {
		h.CSRF.Delete(preSessionID)
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	clientIP := clientIPFromRequest(r)

	if h.LoginLimiter.IsLocked(username, clientIP) {
		h.CSRF.Delete(preSessionID)
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	user, err := auth.AuthenticateUser(h.DB, username, password)
	if err != nil {
		h.LoginLimiter.RegisterFailure(username, clientIP)
		h.CSRF.Delete(preSessionID)
		h.renderLoginWithError(w, "Invalid username or password.")
		return
	}

	h.LoginLimiter.RegisterSuccess(username, clientIP)
	h.CSRF.Delete(preSessionID)
	expireLoginCSRFCookie(w)

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
	preSessionID, err := generatePreSessionID()
	if err != nil {
		return "", err
	}

	token, err := h.CSRF.Generate(preSessionID)
	if err != nil {
		return "", err
	}

	cookie := &http.Cookie{
		Name:     loginCSRFCookieName,
		Value:    preSessionID,
		Path:     "/login",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	}

	http.SetCookie(w, cookie)

	return token, nil
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

func expireLoginCSRFCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     loginCSRFCookieName,
		Value:    "",
		Path:     "/login",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}

	http.SetCookie(w, cookie)
}

func (h *AuthHandler) validateSessionCSRFToken(r *http.Request, sessionID string) bool {
	return h.CSRF.Validate(r, sessionID)
}

func generatePreSessionID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
