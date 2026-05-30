package security

import "testing"

func TestLoginLimiterUsesUsernameAndIP(t *testing.T) {

	limiter := NewLoginLimiter()
	username := "alice"
	ipA := "192.168.1.10"
	ipB := "192.168.1.20"
	for i := 0; i < MaxFailedLoginAttempts; i++ {
		limiter.RegisterFailure(username, ipA)
	}
	if !limiter.IsLocked(username, ipA) {
		t.Fatal("expected username and ipA combination to be locked")
	}
	if limiter.IsLocked(username, ipB) {
		t.Fatal("expected same username from different IP not to be locked")
	}

}

func TestLoginLimiterDoesNotLockDifferentUsernameFromSameIP(t *testing.T) {

	limiter := NewLoginLimiter()
	ip := "192.168.1.10"
	for i := 0; i < MaxFailedLoginAttempts; i++ {
		limiter.RegisterFailure("alice", ip)
	}
	if !limiter.IsLocked("alice", ip) {
		t.Fatal("expected alice from this IP to be locked")
	}
	if limiter.IsLocked("bob", ip) {
		t.Fatal("expected different username from same IP not to be locked")
	}

}

func TestLoginLimiterRegisterSuccessClearsOnlyMatchingUsernameAndIP(t *testing.T) {

	limiter := NewLoginLimiter()
	username := "alice"
	ipA := "192.168.1.10"
	ipB := "192.168.1.20"
	for i := 0; i < MaxFailedLoginAttempts; i++ {
		limiter.RegisterFailure(username, ipA)
		limiter.RegisterFailure(username, ipB)
	}
	limiter.RegisterSuccess(username, ipA)
	if limiter.IsLocked(username, ipA) {
		t.Fatal("expected successful login to clear lockout for matching username and IP")
	}
	if !limiter.IsLocked(username, ipB) {
		t.Fatal("expected different IP lockout to remain active")
	}

}
