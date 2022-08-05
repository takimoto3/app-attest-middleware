package handler

import (
	"net/http"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/logger"
)

type AttestationService interface {
	Verify(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
}

type AttestationHandler struct {
	logger.Logger
	AttestationService
	AttestationPlugin
}

func (h *AttestationHandler) NewChallenge(w http.ResponseWriter, r *http.Request) {
	h.Logger.SetContext(r.Context())
	if err := h.ResponseNewChallenge(w, r); err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *AttestationHandler) VerifyAttestation(w http.ResponseWriter, r *http.Request) {
	h.Logger.SetContext(r.Context())
	r, attestObj, clientDataHash, keyID, err := h.ParseRequest(r)
	if err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	assginedChallenge, err := h.GetAssignedChallenge(r)
	if err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if assginedChallenge == "" {
		if err = h.ResponseNewChallenge(w, r); err != nil {
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

	if err = h.StoreResult(r, result); err != nil {
		h.Logger.Errorf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
