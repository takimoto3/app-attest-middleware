package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/takimoto3/app-attest-middleware/adapter"
	"github.com/takimoto3/app-attest-middleware/plugin"
	"github.com/takimoto3/app-attest-middleware/requestid"
)

// VerifyHooks defines hooks for the Verify handler.
// Setup: pre-processing (cannot write to response)
// Success: called on successful verification
// Failed: called on failure (default implementation is just an example and can be overridden)
type VerifyHooks struct {
	Setup   func(r *http.Request)
	Success func(w http.ResponseWriter, r *http.Request)
	Failed  func(w http.ResponseWriter, r *http.Request, err error)
}

// NewChallengeHooks defines hooks for the NewChallenge handler.
// Setup: pre-processing (cannot write to response)
// Success: called on successful challenge creation
// Failed: called on failure (default implementation is just an example and can be overridden)
type NewChallengeHooks struct {
	Setup   func(r *http.Request)
	Success func(w http.ResponseWriter, r *http.Request, challenge string)
	Failed  func(w http.ResponseWriter, r *http.Request, err error)
}

// AppAttestHandler is an HTTP handler for App Attest verification.
// VerifyHooks and NewChallengeHooks allow customizing success, failure, and pre-processing behavior.
type AppAttestHandler struct {
	logger  *slog.Logger
	adapter adapter.AttestationAdapter
	VerifyHooks
	NewChallengeHooks
}

// NewAppAttestHandler creates a default AppAttestHandler.
// Default Failed hooks are just examples and can be overridden.
func NewAppAttestHandler(logger *slog.Logger, attestAdapter adapter.AttestationAdapter) *AppAttestHandler {
	return &AppAttestHandler{
		logger:  logger,
		adapter: attestAdapter,
		VerifyHooks: VerifyHooks{
			Setup: func(r *http.Request) {},
			Success: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			Failed: func(w http.ResponseWriter, r *http.Request, err error) {
				if errors.Is(err, adapter.ErrBadRequest) {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			},
		},
		NewChallengeHooks: NewChallengeHooks{
			Setup: func(r *http.Request) {},
			Success: func(w http.ResponseWriter, r *http.Request, challenge string) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
			},
			Failed: func(w http.ResponseWriter, r *http.Request, err error) {
				if errors.Is(err, adapter.ErrBadRequest) {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			},
		},
	}
}

func (h *AppAttestHandler) Verify(w http.ResponseWriter, r *http.Request) {
	r, logger, err := h.getLogger(r)
	if err != nil {
		h.logger.Error("failed to generate request ID", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.VerifyHooks.Setup(r)
	err = h.adapter.Verify(r.Context(), &plugin.AttestationRequest{Request: r})
	if err != nil {
		if errors.Is(err, adapter.ErrNewChallenge) {
			h.NewChallenge(w, r)
			return
		}
		logger.Error("verification failed", "err", err)
		h.VerifyHooks.Failed(w, r, err)
		return
	}

	logger.Info("verification succeeded")
	h.VerifyHooks.Success(w, r)
}

func (h *AppAttestHandler) NewChallenge(w http.ResponseWriter, r *http.Request) {
	r, logger, err := h.getLogger(r)
	if err != nil {
		h.logger.Error("failed to generate request ID", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.NewChallengeHooks.Setup(r)
	challenge, err := h.adapter.NewChallenge(r.Context(), &plugin.AttestationRequest{Request: r})
	if err != nil {
		logger.Error("new challenge failed", "err", err)
		h.NewChallengeHooks.Failed(w, r, err)
		return
	}

	logger.Info("new challenge succeeded")
	h.NewChallengeHooks.Success(w, r, challenge)
}

func (h *AppAttestHandler) getLogger(r *http.Request) (*http.Request, *slog.Logger, error) {
	r, requestID, err := requestid.EnsureRequest(r)
	if err != nil {
		return r, nil, err
	}
	return r, h.logger.With("request_id", requestID), nil
}
