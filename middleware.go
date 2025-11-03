package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/takimoto3/app-attest-middleware/requestid"
)

type Config struct {
	BodyLimit       int64
	AttestationURL  string
	NewChallengeURL string
}

type AssertionMiddleware struct {
	logger  *slog.Logger
	adapter Adapter
	config  Config
}

func NewAssertionMiddleware(logger *slog.Logger, config Config, adapter Adapter) *AssertionMiddleware {
	m := &AssertionMiddleware{
		logger:  logger,
		adapter: adapter,
		config:  config,
	}
	if m.config.BodyLimit == 0 {
		m.config.BodyLimit = 10 << 20 // 10MB
	}
	if logger == nil {
		m.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return m
}

func (m *AssertionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r, requestID, err := requestid.EnsureRequest(r)
		if err != nil {
			m.logger.Error("failed to generate request ID", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logger := m.logger.With("request_id", requestID)
		var body []byte
		if r.Body != nil {
			body, err = io.ReadAll(io.LimitReader(r.Body, m.config.BodyLimit+1))
			if err != nil {
				logger.Error("failed to read request body", "err", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if int64(len(body)) > m.config.BodyLimit {
				logger.Warn("request body exceeded limit",
					"limit_bytes", m.config.BodyLimit,
					"actual_bytes", len(body),
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
				)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}
		req := &Request{
			Request: r,
			Body:    body,
		}

		err = m.adapter.Verify(r.Context(), req)
		if err != nil {
			switch err {
			case ErrAttestationRequired:
				logger.Info("redirecting to attestation", "url", m.config.AttestationURL)
				http.Redirect(w, r, m.config.AttestationURL, http.StatusSeeOther)
			case ErrNewChallenge:
				logger.Info("redirecting to new challenge", "url", m.config.NewChallengeURL)
				redirect := m.config.NewChallengeURL
				if redirect == "" {
					redirect = r.Header.Get("Referer")
					logger.Info("fallback to Referer for redirect", "referer", redirect)
					if redirect == "" {
						redirect = "/"
					}
				}
				http.Redirect(w, r, redirect, http.StatusSeeOther)
			case ErrBadRequest:
				logger.Warn("bad request in assertion middleware")
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case ErrInternal:
				logger.Error("internal error in assertion middleware")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			default:
				logger.Error("unexpected error in assertion middleware", "err", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		logger.Debug("request passed assertion middleware")
		next.ServeHTTP(w, r)
	})
}
