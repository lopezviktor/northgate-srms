package security

import (
	"strings"
	"sync"
	"time"
)

const (
	MaxFailedLoginAttempts = 5
	LoginLockoutDuration   = 2 * time.Minute
)

type LoginAttempt struct {
	FailedAttempts int
	LockedUntil    time.Time
}

type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]LoginAttempt
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		attempts: make(map[string]LoginAttempt),
	}
}

func (l *LoginLimiter) IsLocked(username, clientIP string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := loginAttemptKey(username, clientIP)

	attempt, exists := l.attempts[key]
	if !exists {
		return false
	}

	if attempt.LockedUntil.IsZero() {
		return false
	}

	if time.Now().After(attempt.LockedUntil) {
		delete(l.attempts, key)
		return false
	}

	return true
}

func (l *LoginLimiter) RegisterFailure(username, clientIP string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := loginAttemptKey(username, clientIP)

	attempt := l.attempts[key]
	attempt.FailedAttempts++

	if attempt.FailedAttempts >= MaxFailedLoginAttempts {
		attempt.LockedUntil = time.Now().Add(LoginLockoutDuration)
	}
	l.attempts[key] = attempt
}

func (l *LoginLimiter) RegisterSuccess(username, clientIP string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := loginAttemptKey(username, clientIP)

	delete(l.attempts, key)
}

func loginAttemptKey(username, clientIP string) string {
	normalisedUsername := strings.ToLower(strings.TrimSpace(username))
	normalisedIP := strings.TrimSpace(clientIP)

	return normalisedUsername + "|" + normalisedIP
}
