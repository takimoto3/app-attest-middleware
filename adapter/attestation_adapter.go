package adapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/plugin"
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

type AttestationAdapter interface {
	// NewChallenge generates a new challenge
	NewChallenge(ctx context.Context, r *plugin.AttestationRequest) (string, error)
	// Verify verifies the attestation request
	Verify(ctx context.Context, r *plugin.AttestationRequest) error
}

// attestationAdapter implements Adapter interface
type attestationAdapter struct {
	logger  *slog.Logger
	service AttestationService
	plugin  plugin.AttestationPlugin
}

// NewAttestationAdapter creates a new AttestationAdapter
func NewAttestationAdapter(logger *slog.Logger, service AttestationService, plugin plugin.AttestationPlugin) AttestationAdapter {
	return &attestationAdapter{
		logger:  logger,
		service: service,
		plugin:  plugin,
	}
}

// NewChallenge requests a new challenge from the plugin
func (a *attestationAdapter) NewChallenge(ctx context.Context, r *plugin.AttestationRequest) (string, error) {
	requestID := requestid.FromContext(ctx)
	logger := a.logger.With("request_id", requestID)
	logger.Debug("requesting new challenge")

	challenge, err := a.plugin.NewChallenge(ctx, r)
	if err != nil {
		logger.Error(" failed to generate new challenge", "err", err)
		return "", fmt.Errorf("%w: failed to generate new challenge: %v", ErrInternal, err)
	}
	return challenge, nil
}

// Verify performs attestation verification
func (a *attestationAdapter) Verify(ctx context.Context, r *plugin.AttestationRequest) error {
	requestID := requestid.FromContext(ctx)
	logger := a.logger.With("request_id", requestID)
	logger.Debug("starting attestation verification")

	// Extract attestation data from plugin
	attestObj, clientDataHash, keyID, err := a.plugin.ExtractData(ctx, r)
	if err != nil {
		logger.Error("failed to parse request", "err", err)
		return fmt.Errorf("%w: failed to parse request: %v", ErrBadRequest, err)
	}

	// Check if challenge was assigned
	assigned, err := a.plugin.IsChallengeAssigned(ctx, r)
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
	if err := a.plugin.StoreResult(ctx, r); err != nil {
		logger.Error("failed to store attestation result", "err", err)
		return fmt.Errorf("%w: failed to store result: %v", ErrInternal, err)
	}
	logger.Info("attestation result stored")

	return nil
}
