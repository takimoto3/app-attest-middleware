package requestid

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/sony/sonyflake/v2"
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

func UseSnowFlake(st sonyflake.Settings) error {
	sf, err := sonyflake.New(st)
	if err != nil {
		return fmt.Errorf("failed to create sonyflake: %w", err)
	}
	UseGenerator(&sonyFlakeGenerator{sf: sf})

	return nil
}

type sonyFlakeGenerator struct {
	sf *sonyflake.Sonyflake
}

func (g *sonyFlakeGenerator) NextID() (string, error) {
	id, err := g.sf.NextID()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
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
