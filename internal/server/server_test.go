package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/CyberPigeon/internal/config"
	"github.com/CyberPigeon/internal/forwarder"
)

func TestHandleSaveChannelsPersistsNewChannels(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	initial := &config.Config{
		Server:  config.ServerConfig{Enabled: true, Listen: ":8080"},
		Storage: config.StorageConfig{Enabled: false},
		Channels: []config.ChannelConfig{
			{Type: "telegram", Enabled: true, BotToken: "old-token", ChatID: "old-chat"},
		},
	}
	if err := initial.Save(configPath); err != nil {
		t.Fatalf("initial.Save failed: %v", err)
	}

	fwd, err := forwarder.New(initial, nil, nil)
	if err != nil {
		t.Fatalf("forwarder.New failed: %v", err)
	}

	srv := New(initial, fwd, nil, configPath)
	body := []config.ChannelConfig{{Type: "telegram", Enabled: true, BotToken: "new-token", ChatID: "new-chat"}}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/channels/save", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	srv.handleSaveChannels(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile failed: %v", err)
	}
	if !bytes.Contains(data, []byte("new-token")) || !bytes.Contains(data, []byte("new-chat")) {
		t.Fatalf("saved config does not contain updated channels:\n%s", data)
	}
	if bytes.Contains(data, []byte("old-token")) || bytes.Contains(data, []byte("old-chat")) {
		t.Fatalf("saved config still contains stale channels:\n%s", data)
	}
}
