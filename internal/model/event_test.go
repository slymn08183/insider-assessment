package model

import (
	"testing"
	"time"
)

func TestGenerateHash(t *testing.T) {
	event := &Event{
		EventName: "product_view",
		UserID:    "user_123",
		Timestamp: 1723475612,
	}

	hash1 := event.GenerateHash()
	hash2 := event.GenerateHash()

	// Same event, same hash
	if hash1 != hash2 {
		t.Errorf("same event should produce same hash, got %s and %s", hash1, hash2)
	}

	// Hash must be 64 chars (SHA256 hex)
	if len(hash1) != 64 {
		t.Errorf("hash should be 64 chars, got %d", len(hash1))
	}

	// Different event, different hash
	event2 := &Event{
		EventName: "product_view",
		UserID:    "user_456",
		Timestamp: 1723475612,
	}
	hash3 := event2.GenerateHash()
	if hash1 == hash3 {
		t.Error("different events should produce different hashes")
	}
}

func TestValidateTimestamp(t *testing.T) {
	// Valid timestamp
	event := &Event{
		Timestamp: time.Now().Unix(),
	}
	if err := event.ValidateTimestamp(); err != nil {
		t.Errorf("valid timestamp should pass, got: %v", err)
	}

	// Too old timestamp (2019)
	event.Timestamp = time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	if err := event.ValidateTimestamp(); err == nil {
		t.Error("timestamp before 2020 should fail")
	}

	// Future timestamp (2 hours ahead)
	event.Timestamp = time.Now().Add(2 * time.Hour).Unix()
	if err := event.ValidateTimestamp(); err == nil {
		t.Error("timestamp 2 hours in the future should fail")
	}

	// Boundary: just after 2020-01-01 should pass
	event.Timestamp = time.Date(2020, 1, 1, 0, 0, 1, 0, time.UTC).Unix()
	if err := event.ValidateTimestamp(); err != nil {
		t.Errorf("timestamp just after 2020 should pass, got: %v", err)
	}
}
