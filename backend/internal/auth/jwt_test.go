package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	m := NewManager("test-secret", time.Hour)

	token, err := m.GenerateToken(42, "client")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Role != "client" {
		t.Errorf("Role = %q, want %q", claims.Role, "client")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	signed := NewManager("secret-one", time.Hour)
	verifier := NewManager("secret-two", time.Hour)

	token, err := signed.GenerateToken(1, "client")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	if _, err := verifier.ParseToken(token); err == nil {
		t.Error("expected error parsing token signed with a different secret, got nil")
	}
}

func TestParseToken_Expired(t *testing.T) {
	// Отрицательный TTL — токен считается истёкшим сразу после выпуска.
	m := NewManager("secret", -time.Hour)

	token, err := m.GenerateToken(1, "client")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := m.ParseToken(token); err == nil {
		t.Error("expected error parsing expired token, got nil")
	}
}

func TestParseToken_Garbage(t *testing.T) {
	m := NewManager("secret", time.Hour)
	if _, err := m.ParseToken("not-a-real-jwt"); err == nil {
		t.Error("expected error parsing garbage input, got nil")
	}
}
