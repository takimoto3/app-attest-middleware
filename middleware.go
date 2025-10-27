package middleware

import (
	"bytes"
	"crypto/ecdsa"
	"io"
	"net/http"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/logger"
)

// AssertionServiceProvider creates a new AssertionService for verifying an assertion.
type AssertionServiceProvider func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService

// AssertionService defines the interface for verifying an assertion object.
type AssertionService interface {
	Verify(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error)
}

// AssertionMiddleware handles App Attest assertion verification in HTTP request flow.
type AssertionMiddleware struct {
	logger logger.Logger
	appID  string
	plugin AssertionPlugin

	// Factory function for creating an AssertionService used to verify assertions.
	NewService AssertionServiceProvider
}

// NewMiddleware creates a new AssertionMiddleware bound to the given appID and plugin.
func NewMiddleware(logger logger.Logger, appID string, plugin AssertionPlugin) *AssertionMiddleware {
	return &AssertionMiddleware{
		logger: logger,
		appID:  appID,
		plugin: plugin,
		NewService: func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService {
			return &attest.AssertionService{
				AppID:     appID,
				PublicKey: pubkey,
				Challenge: challenge,
				Counter:   counter,
			}
		},
	}
}

// AttestAssertWith wraps an HTTP handler with assertion verification logic.
func (m *AssertionMiddleware) AttestAssertWith(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.logger.SetContext(r.Context())
		var err error
		var requestBody []byte
		if r.Body != nil {
			// Read and preserve request body since it may be parsed multiple times downstream.
			requestBody, err = io.ReadAll(r.Body)
			if err != nil {
				m.logger.Errorf("failed to read body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		r, assertion, challenge, err := m.plugin.ParseRequest(r, requestBody)
		if err != nil {
			m.logger.Errorf("%s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pubkey, counter, err := m.plugin.GetPublicKeyAndCounter(r)
		if err != nil {
			m.logger.Errorf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if pubkey == nil {
			// User has not completed Attestation yet
			// → redirect client to attestation flow
			m.plugin.RedirectToAttestation(w, r)
			return
		}
		assignedChallenge, err := m.plugin.GetAssignedChallenge(r)
		if err != nil {
			m.logger.Errorf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if assignedChallenge == "" {
			if err := m.plugin.ResponseNewChallenge(w, r); err != nil {
				m.logger.Errorf("%s", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		service := m.NewService(assignedChallenge, pubkey, counter)
		cnt, err := service.Verify(assertion, challenge, requestBody)
		if err != nil {
			m.logger.Errorf("%s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err = m.plugin.StoreNewCounter(r, cnt); err != nil {
			m.logger.Errorf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
