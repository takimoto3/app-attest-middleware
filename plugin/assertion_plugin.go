package plugin

import (
	"context"
	"crypto/ecdsa"

	attest "github.com/takimoto3/app-attest"
)

type AssertionRequest struct {
	// Request is the original request object, typically *http.Request.
	Request any
	Body    []byte
	Object  any
}

// AssertionPlugin defines the application-specific operations required
// by the AssertionMiddleware to complete the App Attest assertion flow.
//
// Implementations handle challenge management, request parsing,
// redirecting clients that lack valid attestations, and updating counters
// after successful verification.
type AssertionPlugin interface {
	// AssignedChallenge returns the assigned challenge.
	AssignedChallenge(ctx context.Context, r *AssertionRequest) (string, error)
	// ParseRequest parses the incoming request and returns the assertion object and challenge.
	ParseRequest(ctx context.Context, r *AssertionRequest) (*attest.AssertionObject, string, error)
	// PublicKeyAndCounter returns the stored public key and counter.
	PublicKeyAndCounter(ctx context.Context, r *AssertionRequest) (*ecdsa.PublicKey, uint32, error)
	// UpdateCounter saves the latest assertion counter.
	UpdateCounter(ctx context.Context, r *AssertionRequest, counter uint32) error
}
