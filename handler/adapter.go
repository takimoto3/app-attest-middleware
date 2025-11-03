package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/requestid"
)

var (
	// ErrNewChallenge indicates no challenge is assigned yet
	ErrNewChallenge = errors.New("no challenge assigned")
	// ErrBadRequest indicates the request is invalid
	ErrBadRequest = errors.New("bad request")
	// ErrInternal indicates an internal server error
	ErrInternal = errors.New("internal error")
)

// AttestationService defines the interface for verifying attestation
type AttestationService interface {
	Verify(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
}

var _ Adapter = &AttestationAdapter{}

type Adapter interface {
	// NewChallenge generates a new challenge
	NewChallenge(ctx context.Context, r *Request) (string, error)
	// Verify verifies the attestation request
	Verify(ctx context.Context, r *Request) error
}

// Request wraps the request data and verification result
type Request struct {
	Request any
	Result  *attest.Result
	Object  any
}

// AttestationAdapter implements Adapter interface
type AttestationAdapter struct {
	logger  *slog.Logger
	service AttestationService
	plugin  AdapterPlugin
}

// NewChallenge requests a new challenge from the plugin
func (a *AttestationAdapter) NewChallenge(ctx context.Context, r *Request) (string, error) {
	requestID := requestid.FromContext(ctx)
	logger := a.logger.With("request_id", requestID)
	logger.Debug("requesting new challenge")

	challenge, err := a.plugin.NewChallenge(r)
	if err != nil {
		logger.Error(" failed to generate new challenge", "err", err)
		return "", fmt.Errorf("%w: failed to generate new challenge: %v", ErrInternal, err)
	}
	return challenge, nil
}

// Verify performs attestation verification
func (a *AttestationAdapter) Verify(ctx context.Context, r *Request) error {
	requestID := requestid.FromContext(ctx)
	logger := a.logger.With("request_id", requestID)
	logger.Debug("starting attestation verification")

	// Extract attestation data from plugin
	attestObj, clientDataHash, keyID, err := a.plugin.ExtractData(r)
	if err != nil {
		logger.Error("failed to parse request", "err", err)
		return fmt.Errorf("%w: failed to parse request: %v", ErrBadRequest, err)
	}

	// Check if challenge was assigned
	assigned, err := a.plugin.IsChallengeAssigned(r)
	if err != nil {
		logger.Error("failed to check challenge assignment", "err", err)
		return fmt.Errorf("%w: failed to check challenge: %v", ErrInternal, err)
	}
	if !assigned {
		logger.Info("no challenge assigned, new challenge needed")
		return ErrNewChallenge
	}

	// Verify attestation with service
	result, err := a.service.Verify(attestObj, clientDataHash, keyID)
	if err != nil {
		logger.Error("failed to verify attestation", "keyID", string(keyID), "err", err)
		return fmt.Errorf("%w: failed to verify attestation: %v", ErrBadRequest, err)
	}
	r.Result = result
	logger.Debug("attestation verified successfully", "keyID", string(keyID))

	// Store verification result via plugin
	if err := a.plugin.StoreResult(r); err != nil {
		logger.Error("failed to store attestation result", "err", err)
		return fmt.Errorf("%w: failed to store result: %v", ErrInternal, err)
	}
	logger.Info("attestation result stored")

	return nil
}
