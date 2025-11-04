package plugin

import (
	"context"

	attest "github.com/takimoto3/app-attest"
)

// AttestationRequest wraps the request data and verification result
type AttestationRequest struct {
	// Request is the original request object, typically *http.Request.
	Request any
	Result  *attest.Result
	Object  any
}

// AttestationPlugin defines application-specific hooks used by
// the AttestationMiddleware to handle the App Attest attestation flow.
type AttestationPlugin interface {
	// ExtractData parses the request and returns:
	// - The attestation object
	// - The clientDataHash
	// - The keyID
	ExtractData(ctx context.Context, r *AttestationRequest) (*attest.AttestationObject, []byte, []byte, error)

	// IsChallengeAssigned reports whether a challenge is already assigned
	// for the current request or session.
	IsChallengeAssigned(ctx context.Context, r *AttestationRequest) (bool, error)

	// NewChallenge creates and stores a new challenge for the client.
	NewChallenge(ctx context.Context, r *AttestationRequest) (string, error)

	// StoreResult persists the attestation result after successful verification.
	StoreResult(ctx context.Context, r *AttestationRequest) error
}
