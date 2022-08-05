package middleware_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	attest "github.com/takimoto3/app-attest"
	middleware "github.com/takimoto3/app-attest-middleware"
	"github.com/takimoto3/app-attest-middleware/logger"
)

var _ middleware.AssertionPlugin = (*MockAssertionPlugin)(nil)

type MockAssertionPlugin struct {
	Func_GetAssignedChallenge   func(*http.Request) (string, error)
	Func_ResponseNewChallenge   func(http.ResponseWriter, *http.Request) error
	Func_ParseRequest           func(*http.Request, []byte) (*http.Request, *attest.AssertionObject, string, error)
	Func_RedirectToAttestation  func(http.ResponseWriter, *http.Request)
	Func_GetPublicKeyAndCounter func(*http.Request) (*ecdsa.PublicKey, uint32, error)
	Func_StoreNewCounter        func(*http.Request, uint32) error
}

func (plugin *MockAssertionPlugin) GetAssignedChallenge(r *http.Request) (string, error) {
	return plugin.Func_GetAssignedChallenge(r)
}
func (plugin *MockAssertionPlugin) ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error {
	return plugin.Func_ResponseNewChallenge(w, r)
}
func (plugin *MockAssertionPlugin) ParseRequest(r *http.Request, requestBody []byte) (*http.Request, *attest.AssertionObject, string, error) {
	return plugin.Func_ParseRequest(r, requestBody)
}
func (plugin *MockAssertionPlugin) RedirectToAttestation(w http.ResponseWriter, r *http.Request) {
	plugin.Func_RedirectToAttestation(w, r)
}
func (plugin *MockAssertionPlugin) GetPublicKeyAndCounter(r *http.Request) (*ecdsa.PublicKey, uint32, error) {
	return plugin.Func_GetPublicKeyAndCounter(r)
}
func (plugin *MockAssertionPlugin) StoreNewCounter(r *http.Request, counter uint32) error {
	return plugin.Func_StoreNewCounter(r, counter)
}

type TestData struct {
	AppID     string
	Publickey string
	Assertion string
}

var challenge = "bBjeLwdQD4KYRpzL"
var requestBody = "{\"levelId\":\"1234\",\"action\":\"getGameLevel\",\"challenge\":\"bBjeLwdQD4KYRpzL\"}"

func TestAppAttestAssert(t *testing.T) {
	buf, err := ioutil.ReadFile("testdata/attestdata.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData TestData
	if err := json.Unmarshal(buf, &testData); err != nil {
		t.Fatal(err)
	}

	buf, err = base64.StdEncoding.DecodeString(testData.Assertion)
	if err != nil {
		t.Fatal(err)
	}
	assertionObject := &attest.AssertionObject{}
	err = assertionObject.Unmarshal(buf)
	if err != nil {
		t.Fatal(err)
	}
	buf, err = hex.DecodeString("04" + testData.Publickey) // "04" uncompressed point
	if err != nil {
		t.Fatal(err)
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), buf)
	pubkey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	tests := map[string]struct {
		ParseRequest           func() (*attest.AssertionObject, string, error)
		GetPublicKeyAndCounter func(*http.Request) (*ecdsa.PublicKey, uint32, error)
		RedirectToAttestation  func(http.ResponseWriter, *http.Request)
		GetAssignedChallenge   func(*http.Request) (string, error)
		ResponseNewChallenge   func(http.ResponseWriter, *http.Request) error
		StoreNewCounter        func(*http.Request, uint32) error
		wantStatusCode         int
		wantCalledNestHandler  bool
	}{
		"error case(func ParseRequest return error)": {
			func() (*attest.AssertionObject, string, error) {
				return nil, "", errors.New("parse error")
			},
			nil,
			nil,
			nil,
			nil,
			nil,
			http.StatusBadRequest,
			false,
		},
		"error case(func GetPublicKeyAndCounter return error)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) {
				return nil, 0, errors.New("get public key and counter error")
			},
			nil,
			nil,
			nil,
			nil,
			http.StatusInternalServerError,
			false,
		},
		"success case(redirect to attestation site)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return nil, 0, nil },
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusSeeOther) },
			nil,
			nil,
			nil,
			http.StatusSeeOther,
			false,
		},
		"error case(func GetAssignedChallenge return error)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return "", errors.New("get assigned challenge error") },
			nil,
			nil,
			http.StatusInternalServerError,
			false,
		},
		"error case(func ResponseNewChallenge return error)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return "", nil },
			func(rw http.ResponseWriter, r *http.Request) error { return errors.New("response new challenge error") },
			nil,
			http.StatusInternalServerError,
			false,
		},
		"success case(response new challegne)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return "", nil },
			func(rw http.ResponseWriter, r *http.Request) error { return nil },
			nil,
			http.StatusOK,
			false,
		},
		"error case(assertion verify failed)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return "xxxxxxxxxxxxx", nil },
			nil,
			nil,
			http.StatusBadRequest,
			false,
		},
		"error case(func StoreNewCounter return error)": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return challenge, nil },
			nil,
			func(r *http.Request, u uint32) error { return errors.New("store new counter error") },
			http.StatusInternalServerError,
			false,
		},
		"success case": {
			func() (*attest.AssertionObject, string, error) {
				return assertionObject, challenge, nil
			},
			func(r *http.Request) (*ecdsa.PublicKey, uint32, error) { return pubkey, 0, nil },
			nil,
			func(r *http.Request) (string, error) { return challenge, nil },
			nil,
			func(r *http.Request, u uint32) error { return nil },
			http.StatusOK,
			true,
		},
	}

	for name, tt := range tests {
		plugin := MockAssertionPlugin{
			Func_ParseRequest: func(r *http.Request, b []byte) (*http.Request, *attest.AssertionObject, string, error) {
				assertionObj, challenge, err := tt.ParseRequest()
				return r, assertionObj, challenge, err
			},
			Func_GetPublicKeyAndCounter: func(r *http.Request) (*ecdsa.PublicKey, uint32, error) {
				return tt.GetPublicKeyAndCounter(r)
			},
			Func_RedirectToAttestation: func(w http.ResponseWriter, r *http.Request) {
				tt.RedirectToAttestation(w, r)
			},
			Func_GetAssignedChallenge: func(r *http.Request) (string, error) {
				return tt.GetAssignedChallenge(r)
			},
			Func_ResponseNewChallenge: func(w http.ResponseWriter, r *http.Request) error {
				return tt.ResponseNewChallenge(w, r)
			},
			Func_StoreNewCounter: func(r *http.Request, cnt uint32) error {
				return tt.StoreNewCounter(r, cnt)
			},
		}

		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			calledNestHandler := false
			mux.Handle("/assert", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				calledNestHandler = true

			}))

			midd := middleware.AppAttestAssert(&logger.StdLogger{Logger: log.New(os.Stdout, "", log.LstdFlags)}, testData.AppID, &plugin)(mux)
			r := httptest.NewRequest(http.MethodPost, "/assert", bytes.NewBufferString(requestBody))
			w := httptest.NewRecorder()
			midd.ServeHTTP(w, r)

			if w.Result().StatusCode != tt.wantStatusCode {
				t.Errorf("invalid status code return: %d, want: %d", w.Result().StatusCode, tt.wantStatusCode)
			}
			if tt.wantCalledNestHandler != calledNestHandler {
				t.Errorf("illegal call nested handler: %v", calledNestHandler)
			}
		})
	}

}
