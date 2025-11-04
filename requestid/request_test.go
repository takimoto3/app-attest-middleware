package requestid

import (
	"context"
	"errors"
	"net/http"
	"net/textproto"
	"sync/atomic"
	"testing"
)

type mockGenerator struct {
	ID  string
	Err error
}

func (m *mockGenerator) NextID() (string, error) {
	return m.ID, m.Err
}

func TestEnsureRequest(t *testing.T) {
	tests := map[string]struct {
		request        *http.Request
		mockID         string
		mockErr        error
		wantID         string
		genInitialized bool
		wantErr        bool
		errMsg         string
	}{
		"Generate new ID": {
			request:        &http.Request{},
			mockID:         "mock-id-001",
			wantID:         "mock-id-001",
			genInitialized: true,
			wantErr:        false,
		},
		"Use existing header": {
			request: &http.Request{
				Header: http.Header{
					textproto.CanonicalMIMEHeaderKey("X-Request-ID"): {"external-id-999"},
				},
			},
			mockID:         "should-not-be-used",
			wantID:         "external-id-999",
			genInitialized: true,
			wantErr:        false,
		},
		"Generator not initialized": {
			request:        &http.Request{},
			genInitialized: false,
			wantErr:        true,
			errMsg:         "generator not initialized",
		},
		"Generator error": {
			request:        &http.Request{},
			mockErr:        errors.New("generate error"),
			wantID:         "",
			genInitialized: true,
			wantErr:        true,
			errMsg:         "generate error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() {
				generator = atomic.Value{}
			})
			if tt.genInitialized {
				UseGenerator(&mockGenerator{ID: tt.mockID, Err: tt.mockErr})
			}

			newReq, id, err := EnsureRequest(tt.request)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error mismatch. Got: %s, Want: %s", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("ID mismatch. Got: %s, Want: %s", id, tt.wantID)
			}
			if ctxID := FromContext(newReq.Context()); ctxID != tt.wantID {
				t.Errorf("Context ID mismatch. Got: %s, Want: %s", ctxID, tt.wantID)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	const testID = "test-id-123"

	tests := map[string]struct {
		ctx  context.Context
		want string
	}{
		"ID exists in context": {
			ctx:  context.WithValue(context.Background(), requestIDKey, testID),
			want: testID,
		},
		"ID not exists in context": {
			ctx:  context.Background(),
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := FromContext(tt.ctx)
			if got != tt.want {
				t.Errorf("ID mismatch. Got: %s, Want: %s", got, tt.want)
			}
		})
	}
}

func TestUseGenerator(t *testing.T) {
	t.Cleanup(func() {
		generator = atomic.Value{}
	})

	mock := &mockGenerator{ID: "123"}
	UseGenerator(mock)

	got := currentGenerator()
	if got != mock {
		t.Errorf("UseGenerator failed. Got: %v, Want: %v", got, mock)
	}
}
