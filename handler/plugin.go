package handler

import (
	"net/http"

	attest "github.com/takimoto3/app-attest"
)

type AttestationPlugin interface {
	// GetAssignedChallenge returns the assigned challenge.
	GetAssignedChallenge(r *http.Request) (string, error)
	// ResponseNewChallenge sets the newly generated challenge to the http response.
	ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error
	// ParseRequest returns http.Request, AttstationObject, the hash value of challenge, and KeyID from the argument http.Request.
	ParseRequest(r *http.Request) (*http.Request, *attest.AttestationObject, []byte, []byte, error)
	// StoreResult stores the data of the argument attest.Result (PublicKey, Receipt, etc.).
	StoreResult(r *http.Request, result *attest.Result) error
}
