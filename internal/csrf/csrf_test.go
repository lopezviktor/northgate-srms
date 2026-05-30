package csrf

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateStoresTokenForSession(t *testing.T) {

	manager := NewManager()
	sessionID := "session-123"
	token, err := manager.Generate(sessionID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected generated token not to be empty")
	}
	storedToken, ok := manager.Token(sessionID)
	if !ok {
		t.Fatal("expected token to be stored for session")
	}
	if storedToken != token {
		t.Fatalf("expected stored token %q, got %q", token, storedToken)
	}

}

func TestValidateAcceptsCorrectToken(t *testing.T) {

	manager := NewManager()
	sessionID := "session-123"
	token, err := manager.Generate(sessionID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	request := httptest.NewRequest("POST", "/record/update", strings.NewReader(FormFieldName+"="+token))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !manager.Validate(request, sessionID) {
		t.Fatal("expected valid CSRF token to be accepted")
	}

}

func TestValidateRejectsInvalidToken(t *testing.T) {

	manager := NewManager()
	sessionID := "session-123"
	_, err := manager.Generate(sessionID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	request := httptest.NewRequest("POST", "/record/update", strings.NewReader(FormFieldName+"=invalid-token"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if manager.Validate(request, sessionID) {
		t.Fatal("expected invalid CSRF token to be rejected")
	}

}

func TestValidateRejectsMissingToken(t *testing.T) {

	manager := NewManager()
	sessionID := "session-123"
	_, err := manager.Generate(sessionID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	request := httptest.NewRequest("POST", "/record/update", strings.NewReader("phone=123456789"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if manager.Validate(request, sessionID) {
		t.Fatal("expected missing CSRF token to be rejected")
	}

}

func TestValidateRejectsDeletedToken(t *testing.T) {

	manager := NewManager()
	sessionID := "session-123"
	token, err := manager.Generate(sessionID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	manager.Delete(sessionID)
	request := httptest.NewRequest("POST", "/record/update", strings.NewReader(FormFieldName+"="+token))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if manager.Validate(request, sessionID) {
		t.Fatal("expected deleted CSRF token to be rejected")
	}

}
