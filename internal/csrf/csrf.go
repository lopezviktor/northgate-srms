package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
)

const FormFieldName = "csrf_token"

type Manager struct {
	mu     sync.Mutex
	tokens map[string]string
}

func NewManager() *Manager {
	return &Manager{
		tokens: make(map[string]string),
	}
}

func (m *Manager) Generate(sessionID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	m.tokens[sessionID] = token
	m.mu.Unlock()

	return token, nil
}

func (m *Manager) Token(sessionID string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, ok := m.tokens[sessionID]
	return token, ok
}

func (m *Manager) Validate(r *http.Request, sessionID string) bool {
	submittedToken := r.FormValue(FormFieldName)

	m.mu.Lock()
	expectedToken, ok := m.tokens[sessionID]
	m.mu.Unlock()

	if !ok || submittedToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(submittedToken),
		[]byte(expectedToken),
	) == 1
}

func (m *Manager) Delete(sessionID string) {
	m.mu.Lock()
	delete(m.tokens, sessionID)
	m.mu.Unlock()
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
