package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	attest "github.com/takimoto3/app-attest"
)

type mockPluginFunc struct {
	extractData         func(r *Request) (*attest.AttestationObject, []byte, []byte, error)
	isChallengeAssigned func(r *Request) (bool, error)
	newChallenge        func(r *Request) (string, error)
	assignedChallenge   func(ctx context.Context, sessionID string) (string, error)
	storeResult         func(r *Request) error
}

func (m *mockPluginFunc) ExtractData(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
	return m.extractData(r)
}
func (m *mockPluginFunc) IsChallengeAssigned(r *Request) (bool, error) {
	return m.isChallengeAssigned(r)
}
func (m *mockPluginFunc) NewChallenge(r *Request) (string, error) {
	if m.newChallenge == nil {
		return "mock-challenge", nil
	}
	return m.newChallenge(r)
}
func (m *mockPluginFunc) AssignedChallenge(ctx context.Context, sessionID string) (string, error) {
	if m.assignedChallenge == nil {
		return "", nil
	}
	return m.assignedChallenge(ctx, sessionID)
}
func (m *mockPluginFunc) StoreResult(r *Request) error {
	if m.storeResult == nil {
		return nil
	}
	return m.storeResult(r)
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
		extractData         func(r *Request) (*attest.AttestationObject, []byte, []byte, error)
		isChallengeAssigned func(r *Request) (bool, error)
		verify              func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error)
		storeResult         func(r *Request) error
		wantErr             error
	}{
		"no challenge assigned": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return false, nil },
			wantErr:             ErrNewChallenge,
		},
		"verify success": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			storeResult: func(r *Request) error { return nil },
			wantErr:     nil,
		},
		"verify service error": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return nil, errors.New("verify failed")
			},
			storeResult: nil,
			wantErr:     ErrBadRequest,
		},
		"extract data error": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return nil, nil, nil, errors.New("parse error")
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return true, nil },
			wantErr:             ErrBadRequest,
		},
		"isChallengeAssigned error": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return false, errors.New("check error") },
			wantErr:             ErrInternal,
		},
		"store result error": {
			extractData: func(r *Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("hash"), []byte("key"), nil
			},
			isChallengeAssigned: func(r *Request) (bool, error) { return true, nil },
			verify: func(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			storeResult: func(r *Request) error { return errors.New("store failed") },
			wantErr:     ErrInternal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			plugin := &mockPluginFunc{
				extractData:         tt.extractData,
				isChallengeAssigned: tt.isChallengeAssigned,
				newChallenge:        func(r *Request) (string, error) { return "mock-challenge", nil },
				assignedChallenge:   func(ctx context.Context, sessionID string) (string, error) { return "", nil },
				storeResult:         tt.storeResult,
			}
			service := &mockServiceFunc{
				verify: tt.verify,
			}

			a := &AttestationAdapter{
				plugin:  plugin,
				service: service,
				logger:  logger,
			}

			req := &Request{}
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
		newChallenge func(r *Request) (string, error)
		wantErr      error
	}{
		"success": {
			newChallenge: func(r *Request) (string, error) { return "mock-challenge", nil },
			wantErr:      nil,
		},
		"plugin error": {
			newChallenge: func(r *Request) (string, error) { return "", errors.New("plugin failed") },
			wantErr:      ErrInternal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			plugin := &mockPluginFunc{
				newChallenge: tt.newChallenge,
			}
			a := &AttestationAdapter{
				plugin: plugin,
				logger: logger,
			}

			req := &Request{}
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
