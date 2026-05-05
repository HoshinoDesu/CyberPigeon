package storage

import (
	"testing"
	"time"

	"github.com/HoshinoDesu/CyberPigeon/internal/modem"
)

func TestGenerateIDUsesStableMessageFields(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 5, 5, 12, 0, 0, 123, time.UTC)
	sms := &modem.SMS{
		Number:    "+8613800000000",
		Text:      "hello",
		Timestamp: timestamp,
	}

	first := GenerateID("imei-1", sms)
	second := GenerateID("imei-1", sms)
	if first != second {
		t.Fatalf("GenerateID should be stable: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("GenerateID should return a sha256 hex digest, got length %d", len(first))
	}
}

func TestGenerateIDDifferentiatesMessageContent(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	base := &modem.SMS{Number: "+8613800000000", Text: "hello", Timestamp: timestamp}
	changed := &modem.SMS{Number: "+8613800000000", Text: "hello again", Timestamp: timestamp}

	if GenerateID("imei-1", base) == GenerateID("imei-1", changed) {
		t.Fatal("GenerateID should change when message text changes")
	}
}
