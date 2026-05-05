package notifier

import (
	"testing"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
)

func TestNewReturnsErrorForInvalidEnabledChannel(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Channels: []config.ChannelConfig{
		{Type: "telegram", Enabled: true, BotToken: "", ChatID: "chat"},
	}}

	if _, err := New(cfg); err == nil {
		t.Fatal("New should reject invalid enabled channels")
	}
}

func TestNewAllowsNoEnabledChannels(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Channels: []config.ChannelConfig{
		{Type: "telegram", Enabled: false},
	}}

	n, err := New(cfg)
	if err != nil {
		t.Fatalf("New should allow configs with no enabled channels: %v", err)
	}
	if n == nil {
		t.Fatal("New returned nil notifier")
	}
}
