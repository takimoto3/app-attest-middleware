package adapter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/plugin"
)

type mockPluginFunc struct {
	extractData         func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error)
	isChallengeAssigned func(ctx context.Context, r *plugin.AttestationRequest) (bool, error)
	newChallenge        func(ctx context.Context, r *plugin.AttestationRequest) (string, error)
	storeResult         func(ctx context.Context, r *plugin.AttestationRequest) error
}

func (m *mockPluginFunc) ExtractData(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
	return m.extractData(ctx, r)
}
func (m *mockPluginFunc) IsChallengeAssigned(ctx context.Context, r *plugin.AttestationRequest) (bool, error) {
	return m.isChallengeAssigned(ctx, r)
}
func (m *mockPluginFunc) NewChallenge(ctx context.Context, r *plugin.AttestationRequest) (string, error) {
	if m.newChallenge == nil {
		return "mock-challenge", nil
	}
	return m.newChallenge(ctx, r)
}
func (m *mockPluginFunc) StoreResult(ctx context.Context, r *plugin.AttestationRequest) error {
	if m.storeResult == nil {
		return nil
	}
	return m.storeResult(ctx, r)
}

type mockServiceFunc struct {
	verify func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
}

func (m *mockServiceFunc) Verify(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
	if m.verify == nil {
		return &attest.Result{}, nil
	}
	return m.verify(attestObj, clientDataHash, keyID)
}

func TestAttestationAdapter_Verify(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := map[string]struct {
		extractData         func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error)
		isChallengeAssigned func(ctx context.Context, r *plugin.AttestationRequest) (bool, error)
		verify              func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
		storeResult         func(ctx context.Context, r *plugin.AttestationRequest) error
		wantErr             error
	}{
		"no challenge assigned": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) { return false, nil },
			wantErr:             ErrNewChallenge,
		},
		"verify success": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			storeResult: func(ctx context.Context, r *plugin.AttestationRequest) error { return nil },
			wantErr:     nil,
		},
		"verify service error": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return nil, errors.New("verify failed")
			},
			storeResult: nil,
			wantErr:     ErrBadRequest,
		},
		"extract data error": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return nil, nil, nil, errors.New("parse error")
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) { return true, nil },
			wantErr:             ErrBadRequest,
		},
		"isChallengeAssigned error": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) {
				return false, errors.New("check error")
			},
			wantErr: ErrInternal,
		},
		"store result error": {
			extractData: func(ctx context.Context, r *plugin.AttestationRequest) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(ctx context.Context, r *plugin.AttestationRequest) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			storeResult: func(ctx context.Context, r *plugin.AttestationRequest) error { return errors.New("store failed") },
			wantErr:     ErrInternal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			a := &attestationAdapter{
				plugin: &mockPluginFunc{
					extractData:         tt.extractData,
					isChallengeAssigned: tt.isChallengeAssigned,
					newChallenge:        func(ctx context.Context, r *plugin.AttestationRequest) (string, error) { return "mock-challenge", nil },
					storeResult:         tt.storeResult,
				},
				service: &mockServiceFunc{
					verify: tt.verify,
				},
				logger: logger,
			}

			req := &plugin.AttestationRequest{}
			err := a.Verify(context.Background(), req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestAttestationAdapter_NewChallenge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	tests := map[string]struct {
		newChallenge func(ctx context.Context, r *plugin.AttestationRequest) (string, error)
		wantErr      error
	}{
		"success": {
			newChallenge: func(ctx context.Context, r *plugin.AttestationRequest) (string, error) { return "mock-challenge", nil },
			wantErr:      nil,
		},
		"plugin error": {
			newChallenge: func(ctx context.Context, r *plugin.AttestationRequest) (string, error) {
				return "", errors.New("plugin failed")
			},
			wantErr: ErrInternal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := &attestationAdapter{
				plugin: &mockPluginFunc{
					newChallenge: tt.newChallenge,
				},
				logger: logger,
			}

			req := &plugin.AttestationRequest{}
			challenge, err := a.NewChallenge(context.Background(), req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if challenge == "" {
					t.Errorf("expected non-empty challenge")
				}
			}
		})
	}
}
