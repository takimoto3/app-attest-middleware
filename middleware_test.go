package middleware_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	attest "github.com/takimoto3/app-attest"
	middleware "github.com/takimoto3/app-attest-middleware"
)

type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func (e *errReader) Close() error {
	return nil
}

type mockLogger struct {
	errors    []string
	infos     []string
	debugs    []string
	criticals []string
	warnings  []string
	context   interface{}
}

func (m *mockLogger) SetContext(ctx context.Context) {
	m.context = ctx
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	m.infos = append(m.infos, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.debugs = append(m.debugs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Criticalf(format string, args ...interface{}) {
	m.criticals = append(m.criticals, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Warningf(format string, args ...interface{}) {
	m.warnings = append(m.warnings, fmt.Sprintf(format, args...))
}

type MockPlugin struct {
	PubKey    *ecdsa.PublicKey
	Counter   uint32
	Challenge string

	ErrParse        error
	ErrGetKey       error
	ErrGetChallenge error
	ErrStore        error

	Redirected            bool
	NewChallengeResponded bool
}

func (m *MockPlugin) ParseRequest(r *http.Request, b []byte) (*http.Request, *attest.AssertionObject, string, error) {
	return r, &attest.AssertionObject{}, "clientChallenge", m.ErrParse
}
func (m *MockPlugin) GetPublicKeyAndCounter(r *http.Request) (*ecdsa.PublicKey, uint32, error) {
	return m.PubKey, m.Counter, m.ErrGetKey
}
func (m *MockPlugin) GetAssignedChallenge(r *http.Request) (string, error) {
	return m.Challenge, m.ErrGetChallenge
}
func (m *MockPlugin) RedirectToAttestation(w http.ResponseWriter, r *http.Request) {
	m.Redirected = true
	w.WriteHeader(http.StatusFound)
}
func (m *MockPlugin) ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error {
	m.NewChallengeResponded = true
	w.WriteHeader(http.StatusCreated)
	return nil
}
func (m *MockPlugin) StoreNewCounter(r *http.Request, c uint32) error {
	return m.ErrStore
}

type MockAssertionService struct {
	Called      bool
	ReturnCount uint32
	ReturnErr   error
}

func (m *MockAssertionService) Verify(_ *attest.AssertionObject, _ string, _ []byte) (uint32, error) {
	m.Called = true
	return m.ReturnCount, m.ReturnErr
}

func TestAssertionMiddleware_FullCoverage(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	tests := map[string]struct {
		plugin         *MockPlugin
		service        *MockAssertionService
		wantCode       int
		expectNext     bool
		expectErrorLog bool
		reqBody        io.Reader
	}{
		"success": {
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				Counter:   1,
				Challenge: "serverChallenge",
			},
			service:        &MockAssertionService{ReturnCount: 2},
			wantCode:       http.StatusOK,
			expectNext:     true,
			expectErrorLog: false,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"redirect if no pubkey": {
			plugin:         &MockPlugin{PubKey: nil},
			service:        &MockAssertionService{},
			wantCode:       http.StatusFound,
			expectNext:     false,
			expectErrorLog: false,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"new challenge if empty": {
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				Counter:   1,
				Challenge: "",
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusCreated,
			expectNext:     false,
			expectErrorLog: false,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"parse request error": {
			plugin: &MockPlugin{
				PubKey:   &priv.PublicKey,
				ErrParse: errors.New("parse failed"),
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusBadRequest,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"get pubkey error": {
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				ErrGetKey: errors.New("get pubkey failed"),
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusInternalServerError,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"get challenge error": {
			plugin: &MockPlugin{
				PubKey:          &priv.PublicKey,
				Counter:         1,
				ErrGetChallenge: errors.New("get challenge failed"),
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusInternalServerError,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"verify error": {
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				Counter:   1,
				Challenge: "serverChallenge",
			},
			service:        &MockAssertionService{ReturnErr: errors.New("verify failed")},
			wantCode:       http.StatusBadRequest,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"store error": {
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				Counter:   1,
				Challenge: "serverChallenge",
				ErrStore:  errors.New("store failed"),
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusInternalServerError,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        bytes.NewBufferString(`{}`),
		},
		"body nil": { // r.Body == nil
			plugin: &MockPlugin{
				PubKey:    &priv.PublicKey,
				Counter:   1,
				Challenge: "serverChallenge",
			},
			service:        &MockAssertionService{},
			wantCode:       http.StatusOK,
			expectNext:     true,
			expectErrorLog: false,
			reqBody:        nil,
		},
		"body read error": {
			plugin:         &MockPlugin{},
			service:        &MockAssertionService{},
			wantCode:       http.StatusBadRequest,
			expectNext:     false,
			expectErrorLog: true,
			reqBody:        &errReader{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			logger := &mockLogger{}
			mw := middleware.NewMiddleware(logger, "com.example.app", tt.plugin)
			mw.NewService = func(ch string, pk *ecdsa.PublicKey, c uint32) middleware.AssertionService {
				return tt.service
			}

			req := httptest.NewRequest("POST", "/assert", tt.reqBody)
			w := httptest.NewRecorder()

			calledNext := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledNext = true
				w.WriteHeader(http.StatusOK)
			})

			mw.Handler(next).ServeHTTP(w, req)
			res := w.Result()

			if res.StatusCode != tt.wantCode {
				t.Fatalf("[%s] unexpected status: got %d, want %d", name, res.StatusCode, tt.wantCode)
			}

			if calledNext != tt.expectNext {
				t.Errorf("[%s] next handler called: got %v, want %v", name, calledNext, tt.expectNext)
			}

			if tt.expectErrorLog && len(logger.errors) == 0 {
				t.Errorf("[%s] expected error log but got none", name)
			}
		})
	}
}
