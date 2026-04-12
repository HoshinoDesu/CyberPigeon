package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/CyberPigeon/internal/storage"
)

func TestHandleAuthLogoutDeletesSessionFromAllTokenSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setRequest func(*http.Request, string)
	}{
		{
			name: "bearer token",
			setRequest: func(r *http.Request, token string) {
				r.Header.Set("Authorization", "Bearer "+token)
			},
		},
		{
			name: "query token",
			setRequest: func(r *http.Request, token string) {
				q := r.URL.Query()
				q.Set("token", token)
				r.URL.RawQuery = q.Encode()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newTestStorage(t)
			if err := store.SetPassword("abc123"); err != nil {
				t.Fatalf("SetPassword failed: %v", err)
			}

			token, err := store.CreateSession()
			if err != nil {
				t.Fatalf("CreateSession failed: %v", err)
			}

			srv := &Server{storage: store}
			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			tt.setRequest(req, token)
			rr := httptest.NewRecorder()

			srv.handleAuthLogout(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
			}

			ok, err := store.ValidateSession(token)
			if err != nil {
				t.Fatalf("ValidateSession failed: %v", err)
			}
			if ok {
				t.Fatal("session should be deleted on logout")
			}
		})
	}
}

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()

	store, err := storage.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("storage.New failed: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}
