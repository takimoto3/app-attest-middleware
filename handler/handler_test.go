package handler_test

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/handler"
	"github.com/takimoto3/app-attest-middleware/logger"
)

var _ handler.AttestationService = (*MockAttestationService)(nil)

type MockAttestationService struct {
	Func_Verify func(*attest.AttestationObject, []byte, []byte) (*attest.Result, error)
}

func (s *MockAttestationService) Verify(attestObj *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
	return s.Func_Verify(attestObj, clientDataHash, keyID)
}

var _ handler.AttestationPlugin = (*MockAttestationPlugin)(nil)

type MockAttestationPlugin struct {
	Func_GetAssignedChallenge func(*http.Request) (string, error)
	Func_ResponseNewChallenge func(http.ResponseWriter, *http.Request) error
	Func_ParseRequest         func(*http.Request) (*http.Request, *attest.AttestationObject, []byte, []byte, error)
	Func_StoreResult          func(*http.Request, *attest.Result) error
}

func (plugin *MockAttestationPlugin) GetAssignedChallenge(r *http.Request) (string, error) {
	return plugin.Func_GetAssignedChallenge(r)
}

func (plugin *MockAttestationPlugin) ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error {
	return plugin.Func_ResponseNewChallenge(w, r)
}
func (plugin *MockAttestationPlugin) ParseRequest(r *http.Request) (*http.Request, *attest.AttestationObject, []byte, []byte, error) {
	return plugin.Func_ParseRequest(r)
}

func (plugin *MockAttestationPlugin) StoreResult(r *http.Request, result *attest.Result) error {
	return plugin.Func_StoreResult(r, result)
}

func TestAttestationHandler_NewChallenge(t *testing.T) {
	tests := map[string]struct {
		ResponseNewChallenge func(w http.ResponseWriter, r *http.Request) error
		wantStatusCode       int
	}{
		"error case(func ResponseNewChallenge return error)": {
			func(w http.ResponseWriter, r *http.Request) error { return errors.New("response new challenge error") },
			http.StatusInternalServerError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			target := handler.AttestationHandler{
				Logger:            &logger.StdLogger{Logger: log.New(os.Stdout, "", log.LstdFlags)},
				AttestationPlugin: &MockAttestationPlugin{Func_ResponseNewChallenge: tt.ResponseNewChallenge},
			}

			r := httptest.NewRequest(http.MethodGet, "/attest/", nil)
			w := httptest.NewRecorder()

			target.NewChallenge(w, r)

			if w.Result().StatusCode != tt.wantStatusCode {
				t.Errorf("invalid status code return: %d, want: %d", w.Result().StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestAttestationHandler_VerifyAttestation(t *testing.T) {
	tests := map[string]struct {
		ParseRequest         func(*http.Request) (*attest.AttestationObject, []byte, []byte, error)
		GetAssignedChallenge func(*http.Request) (string, error)
		ResponseNewChallenge func(http.ResponseWriter, *http.Request) error
		Verify               func(*attest.AttestationObject, []byte, []byte) (*attest.Result, error)
		StoreResult          func(*http.Request, *attest.Result) error
		wantStatusCode       int
	}{
		"error case(func ParseRequest return error)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return nil, nil, nil, errors.New("parse request error")
			},
			nil,
			nil,
			nil,
			nil,
			http.StatusBadRequest,
		},
		"error case(func GetAssignedChallenge return error)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "", errors.New("get assigned challenge error") },
			nil,
			nil,
			nil,
			http.StatusInternalServerError,
		},
		"error case(func ResponseNewChallenge return error)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "", nil },
			func(rw http.ResponseWriter, r *http.Request) error { return errors.New("reponse new challenge error") },
			nil,
			nil,
			http.StatusInternalServerError,
		},
		"success case(response new challenge)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "", nil },
			func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusSeeOther)
				return nil
			},
			nil,
			nil,
			http.StatusSeeOther,
		},
		"error case(func Verify return error)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "assigned challenge", nil },
			nil,
			func(ao *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return nil, errors.New("attestation verify error")
			},
			nil,
			http.StatusBadRequest,
		},
		"error case(func StoreResult return error)": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "assigned challenge", nil },
			nil,
			func(ao *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			func(r *http.Request, result *attest.Result) error { return errors.New("store result error") },
			http.StatusInternalServerError,
		},
		"success case": {
			func(r *http.Request) (*attest.AttestationObject, []byte, []byte, error) {
				return &attest.AttestationObject{}, []byte("clientHash"), []byte("keyID"), nil
			},
			func(r *http.Request) (string, error) { return "assigned challenge", nil },
			nil,
			func(ao *attest.AttestationObject, clientDataHash, keyID []byte) (*attest.Result, error) {
				return &attest.Result{}, nil
			},
			func(r *http.Request, result *attest.Result) error { return nil },
			http.StatusOK,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			target := handler.AttestationHandler{
				Logger: &logger.StdLogger{Logger: log.New(os.Stdout, "", log.LstdFlags)},
				AttestationPlugin: &MockAttestationPlugin{
					Func_ParseRequest: func(r *http.Request) (*http.Request, *attest.AttestationObject, []byte, []byte, error) {
						attestObj, clientDataHash, keyID, err := tt.ParseRequest(r)
						return r, attestObj, clientDataHash, keyID, err
					},
					Func_GetAssignedChallenge: tt.GetAssignedChallenge,
					Func_ResponseNewChallenge: tt.ResponseNewChallenge,
					Func_StoreResult:          tt.StoreResult,
				},
				AttestationService: &MockAttestationService{Func_Verify: tt.Verify},
			}

			r := httptest.NewRequest(http.MethodGet, "/attest/", nil)
			w := httptest.NewRecorder()

			target.VerifyAttestation(w, r)

			if w.Result().StatusCode != tt.wantStatusCode {
				t.Errorf("invalid status code return: %d, want: %d", w.Result().StatusCode, tt.wantStatusCode)
			}
		})
	}
}
