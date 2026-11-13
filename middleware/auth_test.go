package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/naushadnizarali/dev-modules/auth"
	"github.com/naushadnizarali/dev-modules/middleware"
)

func TestAuthenticateAddsClaimsToContext(t *testing.T) {
	manager, err := auth.New(auth.Config{
		Secret: []byte("0123456789abcdef0123456789abcdef"),
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.Generate("user-42", auth.Claims{"role": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	wrap, err := middleware.Authenticate(middleware.AuthConfig{Manager: manager})
	if err != nil {
		t.Fatal(err)
	}
	handler := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, ok := auth.StringClaim(r.Context(), "sub")
		if !ok || subject != "user-42" {
			t.Errorf("subject = %q, %v", subject, ok)
		}
		role, ok := auth.StringClaim(r.Context(), "role")
		if !ok || role != "admin" {
			t.Errorf("role = %q, %v", role, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestAuthenticateRejectsMissingAndInvalidTokens(t *testing.T) {
	manager, err := auth.New(auth.Config{
		Secret: []byte("0123456789abcdef0123456789abcdef"),
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	wrap, err := middleware.Authenticate(middleware.AuthConfig{Manager: manager})
	if err != nil {
		t.Fatal(err)
	}
	handler := wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not run")
	}))

	for _, header := range []string{"", "Basic credentials", "Bearer invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/private", nil)
		request.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("header %q status = %d, want %d", header, response.Code, http.StatusUnauthorized)
		}
	}
}
