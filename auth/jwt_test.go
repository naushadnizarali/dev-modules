package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/naushadnizarali/dev-modules/auth"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func TestGenerateParseAndContextClaims(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	manager, err := auth.New(auth.Config{
		Secret:    testSecret,
		Issuer:    "test-service",
		Audience:  []string{"test-api"},
		TTL:       time.Hour,
		Algorithm: "HS256",
		Now:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.Generate("user-42", auth.Claims{
		"role":       "admin",
		"account_id": 72,
		"roles":      []string{"admin", "editor"},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	ctx := auth.ContextWithClaims(context.Background(), claims)
	if subject, ok := auth.StringClaim(ctx, "sub"); !ok || subject != "user-42" {
		t.Fatalf("subject = %q, %v", subject, ok)
	}
	if role, ok := auth.StringClaim(ctx, "role"); !ok || role != "admin" {
		t.Fatalf("role = %q, %v", role, ok)
	}
	if accountID, ok := auth.Int64Claim(ctx, "account_id"); !ok || accountID != 72 {
		t.Fatalf("account_id = %d, %v", accountID, ok)
	}
	if roles, ok := auth.StringSliceClaim(ctx, "roles"); !ok || len(roles) != 2 || roles[1] != "editor" {
		t.Fatalf("roles = %#v, %v", roles, ok)
	}
}

func TestParseRejectsExpiredAndWrongSecret(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	manager, err := auth.New(auth.Config{Secret: testSecret, TTL: time.Minute, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.Generate("user-42", nil)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := manager.Parse(token); err == nil {
		t.Fatal("expected expired token error")
	}

	other, err := auth.New(auth.Config{Secret: []byte("abcdef0123456789abcdef0123456789"), TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Parse(token); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestGenerateRejectsReservedClaim(t *testing.T) {
	manager, err := auth.New(auth.Config{Secret: testSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Generate("user-42", auth.Claims{"exp": 1}); err == nil {
		t.Fatal("expected reserved claim error")
	}
}

func TestNewValidatesConfig(t *testing.T) {
	tests := []auth.Config{
		{},
		{Secret: []byte("short"), TTL: time.Hour},
		{Secret: testSecret},
		{Secret: testSecret, TTL: time.Hour, Algorithm: "none"},
		{Secret: testSecret, TTL: time.Hour, Leeway: -1},
	}
	for _, cfg := range tests {
		if _, err := auth.New(cfg); err == nil {
			t.Fatalf("expected error for config: %#v", cfg)
		}
	}
}
