package requestid

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
)

const Header = "X-Request-ID"

type contextKey struct{}

type Generator interface {
	New() string
}

type RandomGenerator struct{}

func (RandomGenerator) New() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		// crypto/rand failures are process-level failures in practice. Returning
		// a valid opaque ID keeps the request middleware total if that happens.
		return "req_unavailable"
	}
	return "req_" + base64.RawURLEncoding.EncodeToString(bytes)
}

func New() string {
	return (RandomGenerator{}).New()
}

func WithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKey{}).(string)
	return id, ok && valid(id)
}

func Ensure(incoming string, generator Generator) string {
	if valid(incoming) {
		return incoming
	}
	if generator == nil {
		return New()
	}
	return generator.New()
}

func Middleware(next http.Handler) http.Handler {
	return MiddlewareWithGenerator(next, RandomGenerator{})
}

func MiddlewareWithGenerator(next http.Handler, generator Generator) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id := Ensure(strings.TrimSpace(request.Header.Get(Header)), generator)
		request = request.WithContext(WithContext(request.Context(), id))
		writer.Header().Set(Header, id)
		next.ServeHTTP(writer, request)
	})
}

func valid(id string) bool {
	if len(id) < len("req_")+1 || !strings.HasPrefix(id, "req_") {
		return false
	}
	for _, character := range id[len("req_"):] {
		if !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}
