// Package auth provides JWT generation, validation, and request-context claims.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var registeredClaims = map[string]struct{}{
	"aud": {}, "exp": {}, "iat": {}, "iss": {}, "jti": {}, "nbf": {}, "sub": {},
}

// Config controls JWT signing and standard claim validation. Secret must be at
// least 32 bytes and Algorithm must be HS256, HS384, or HS512.
type Config struct {
	Secret    []byte
	Algorithm string
	Issuer    string
	Audience  []string
	TTL       time.Duration
	Leeway    time.Duration
	Now       func() time.Time
}

// Manager signs and validates JWT access tokens.
type Manager struct {
	secret   []byte
	method   jwt.SigningMethod
	issuer   string
	audience []string
	ttl      time.Duration
	leeway   time.Duration
	now      func() time.Time
}

// Claims contains standard and application-defined token claims.
type Claims map[string]any

// New creates a JWT manager with strict algorithm and standard-claim checks.
func New(cfg Config) (*Manager, error) {
	if len(cfg.Secret) < 32 {
		return nil, errors.New("auth: JWT secret must be at least 32 bytes")
	}
	if cfg.TTL <= 0 {
		return nil, errors.New("auth: JWT TTL must be greater than zero")
	}
	if cfg.Leeway < 0 {
		return nil, errors.New("auth: JWT leeway cannot be negative")
	}
	method, err := signingMethod(cfg.Algorithm)
	if err != nil {
		return nil, err
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Manager{
		secret:   append([]byte(nil), cfg.Secret...),
		method:   method,
		issuer:   strings.TrimSpace(cfg.Issuer),
		audience: append([]string(nil), cfg.Audience...),
		ttl:      cfg.TTL,
		leeway:   cfg.Leeway,
		now:      now,
	}, nil
}

// Generate signs a token using the configured TTL.
func (m *Manager) Generate(subject string, custom Claims) (string, error) {
	if m == nil {
		return "", errors.New("auth: JWT manager is nil")
	}
	return m.GenerateWithTTL(subject, custom, m.ttl)
}

// GenerateWithTTL signs a token with a per-token lifetime, useful for issuing
// access and refresh tokens from the same manager.
func (m *Manager) GenerateWithTTL(subject string, custom Claims, ttl time.Duration) (string, error) {
	if m == nil {
		return "", errors.New("auth: JWT manager is nil")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", errors.New("auth: token subject is required")
	}
	if ttl <= 0 {
		return "", errors.New("auth: token TTL must be greater than zero")
	}
	claims := jwt.MapClaims{}
	for key, value := range custom {
		if _, reserved := registeredClaims[key]; reserved {
			return "", fmt.Errorf("auth: custom claim %q is reserved", key)
		}
		claims[key] = value
	}

	now := m.now().UTC()
	id, err := tokenID()
	if err != nil {
		return "", err
	}
	claims["sub"] = subject
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	claims["exp"] = now.Add(ttl).Unix()
	claims["jti"] = id
	if m.issuer != "" {
		claims["iss"] = m.issuer
	}
	if len(m.audience) > 0 {
		claims["aud"] = append([]string(nil), m.audience...)
	}

	token := jwt.NewWithClaims(m.method, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign JWT: %w", err)
	}
	return signed, nil
}

// Parse verifies a token and returns a detached copy of all claims.
func (m *Manager) Parse(value string) (Claims, error) {
	if m == nil {
		return nil, errors.New("auth: JWT manager is nil")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("auth: token is required")
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{m.method.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(m.leeway),
		jwt.WithTimeFunc(m.now),
	}
	if m.issuer != "" {
		options = append(options, jwt.WithIssuer(m.issuer))
	}
	if len(m.audience) > 0 {
		options = append(options, jwt.WithAudience(m.audience...))
	}

	parsedClaims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(value, parsedClaims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != m.method.Alg() {
			return nil, fmt.Errorf("auth: unexpected signing method %q", token.Method.Alg())
		}
		return m.secret, nil
	}, options...)
	if err != nil {
		return nil, fmt.Errorf("auth: validate JWT: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("auth: invalid JWT")
	}

	claims := make(Claims, len(parsedClaims))
	for key, value := range parsedClaims {
		claims[key] = value
	}
	return claims, nil
}

func signingMethod(algorithm string) (jwt.SigningMethod, error) {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "", "HS256":
		return jwt.SigningMethodHS256, nil
	case "HS384":
		return jwt.SigningMethodHS384, nil
	case "HS512":
		return jwt.SigningMethodHS512, nil
	default:
		return nil, fmt.Errorf("auth: unsupported JWT algorithm %q", algorithm)
	}
}

func tokenID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("auth: generate token ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

type claimsContextKey struct{}

// ContextWithClaims returns a child context containing validated claims.
func ContextWithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext returns all validated claims placed by auth middleware.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

// Claim returns a single claim value.
func Claim(ctx context.Context, name string) (any, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, false
	}
	value, ok := claims[name]
	return value, ok
}

// StringClaim returns a string claim.
func StringClaim(ctx context.Context, name string) (string, bool) {
	value, ok := Claim(ctx, name)
	if !ok {
		return "", false
	}
	result, ok := value.(string)
	return result, ok
}

// BoolClaim returns a boolean claim.
func BoolClaim(ctx context.Context, name string) (bool, bool) {
	value, ok := Claim(ctx, name)
	if !ok {
		return false, false
	}
	result, ok := value.(bool)
	return result, ok
}

// StringSliceClaim returns a string-array claim such as roles or permissions.
func StringSliceClaim(ctx context.Context, name string) ([]string, bool) {
	value, ok := Claim(ctx, name)
	if !ok {
		return nil, false
	}
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...), true
	case []any:
		result := make([]string, len(values))
		for index, value := range values {
			item, ok := value.(string)
			if !ok {
				return nil, false
			}
			result[index] = item
		}
		return result, true
	default:
		return nil, false
	}
}

// Int64Claim returns an integer claim, including numbers decoded from JSON.
func Int64Claim(ctx context.Context, name string) (int64, bool) {
	value, ok := Claim(ctx, name)
	if !ok {
		return 0, false
	}
	switch number := value.(type) {
	case int:
		return int64(number), true
	case int64:
		return number, true
	case float64:
		result := int64(number)
		return result, float64(result) == number
	case json.Number:
		result, err := number.Int64()
		return result, err == nil
	default:
		return 0, false
	}
}
