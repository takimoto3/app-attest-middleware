package adapter

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log/slog"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/plugin"
	"github.com/takimoto3/app-attest-middleware/requestid"
)

var (
	ErrAttestationRequired = errors.New("attestation required")
)

// AssertionServiceProvider creates a new AssertionService for verifying an assertion.
type AssertionServiceProvider func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService

// AssertionService defines the interface for verifying an assertion object.
type AssertionService interface {
	Verify(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error)
}

type AssertionAdapter interface {
	Verify(ctx context.Context, r *plugin.AssertionRequest) error
}

type assertionAdapter struct {
	logger *slog.Logger
	// Factory function for creating an AssertionService used to verify assertions.
	NewService AssertionServiceProvider
	plugin     plugin.AssertionPlugin
}

func NewAssertionAdapter(logger *slog.Logger, appID string, plugin plugin.AssertionPlugin) AssertionAdapter {
	return &assertionAdapter{
		logger: logger,
		plugin: plugin,
		NewService: func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService {
			return &attest.AssertionService{
				AppID:     appID,
				PublicKey: pubkey,
				Challenge: challenge,
				Counter:   counter,
			}
		},
	}
}

func (a *assertionAdapter) Verify(ctx context.Context, r *plugin.AssertionRequest) error {
	requestID := requestid.FromContext(ctx)
	logger := a.logger.With("request_id", requestID)
	logger.Debug("starting assertion verification")

	assertion, challenge, err := a.plugin.ParseRequest(ctx, r)
	if err != nil {
		logger.Error("failed to parse request", "err", err)
		return ErrBadRequest
	}
	pubkey, counter, err := a.plugin.PublicKeyAndCounter(ctx, r)
	if err != nil {
		logger.Error("failed to get public key and counter", "err", err)
		return ErrInternal
	}
	if pubkey == nil {
		// User has not completed Attestation yet
		// → redirect client to attestation flow
		return ErrAttestationRequired
	}
	assignedChallenge, err := a.plugin.AssignedChallenge(ctx, r)
	if err != nil {
		logger.Error("failed to get assigned challenge", "err", err)
		return ErrInternal
	}
	if assignedChallenge == "" {
		return ErrNewChallenge
	}
	service := a.NewService(assignedChallenge, pubkey, counter)
	cnt, err := service.Verify(assertion, challenge, r.Body)
	if err != nil {
		logger.Error("failed to verify assertion", "err", err)
		return ErrBadRequest
	}

	if err = a.plugin.UpdateCounter(ctx, r, cnt); err != nil {
		logger.Error("failed to store new counter", "err", err)
		return ErrInternal
	}

	return nil
}
