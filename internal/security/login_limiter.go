package security

import (
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

func (l *LoginLimiter) IsLocked(username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, exists := l.attempts[username]
	if !exists {
		return false
	}

	if attempt.LockedUntil.IsZero() {
		return false
	}

	if time.Now().After(attempt.LockedUntil) {
		delete(l.attempts, username)
		return false
	}

	return true
}

func (l *LoginLimiter) RegisterFailure(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[username]
	attempt.FailedAttempts++

	if attempt.FailedAttempts >= MaxFailedLoginAttempts {
		attempt.LockedUntil = time.Now().Add(LoginLockoutDuration)
	}
	l.attempts[username] = attempt
}

func (l *LoginLimiter) RegisterSuccess(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, username)
}
