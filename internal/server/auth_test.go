package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/HoshinoDesu/CyberPigeon/internal/storage"
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
			name: "query token on websocket path",
			setRequest: func(r *http.Request, token string) {
				r.URL.Path = "/ws"
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
			if err := store.SetPassword("abc1234567"); err != nil {
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

func TestSessionTokenFromRequestIgnoresQueryTokenForAPI(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/modems?token=secret", nil)
	if got := sessionTokenFromRequest(req); got != "" {
		t.Fatalf("API query token should be ignored, got %q", got)
	}
}

func TestAuthRateLimitKeyIgnoresSpoofableForwardedHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	req.Header.Set("X-Real-IP", "203.0.113.100")

	if got := authRateLimitKey(req); got != "192.0.2.10" {
		t.Fatalf("authRateLimitKey should use RemoteAddr host, got %q", got)
	}
}

func TestIssueSessionSetsSecureCookieForHTTPS(t *testing.T) {
	t.Parallel()

	store := newTestStorage(t)
	srv := &Server{storage: store}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	if err := srv.issueSession(rr, req); err != nil {
		t.Fatalf("issueSession failed: %v", err)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if !cookies[0].Secure {
		t.Fatal("session cookie should be Secure for HTTPS requests")
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
