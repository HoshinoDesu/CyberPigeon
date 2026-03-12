package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	settingPasswordHash = "auth.password_hash"
	sessionTTL          = 7 * 24 * time.Hour
)

func validateManagementPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 6 {
		return "", fmt.Errorf("密码长度不能少于 6 位")
	}
	for _, ch := range password {
		if (ch < '0' || ch > '9') && (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
			return "", fmt.Errorf("密码仅允许字母和数字")
		}
	}
	return password, nil
}

func (s *Storage) IsPasswordConfigured() (bool, error) {
	value, ok, err := s.GetSetting(settingPasswordHash)
	if err != nil {
		return false, err
	}
	return ok && strings.TrimSpace(value) != "", nil
}

func (s *Storage) SetPassword(password string) error {
	password, err := validateManagementPassword(password)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return s.SetSetting(settingPasswordHash, string(hash))
}

func (s *Storage) VerifyPassword(password string) (bool, error) {
	password = strings.TrimSpace(password)
	hash, ok, err := s.GetSetting(settingPasswordHash)
	if err != nil {
		return false, err
	}
	if !ok || strings.TrimSpace(hash) == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	return false, fmt.Errorf("校验密码失败: %w", err)
}

func (s *Storage) CreateSession() (string, error) {
	if err := s.DeleteExpiredSessions(); err != nil {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("生成会话令牌失败: %w", err)
	}
	token := hex.EncodeToString(raw)
	hash := sessionTokenHash(token)
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		"INSERT INTO sessions (token_hash, expires_at, created_at) VALUES (?, ?, ?)",
		hash, now.Add(sessionTTL), now,
	); err != nil {
		return "", fmt.Errorf("保存会话失败: %w", err)
	}
	return token, nil
}

func (s *Storage) ValidateSession(token string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	if err := s.DeleteExpiredSessions(); err != nil {
		return false, err
	}
	var expiresAt time.Time
	err := s.db.QueryRow(
		"SELECT expires_at FROM sessions WHERE token_hash = ?",
		sessionTokenHash(token),
	).Scan(&expiresAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询会话失败: %w", err)
	}
	now := time.Now().UTC()
	if !expiresAt.After(now) {
		_, _ = s.db.Exec("DELETE FROM sessions WHERE token_hash = ?", sessionTokenHash(token))
		return false, nil
	}
	if _, err := s.db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE token_hash = ?",
		now.Add(sessionTTL), sessionTokenHash(token),
	); err != nil {
		return false, fmt.Errorf("续期会话失败: %w", err)
	}
	return true, nil
}

func (s *Storage) DeleteSession(token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM sessions WHERE token_hash = ?", sessionTokenHash(token))
	if err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}
	return nil
}

func (s *Storage) DeleteAllSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions")
	if err != nil {
		return fmt.Errorf("删除全部会话失败: %w", err)
	}
	return nil
}

func (s *Storage) DeleteExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().UTC())
	if err != nil {
		return fmt.Errorf("清理过期会话失败: %w", err)
	}
	return nil
}

func (s *Storage) GetSetting(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("读取设置失败: %w", err)
	}
	return value, true, nil
}

func (s *Storage) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	return nil
}

func sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
