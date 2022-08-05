package middleware

import (
	"crypto/ecdsa"
	"net/http"

	attest "github.com/takimoto3/app-attest"
)

type AssertionPlugin interface {
	// GetAssignedChallenge returns the assigned challenge.
	GetAssignedChallenge(r *http.Request) (string, error)
	// ResponseNewChallenge sets the newly generated challenge to the http response.
	ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error
	// ParseRequest returns http.Request, AssertionObject and challenge from the arguments.
	ParseRequest(r *http.Request, requestBody []byte) (*http.Request, *attest.AssertionObject, string, error)
	// RedirectToAttestation redirects to the URL where you want to save the attestation.
	RedirectToAttestation(w http.ResponseWriter, r *http.Request)
	// GetPublicKeyAndCounter returns the stored public key and counter.
	GetPublicKeyAndCounter(r *http.Request) (*ecdsa.PublicKey, uint32, error)
	// StoreNewCounter save the assertion counter.
	StoreNewCounter(r *http.Request, counter uint32) error
}
