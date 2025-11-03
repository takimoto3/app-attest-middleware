package middleware

import (
	"context"
	"crypto/ecdsa"

	attest "github.com/takimoto3/app-attest"
)

// AdapterPlugin defines the application-specific operations required
// by the AssertionMiddleware to complete the App Attest assertion flow.
//
// Implementations handle challenge management, request parsing,
// redirecting clients that lack valid attestations, and updating counters
// after successful verification.
type AdapterPlugin interface {
	// AssignedChallenge returns the assigned challenge.
	AssignedChallenge(ctx context.Context, r *Request) (string, error)
	// ParseRequest parses the incoming request and returns the assertion object and challenge.
	ParseRequest(ctx context.Context, r *Request) (*attest.AssertionObject, string, error)
	// PublicKeyAndCounter returns the stored public key and counter.
	PublicKeyAndCounter(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error)
	// UpdateCounter saves the latest assertion counter.
	UpdateCounter(ctx context.Context, r *Request, counter uint32) error
}
