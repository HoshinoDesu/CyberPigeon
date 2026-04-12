package config

import "testing"

func TestCloneCreatesDeepCopy(t *testing.T) {
	t.Parallel()

	original := &Config{
		Server: ServerConfig{AllowedOrigins: []string{"https://a.example.com"}},
		Channels: []ChannelConfig{
			{
				Type:    "webhook",
				Enabled: true,
				To:      []string{"a@example.com"},
				Headers: map[string]string{"X-Test": "old"},
			},
		},
	}

	clone := original.Clone()
	clone.Server.AllowedOrigins[0] = "https://b.example.com"
	clone.Channels[0].To[0] = "b@example.com"
	clone.Channels[0].Headers["X-Test"] = "new"

	if got := original.Server.AllowedOrigins[0]; got != "https://a.example.com" {
		t.Fatalf("AllowedOrigins was mutated: %q", got)
	}
	if got := original.Channels[0].To[0]; got != "a@example.com" {
		t.Fatalf("channel recipients were mutated: %q", got)
	}
	if got := original.Channels[0].Headers["X-Test"]; got != "old" {
		t.Fatalf("channel headers were mutated: %q", got)
	}
}
