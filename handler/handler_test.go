package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sony/sonyflake/v2"
	"github.com/takimoto3/app-attest-middleware/adapter"
	"github.com/takimoto3/app-attest-middleware/handler"
	"github.com/takimoto3/app-attest-middleware/plugin"
	"github.com/takimoto3/app-attest-middleware/requestid"
)

var _ adapter.AttestationAdapter = &mockAdapter{}

// Mock adapter to simulate Verify / NewChallenge behavior
type mockAdapter struct {
	verifyFunc       func() error
	newChallengeFunc func() (string, error)
}

func (m *mockAdapter) Verify(ctx context.Context, _ *plugin.AttestationRequest) error {
	return m.verifyFunc()
}

func (m *mockAdapter) NewChallenge(ctx context.Context, _ *plugin.AttestationRequest) (string, error) {
	return m.newChallengeFunc()
}

func TestVerifyHandler(t *testing.T) {
	requestid.UseSnowFlake(sonyflake.Settings{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cases := map[string]struct {
		verifyErr       error
		newChallengeStr string
		newChallengeErr error
		wantStatus      int
		wantBody        string
	}{
		"verify_success": {
			verifyErr:  nil,
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		"verify_failure": {
			verifyErr:  adapter.ErrBadRequest,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Bad Request\n",
		},
		"triggers_new_challenge_success": {
			verifyErr:       adapter.ErrNewChallenge,
			newChallengeStr: "challenge123",
			newChallengeErr: nil,
			wantStatus:      http.StatusOK,
			wantBody:        "challenge123",
		},
		"triggers_new_challenge_failure": {
			verifyErr:       adapter.ErrNewChallenge,
			newChallengeStr: "",
			newChallengeErr: adapter.ErrBadRequest,
			wantStatus:      http.StatusBadRequest,
			wantBody:        "Bad Request\n",
		},
		"new_challenge_failed_internal": {
			verifyErr:       adapter.ErrNewChallenge,
			newChallengeStr: "",
			newChallengeErr: errors.New("some internal error"),
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "Internal Server Error\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			adapter := &mockAdapter{
				verifyFunc: func() error { return tc.verifyErr },
				newChallengeFunc: func() (string, error) {
					return tc.newChallengeStr, tc.newChallengeErr
				},
			}
			handler := handler.NewAppAttestHandler(logger, adapter)

			req := httptest.NewRequest(http.MethodPost, "/verify", nil)
			w := httptest.NewRecorder()

			handler.Verify(w, req)

			resp := w.Result()
			body := w.Body.String()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			if body != tc.wantBody {
				t.Errorf("expected body %q, got %q", tc.wantBody, body)
			}
		})
	}
}
