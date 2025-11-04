package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	attest "github.com/takimoto3/app-attest"
)

type mockPlugin struct {
	ParseRequestFn        func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error)
	PublicKeyAndCounterFn func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error)
	AssignedChallengeFn   func(ctx context.Context, r *Request) (string, error)
	UpdateCounterFn       func(ctx context.Context, r *Request, cnt uint32) error
}

func (m *mockPlugin) ParseRequest(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
	if m.ParseRequestFn != nil {
		return m.ParseRequestFn(ctx, r)
	}
	return nil, "", nil
}

func (m *mockPlugin) PublicKeyAndCounter(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
	if m.PublicKeyAndCounterFn != nil {
		return m.PublicKeyAndCounterFn(ctx, r)
	}
	return nil, 0, nil
}

func (m *mockPlugin) AssignedChallenge(ctx context.Context, r *Request) (string, error) {
	if m.AssignedChallengeFn != nil {
		return m.AssignedChallengeFn(ctx, r)
	}
	return "", nil
}

func (m *mockPlugin) UpdateCounter(ctx context.Context, r *Request, cnt uint32) error {
	if m.UpdateCounterFn != nil {
		return m.UpdateCounterFn(ctx, r, cnt)
	}
	return nil
}

type mockAssertionService struct {
	VerifyFn func(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error)
}

func (m *mockAssertionService) Verify(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error) {
	if m.VerifyFn != nil {
		return m.VerifyFn(assertObject, challenge, clientData)
	}
	return 0, nil
}

func TestAssertionAdapter_Verify(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		setupPlugin  func(t *testing.T) AdapterPlugin
		setupService func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService
		wantErr      error
	}{
		"successful verification": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return &privkey.PublicKey, 1, nil
					},
					AssignedChallengeFn: func(ctx context.Context, r *Request) (string, error) {
						return "assigned", nil
					},
					UpdateCounterFn: func(ctx context.Context, r *Request, cnt uint32) error {
						if cnt != 42 {
							t.Errorf("unexpected counter value: got %d, want %d", cnt, 42)
						}
						return nil
					},
				}
			},
			setupService: func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService {
				if challenge != "assigned" {
					t.Errorf("unexpected challenge value: got %s, want \"assigned\"", challenge)
				}
				if !privkey.PublicKey.Equal(pubkey) {

					t.Errorf("invalid public key")
				}
				if counter != 1 {
					t.Errorf("unexpected counter value: got %d, want %d", counter, 1)
				}
				return &mockAssertionService{
					VerifyFn: func(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error) {
						if challenge != "challenge" {
							t.Errorf("unexpected challenge value: got %s, want \"challenge\"", challenge)
						}
						return 42, nil
					},
				}
			},
			wantErr: nil,
		},

		"ParseRequest fails": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return nil, "", errors.New("parse error")
					},
				}
			},
			setupService: nil,
			wantErr:      ErrBadRequest,
		},

		"PublicKeyAndCounter fails": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return nil, 0, errors.New("db error")
					},
				}
			},
			setupService: nil,
			wantErr:      ErrInternal,
		},

		"missing attestation": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return nil, 0, nil
					},
				}
			},
			setupService: nil,
			wantErr:      ErrAttestationRequired,
		},

		"AssignedChallenge fails": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return &ecdsa.PublicKey{}, 1, nil
					},
					AssignedChallengeFn: func(ctx context.Context, r *Request) (string, error) {
						return "", errors.New("db error")
					},
				}
			},
			setupService: nil,
			wantErr:      ErrInternal,
		},

		"no assigned challenge": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return &ecdsa.PublicKey{}, 1, nil
					},
					AssignedChallengeFn: func(ctx context.Context, r *Request) (string, error) {
						return "", nil
					},
				}
			},
			setupService: nil,
			wantErr:      ErrNewChallenge,
		},

		"Verify fails": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return &ecdsa.PublicKey{}, 1, nil
					},
					AssignedChallengeFn: func(ctx context.Context, r *Request) (string, error) {
						return "challenge", nil
					},
				}
			},
			setupService: func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService {
				return &mockAssertionService{
					VerifyFn: func(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error) {
						return 0, errors.New("verify failed")
					},
				}
			},
			wantErr: ErrBadRequest,
		},

		"UpdateCounter fails": {
			setupPlugin: func(t *testing.T) AdapterPlugin {
				return &mockPlugin{
					ParseRequestFn: func(ctx context.Context, r *Request) (*attest.AssertionObject, string, error) {
						return &attest.AssertionObject{}, "challenge", nil
					},
					PublicKeyAndCounterFn: func(ctx context.Context, r *Request) (*ecdsa.PublicKey, uint32, error) {
						return &ecdsa.PublicKey{}, 1, nil
					},
					AssignedChallengeFn: func(ctx context.Context, r *Request) (string, error) {
						return "challenge", nil
					},
					UpdateCounterFn: func(ctx context.Context, r *Request, cnt uint32) error {
						return errors.New("db error")
					},
				}
			},
			setupService: func(challenge string, pubkey *ecdsa.PublicKey, counter uint32) AssertionService {
				return &mockAssertionService{
					VerifyFn: func(assertObject *attest.AssertionObject, challenge string, clientData []byte) (uint32, error) {
						return 42, nil
					},
				}
			},
			wantErr: ErrInternal,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			plugin := tc.setupPlugin(t)
			adapter := NewAssertionAdapter(logger, "appID", plugin).(*AssertionAdapter)

			// Override NewService to inject mock AssertionService
			adapter.NewService = tc.setupService

			err := adapter.Verify(context.Background(), &Request{})
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("[%s] got err %v, want %v", name, err, tc.wantErr)
			}
		})
	}
}

func TestAssertionAdapter_NewServiceCreation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	plugin := &mockPlugin{}

	a := NewAssertionAdapter(logger, "testAppID", plugin).(*AssertionAdapter)

	service := a.NewService("challenge", &ecdsa.PublicKey{}, 10)
	if _, ok := service.(*attest.AssertionService); !ok {
		t.Fatalf("NewService did not return *attest.AssertionService")
	}
}
