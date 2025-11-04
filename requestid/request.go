package requestid

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
)

type contextKey struct{}

var requestIDKey = contextKey{}

type Generator interface {
	NextID() (string, error)
}

var generator atomic.Value // holds Generator

func UseGenerator(gen Generator) {
	generator.Store(gen)
}

func currentGenerator() Generator {
	if gen, ok := generator.Load().(Generator); ok {
		return gen
	}
	return nil
}

func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func EnsureRequest(r *http.Request) (*http.Request, string, error) {
	gen := currentGenerator()
	if gen == nil {
		return nil, "", fmt.Errorf("generator not initialized")
	}
	id := r.Header.Get("X-Request-ID")
	if id == "" {
		next, err := gen.NextID()
		if err != nil {
			return nil, "", err
		}
		id = next
	}
	ctx := context.WithValue(r.Context(), requestIDKey, id)

	return r.WithContext(ctx), id, nil
}
