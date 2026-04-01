package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slymn08183/insider-assessment/internal/model"
)

const (
	queueKey    = "events_queue"
	dedupPrefix = "dedup:" // each hash gets its own key with TTL
	dedupTTL    = 24 * time.Hour
)

type EventQueue struct {
	client *redis.Client
}

func NewEventQueue(client *redis.Client) *EventQueue {
	return &EventQueue{client: client}
}

// CheckAndEnqueue — Duplicate check + mark + enqueue
// Set with NX: set if not exists, returns true if key was set (new event)
func (q *EventQueue) CheckAndEnqueue(ctx context.Context, hash string, event *model.Event) (duplicate bool, err error) {
	data, err := json.Marshal(event)
	if err != nil {
		return false, fmt.Errorf("failed to marshal event: %w", err)
	}

	// SET key value NX EX (set if not exists), with TTL
	wasSet, err := q.client.SetArgs(ctx, dedupPrefix+hash, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  dedupTTL,
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, fmt.Errorf("failed to check/set hash: %w", err)
	}
	if wasSet != "OK" {
		// duplicate
		return true, nil
	}

	// New event - enqueue
	if err := q.client.LPush(ctx, queueKey, data).Err(); err != nil {
		q.client.Del(ctx, dedupPrefix+hash)
		return false, fmt.Errorf("failed to enqueue: %w", err)
	}

	return false, nil
}

// Dequeue BRPOP
func (q *EventQueue) Dequeue(ctx context.Context, timeout time.Duration) (*model.Event, error) {
	result, err := q.client.BRPop(ctx, timeout, queueKey).Result()
	if err != nil {
		return nil, err
	}

	// BRPop [key, value]
	var event model.Event
	if err := json.Unmarshal([]byte(result[1]), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}
