package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takimoto3/app-attest-middleware/adapter"
	"github.com/takimoto3/app-attest-middleware/plugin"
	"github.com/takimoto3/app-attest-middleware/requestid"
)

type errReader struct{}

func (*errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

type mockAdapter struct {
	verifyFunc func(ctx context.Context, req *plugin.AssertionRequest) error
}

func (m *mockAdapter) Verify(ctx context.Context, req *plugin.AssertionRequest) error {
	return m.verifyFunc(ctx, req)
}

type mockGenerator struct {
	ID  string
	Err error
}

func (m *mockGenerator) NextID() (string, error) {
	return m.ID, m.Err
}

func TestAssertionMiddleware_Handler(t *testing.T) {
	requestid.UseGenerator(&mockGenerator{ID: "generated_id"})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := map[string]struct {
		adapterErr    error
		config        Config
		refererHeader string
		body          string
		forceReadErr  bool
		wantStatus    int
		wantLocation  string
	}{
		"successful assertion": {
			adapterErr: nil,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "ok",
			wantStatus: http.StatusOK,
		},
		"attestation required redirect": {
			adapterErr: adapter.ErrAttestationRequired,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:         "ok",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/attest",
		},
		"new challenge redirect with config": {
			adapterErr: adapter.ErrNewChallenge,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:         "ok",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/challenge",
		},
		"new challenge redirect with referer fallback": {
			adapterErr: adapter.ErrNewChallenge,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "",
				BodyLimit:       1024,
			},
			refererHeader: "/referer",
			body:          "ok",
			wantStatus:    http.StatusSeeOther,
			wantLocation:  "/referer",
		},
		"new challenge redirect with empty urls": {
			adapterErr: adapter.ErrNewChallenge,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "",
				BodyLimit:       1024,
			},
			body:         "ok",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/", // fallback to "/" when both NewChallengeURL and Referer are empty
		},
		"bad request": {
			adapterErr: adapter.ErrBadRequest,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "ok",
			wantStatus: http.StatusBadRequest,
		},
		"internal error": {
			adapterErr: adapter.ErrInternal,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "ok",
			wantStatus: http.StatusInternalServerError,
		},
		"unexpected error": {
			adapterErr: errors.New("unexpected"),
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "ok",
			wantStatus: http.StatusInternalServerError,
		},
		"body exceeds limit": {
			adapterErr: nil,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1,
			},
			body:       "ab",
			wantStatus: http.StatusBadRequest,
		},
		"io.ReadAll error": {
			adapterErr: nil,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:         "ok",
			forceReadErr: true,
			wantStatus:   http.StatusBadRequest,
		},
		"no body request": {
			adapterErr: nil,
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "",
			wantStatus: http.StatusOK,
		},
		"wrapped attestation required redirect (fails with current switch)": {
			adapterErr: fmt.Errorf("some context: %w", adapter.ErrAttestationRequired),
			config: Config{
				AttestationURL:  "/attest",
				NewChallengeURL: "/challenge",
				BodyLimit:       1024,
			},
			body:       "ok",
			wantStatus: http.StatusSeeOther, // Expecting redirect if errors.Is were used
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			adapter := &mockAdapter{
				verifyFunc: func(ctx context.Context, req *plugin.AssertionRequest) error {
					return tt.adapterErr
				},
			}

			mw := NewAssertionMiddleware(logger, tt.config, adapter)

			calledNext := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledNext = true
				w.WriteHeader(http.StatusOK)
			})

			var req *http.Request
			if tt.forceReadErr {
				req = httptest.NewRequest(http.MethodPost, "/", &errReader{})
			} else {
				req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			}

			if tt.refererHeader != "" {
				req.Header.Set("Referer", tt.refererHeader)
			}

			w := httptest.NewRecorder()
			handler := mw.Use(next)
			handler.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			if tt.adapterErr == nil && !tt.forceReadErr && tt.config.BodyLimit >= int64(len(tt.body)) && !calledNext {
				t.Errorf("expected next handler to be called")
			}

			if res.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", res.StatusCode, tt.wantStatus)
			}

			if tt.wantLocation != "" {
				loc := res.Header.Get("Location")
				if loc != tt.wantLocation {
					t.Errorf("got Location %q, want %q", loc, tt.wantLocation)
				}
			}
		})
	}
}

func TestAssertionMiddleware_Initialization(t *testing.T) {
	adapter := &mockAdapter{
		verifyFunc: func(ctx context.Context, req *plugin.AssertionRequest) error {
			return nil
		},
	}

	cfg := Config{
		AttestationURL:  "/attest",
		NewChallengeURL: "/challenge",
	}

	mw := NewAssertionMiddleware(nil, cfg, adapter)

	if mw.logger == nil {
		t.Fatal("expected default logger to be set when logger is nil")
	}

	if mw.config.BodyLimit != 10<<20 {
		t.Fatalf("expected BodyLimit to be default 10MB, got %d", mw.config.BodyLimit)
	}

	calledNext := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("ok"))
	w := httptest.NewRecorder()

	mw.Use(next).ServeHTTP(w, req)

	if !calledNext {
		t.Fatal("next handler should be called")
	}
}
