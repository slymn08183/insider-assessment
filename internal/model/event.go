package model

import (
	"crypto/sha256"
	"fmt"
	"time"

	"gorm.io/datatypes"
)

// Event payload
type Event struct {
	ID         uint                        `json:"-"              gorm:"primaryKey"`
	EventHash  string                      `json:"event_hash"     gorm:"uniqueIndex;size:64;not null" swaggerignore:"true"`
	EventName  string                      `json:"event_name"     gorm:"index;size:255;not null"  binding:"required"  example:"product_view"`
	Channel    string                      `json:"channel"        gorm:"index;size:100"           example:"web"`
	CampaignID string                      `json:"campaign_id"    gorm:"size:100"                 example:"cmp_987"`
	UserID     string                      `json:"user_id"        gorm:"index;size:255;not null"  binding:"required"  example:"user_123"`
	Timestamp  int64                       `json:"timestamp"      gorm:"index;not null"           binding:"required"  example:"1723475612"`
	Tags       datatypes.JSONSlice[string] `json:"tags"           gorm:"type:jsonb;default:'[]'"  swaggertype:"array,string"`
	Metadata   datatypes.JSONMap           `json:"metadata"       gorm:"type:jsonb;default:'{}'"  swaggertype:"object"`
	CreatedAt  time.Time                   `json:"-"              gorm:"autoCreateTime"`
}

// GenerateHash Generates fingerprint for idempotency
// event_name + user_id + timestamp = hash(SHA256)
func (e *Event) GenerateHash() string {
	raw := fmt.Sprintf("%s:%s:%d", e.EventName, e.UserID, e.Timestamp)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

// ValidateTimestamp checks if it is valid or not.
// Should not be older than 2020 and should not be in the future
// In production if needed, these numbers can be moved to config.
func (e *Event) ValidateTimestamp() error {
	t := time.Unix(e.Timestamp, 0)
	minTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	maxTime := time.Now().Add(1 * time.Hour)

	if t.Before(minTime) {
		return fmt.Errorf("timestamp is too old: %v", t)
	}
	if t.After(maxTime) {
		return fmt.Errorf("timestamp is in the future: %v", t)
	}
	return nil
}
