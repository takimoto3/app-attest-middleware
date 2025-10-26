package handler

import (
	"net/http"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/logger"
)

// AttestationService defines the interface for verifying an App Attest
// attestation object and returning a verification result.
type AttestationService interface {
	Verify(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
}

// AttestationHandler provides HTTP endpoints for handling App Attest
// attestation flow including challenge issuance and verification.
type AttestationHandler struct {
	logger.Logger
	AttestationService
	AttestationPlugin
}

// NewChallenge issues a new attestation challenge to the client.
func (h *AttestationHandler) NewChallenge(w http.ResponseWriter, r *http.Request) {
	h.Logger.SetContext(r.Context())
	if err := h.ResponseNewChallenge(w, r); err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// VerifyAttestation verifies the attestation object received from the client.
// If verification succeeds, the result is persisted using the plugin.
func (h *AttestationHandler) VerifyAttestation(w http.ResponseWriter, r *http.Request) {
	h.Logger.SetContext(r.Context())
	req, attestObj, clientDataHash, keyID, err := h.ParseRequest(r)
	if err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	assignedChallenge, err := h.GetAssignedChallenge(req)
	if err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if assignedChallenge == "" {
		if err = h.ResponseNewChallenge(w, req); err != nil {
			h.Logger.Errorf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	result, err := h.AttestationService.Verify(attestObj, clientDataHash, keyID)
	if err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = h.StoreResult(req, result); err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
