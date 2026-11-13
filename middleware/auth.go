package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/naushadnizarali/dev-modules/auth"
)

var (
	ErrMissingToken         = errors.New("middleware: authorization token is required")
	ErrInvalidAuthorization = errors.New("middleware: authorization header must use Bearer scheme")
)

// AuthConfig controls JWT authentication middleware.
type AuthConfig struct {
	Manager      *auth.Manager
	Token        func(*http.Request) (string, error)
	Skip         func(*http.Request) bool
	Unauthorized func(http.ResponseWriter, *http.Request, error)
}

// Authenticate validates a JWT and makes its claims available through the
// request context.
func Authenticate(cfg AuthConfig) (func(http.Handler) http.Handler, error) {
	if cfg.Manager == nil {
		return nil, errors.New("middleware: auth manager is required")
	}
	if cfg.Token == nil {
		cfg.Token = BearerToken
	}
	if cfg.Unauthorized == nil {
		cfg.Unauthorized = defaultUnauthorized
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			value, err := cfg.Token(r)
			if err != nil {
				cfg.Unauthorized(w, r, err)
				return
			}
			claims, err := cfg.Manager.Parse(value)
			if err != nil {
				cfg.Unauthorized(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.ContextWithClaims(r.Context(), claims)))
		})
	}, nil
}

// BearerToken extracts a token from the Authorization header.
func BearerToken(r *http.Request) (string, error) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 0 {
		return "", ErrMissingToken
	}
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", ErrInvalidAuthorization
	}
	return parts[1], nil
}

// CookieToken returns a token extractor for a named HTTP cookie.
func CookieToken(name string) func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(name)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			return "", ErrMissingToken
		}
		return strings.TrimSpace(cookie.Value), nil
	}
}

func defaultUnauthorized(w http.ResponseWriter, _ *http.Request, _ error) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
