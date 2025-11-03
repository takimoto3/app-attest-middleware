package handler

import (
	"context"

	attest "github.com/takimoto3/app-attest"
)

// AdapterPlugin defines application-specific hooks used by
// the AttestationMiddleware to handle the App Attest attestation flow.
type AdapterPlugin interface {
	// ExtractData parses the request and returns:
	// - The attestation object
	// - The clientDataHash
	// - The keyID
	ExtractData(ctx context.Context, r *Request) (*attest.AttestationObject, []byte, []byte, error)

	// IsChallengeAssigned reports whether a challenge is already assigned
	// for the current request or session.
	IsChallengeAssigned(ctx context.Context, r *Request) (bool, error)

	// NewChallenge creates and stores a new challenge for the client.
	NewChallenge(ctx context.Context, r *Request) (string, error)

	// StoreResult persists the attestation result after successful verification.
	StoreResult(ctx context.Context, r *Request) error
}
