package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const authCookieName = "cyberpigeon_session"

type authRateLimiter struct {
	mu       sync.Mutex
	maxFails int
	window   time.Duration
	entries  map[string][]time.Time
}

func newAuthRateLimiter(maxFails int, window time.Duration) *authRateLimiter {
	return &authRateLimiter{
		maxFails: maxFails,
		window:   window,
		entries:  make(map[string][]time.Time),
	}
}

func (l *authRateLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entries := l.pruneLocked(key, now)
	return len(entries) < l.maxFails
}

func (l *authRateLimiter) RecordFailure(key string) {
	if l == nil {
		return
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entries := l.pruneLocked(key, now)
	entries = append(entries, now)
	l.entries[key] = entries
}

func (l *authRateLimiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

func (l *authRateLimiter) pruneLocked(key string, now time.Time) []time.Time {
	entries := l.entries[key]
	cutoff := now.Add(-l.window)
	kept := entries[:0]
	for _, ts := range entries {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) == 0 {
		delete(l.entries, key)
		return nil
	}
	l.entries[key] = kept
	return kept
}

type authStatus struct {
	AuthEnabled   bool   `json:"auth_enabled"`
	RequiresSetup bool   `json:"requires_setup"`
	Authenticated bool   `json:"authenticated"`
	Message       string `json:"message,omitempty"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state, err := s.getAuthStatus(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, state)
}

func (s *Server) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.storage == nil {
		http.Error(w, "Storage not enabled", http.StatusServiceUnavailable)
		return
	}
	requiresSetup, err := s.requiresPasswordSetup()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !requiresSetup {
		http.Error(w, "Password already configured", http.StatusConflict)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := s.storage.SetPassword(req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.issueSession(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"success": true})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.storage == nil {
		http.Error(w, "Storage not enabled", http.StatusServiceUnavailable)
		return
	}
	requiresSetup, err := s.requiresPasswordSetup()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requiresSetup {
		http.Error(w, "Password setup required", http.StatusPreconditionRequired)
		return
	}
	clientKey := authRateLimitKey(r)
	if s.authLimiter != nil && !s.authLimiter.Allow(clientKey) {
		writeAuthState(w, http.StatusTooManyRequests, authStatus{
			AuthEnabled:   true,
			Authenticated: false,
			Message:       "登录失败次数过多，请 15 分钟后再试",
		})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	ok, err := s.storage.VerifyPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		if s.authLimiter != nil {
			s.authLimiter.RecordFailure(clientKey)
		}
		writeAuthState(w, http.StatusUnauthorized, authStatus{
			AuthEnabled:   true,
			Authenticated: false,
			Message:       "密码错误",
		})
		return
	}
	if s.authLimiter != nil {
		s.authLimiter.Reset(clientKey)
	}
	if err := s.issueSession(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"success": true})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.storage != nil {
		if cookie, err := r.Cookie(authCookieName); err == nil {
			_ = s.storage.DeleteSession(cookie.Value)
		}
	}
	clearSessionCookie(w)
	writeJSON(w, map[string]bool{"success": true})
}

func (s *Server) handleAuthChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.storage == nil {
		http.Error(w, "Storage not enabled", http.StatusServiceUnavailable)
		return
	}
	requiresSetup, err := s.requiresPasswordSetup()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requiresSetup {
		writeAuthState(w, http.StatusPreconditionRequired, authStatus{
			AuthEnabled:   true,
			RequiresSetup: true,
			Authenticated: false,
			Message:       "当前实例尚未设置管理密码，请先完成初始化",
		})
		return
	}
	authenticated, err := s.isAuthenticated(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !authenticated {
		writeAuthState(w, http.StatusUnauthorized, authStatus{
			AuthEnabled:   true,
			Authenticated: false,
			Message:       "登录已失效，请重新登录后再修改密码",
		})
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	ok, err := s.storage.VerifyPassword(req.CurrentPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		writeAuthState(w, http.StatusUnauthorized, authStatus{
			AuthEnabled:   true,
			Authenticated: true,
			Message:       "当前密码错误",
		})
		return
	}
	if err := s.storage.SetPassword(req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.storage.DeleteAllSessions(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.issueSession(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"success": true})
}

func (s *Server) withAPIAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := s.getAuthStatus(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if state.AuthEnabled && !state.Authenticated {
			code := http.StatusUnauthorized
			if state.RequiresSetup {
				code = http.StatusPreconditionRequired
				state.Message = "当前实例尚未设置管理密码，请先完成初始化"
			} else if state.Message == "" {
				state.Message = "登录已失效，请重新登录"
			}
			writeAuthState(w, code, state)
			return
		}
		next(w, r)
	}
}

func (s *Server) getAuthStatus(r *http.Request) (authStatus, error) {
	if s.storage == nil {
		return authStatus{AuthEnabled: false, Authenticated: true}, nil
	}
	requiresSetup, err := s.requiresPasswordSetup()
	if err != nil {
		return authStatus{}, err
	}
	state := authStatus{AuthEnabled: true, RequiresSetup: requiresSetup}
	if requiresSetup {
		state.Message = "当前实例尚未设置管理密码，请先完成初始化"
		return state, nil
	}
	authenticated, err := s.isAuthenticated(r)
	if err != nil {
		return authStatus{}, err
	}
	state.Authenticated = authenticated
	if !authenticated {
		state.Message = "登录已失效，请重新登录"
	}
	return state, nil
}

func (s *Server) requiresPasswordSetup() (bool, error) {
	if s.storage == nil {
		return false, nil
	}
	configured, err := s.storage.IsPasswordConfigured()
	if err != nil {
		return false, err
	}
	return !configured, nil
}

func (s *Server) isAuthenticated(r *http.Request) (bool, error) {
	if s.storage == nil {
		return true, nil
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return false, nil
		}
		return false, err
	}
	return s.storage.ValidateSession(cookie.Value)
}

func (s *Server) issueSession(w http.ResponseWriter) error {
	token, err := s.storage.CreateSession()
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		MaxAge:   7 * 24 * 3600,
	})
	return nil
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeAuthState(w http.ResponseWriter, statusCode int, state authStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(state)
}

func authRateLimitKey(r *http.Request) string {
	forwardedFor := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if forwardedFor != "" {
		return forwardedFor
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
