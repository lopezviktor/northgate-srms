package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const SessionCookieName = "northgate_session"
const SessionInactivityTimeout = 15 * time.Minute

type User struct {
	ID       int64
	Username string
	Role     string
}

type Session struct {
	ID           string
	User         User
	CreatedAt    time.Time
	LastActivity time.Time
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]Session),
	}
}

func (m *SessionManager) Create(w http.ResponseWriter, user User) error {
	sessionID, err := generateSessionID()
	if err != nil {
		return err
	}

	now := time.Now()

	session := Session{
		ID:           sessionID,
		User:         user,
		CreatedAt:    now,
		LastActivity: now,
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	}

	http.SetCookie(w, cookie)

	return nil
}

func (m *SessionManager) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return Session{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[cookie.Value]
	if !ok {
		return Session{}, false
	}

	now := time.Now()
	if now.Sub(session.LastActivity) > SessionInactivityTimeout {
		delete(m.sessions, cookie.Value)
		return Session{}, false
	}

	session.LastActivity = now
	m.sessions[cookie.Value] = session

	return session, true
}

func (m *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
	}

	expiredCookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, expiredCookie)
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
