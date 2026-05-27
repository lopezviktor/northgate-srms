package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"northgate-srms/internal/storage"
)

const SessionCookieName = "northgate_session"
const SessionInactivityTimeout = 15 * time.Minute
const SessionAbsoluteTimeout = time.Hour

type User struct {
	ID       int64
	Username string
	Role     string
}

type Session struct {
	ID           string
	User         User
	CreatedAt    time.Time
	ExpiresAt    time.Time
	LastActivity time.Time
}

type SessionManager struct {
	db *sql.DB
}

func NewSessionManager(db *sql.DB) *SessionManager {
	manager := &SessionManager{db: db}

	if err := manager.DeleteExpiredSessions(); err != nil {
		fmt.Printf("delete expired sessions: %v\n", err)
	}

	return manager
}

func (m *SessionManager) Create(w http.ResponseWriter, user User) error {
	sessionID, err := generateSessionID()
	if err != nil {
		return fmt.Errorf("generate session id: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(SessionAbsoluteTimeout)
	sessionHash := HashSessionID(sessionID)

	_, err = m.db.Exec(
		`INSERT INTO sessions (session_hash, user_id, created_at, expires_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?)`,
		sessionHash,
		user.ID,
		now.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionAbsoluteTimeout.Seconds()),
	}

	http.SetCookie(w, cookie)

	return nil
}

func (m *SessionManager) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return Session{}, false
	}

	sessionID := cookie.Value
	sessionHash := HashSessionID(sessionID)

	var userID int64
	var createdAtText string
	var expiresAtText string
	var lastActivityText string

	err = m.db.QueryRow(
		`SELECT user_id, created_at, expires_at, last_activity_at
		 FROM sessions
		 WHERE session_hash = ?`,
		sessionHash,
	).Scan(
		&userID,
		&createdAtText,
		&expiresAtText,
		&lastActivityText,
	)
	if err != nil {
		return Session{}, false
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtText)
	if err != nil {
		_ = m.deleteSessionByHash(sessionHash)
		return Session{}, false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtText)
	if err != nil {
		_ = m.deleteSessionByHash(sessionHash)
		return Session{}, false
	}

	lastActivity, err := time.Parse(time.RFC3339, lastActivityText)
	if err != nil {
		_ = m.deleteSessionByHash(sessionHash)
		return Session{}, false
	}

	now := time.Now().UTC()
	if now.After(expiresAt) || now.Sub(lastActivity) > SessionInactivityTimeout {
		_ = m.deleteSessionByHash(sessionHash)
		return Session{}, false
	}

	storedUser, err := storage.GetUserByID(m.db, userID)
	if err != nil || !storedUser.IsActive {
		_ = m.deleteSessionByHash(sessionHash)
		return Session{}, false
	}

	_, err = m.db.Exec(
		`UPDATE sessions
		 SET last_activity_at = ?
		 WHERE session_hash = ?`,
		now.Format(time.RFC3339),
		sessionHash,
	)
	if err != nil {
		return Session{}, false
	}

	return Session{
		ID: sessionID,
		User: User{
			ID:       storedUser.ID,
			Username: storedUser.Username,
			Role:     storedUser.Role,
		},
		CreatedAt:    createdAt,
		ExpiresAt:    expiresAt,
		LastActivity: now,
	}, true
}

func (m *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		sessionHash := HashSessionID(cookie.Value)
		_ = m.deleteSessionByHash(sessionHash)
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

func (m *SessionManager) DeleteExpiredSessions() error {
	_, err := m.db.Exec(
		`DELETE FROM sessions
		 WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}

func (m *SessionManager) deleteSessionByHash(sessionHash string) error {
	_, err := m.db.Exec(
		`DELETE FROM sessions
		 WHERE session_hash = ?`,
		sessionHash,
	)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func HashSessionID(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:])
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
