package handler

import (
	"context"

	attest "github.com/takimoto3/app-attest"
)

type AdapterPlugin interface {
	// client data hash, and key ID.
	ExtractData(r *Request) (*attest.AttestationObject, []byte, []byte, error)
	IsChallengeAssigned(r *Request) (bool, error)
	NewChallenge(r *Request) (string, error)
	AssignedChallenge(ctx context.Context, sessionID string) (string, error)
	StoreResult(r *Request) error
}
