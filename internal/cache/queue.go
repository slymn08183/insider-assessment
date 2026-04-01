package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slymn08183/insider-assessment/internal/model"
)

const (
	queueKey = "events_queue"
	dedupKey = "event_hashes"
)

type EventQueue struct {
	client *redis.Client
}

func NewEventQueue(client *redis.Client) *EventQueue {
	return &EventQueue{client: client}
}

func (q *EventQueue) IsDuplicate(ctx context.Context, hash string) (bool, error) {
	exists, err := q.client.SIsMember(ctx, dedupKey, hash).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}
	return exists, nil
}

// MarkProcessed Set hash
func (q *EventQueue) MarkProcessed(ctx context.Context, hash string) error {
	return q.client.SAdd(ctx, dedupKey, hash).Err()
}

func (q *EventQueue) RemoveHash(ctx context.Context, hash string) {
	q.client.SRem(ctx, dedupKey, hash)
}

// Enqueue LPUSH
func (q *EventQueue) Enqueue(ctx context.Context, event *model.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return q.client.LPush(ctx, queueKey, data).Err()
}

// CheckAndEnqueue — Duplicate check + mark + enqueue
// Redis Pipeline SADD + LPUSH single round-trip
func (q *EventQueue) CheckAndEnqueue(ctx context.Context, hash string, event *model.Event) (duplicate bool, err error) {
	data, err := json.Marshal(event)
	if err != nil {
		return false, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Pipeline
	pipe := q.client.Pipeline()
	sAddCmd := pipe.SAdd(ctx, dedupKey, hash)
	lPushCmd := pipe.LPush(ctx, queueKey, data)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("pipeline failed: %w", err)
	}

	// If SADD is 0 means duplicate, remove from queue
	if sAddCmd.Val() == 0 {
		q.client.LRem(ctx, queueKey, 1, data)
		return true, nil
	}

	// On LPUSH error rollback
	if lPushCmd.Err() != nil {
		q.client.SRem(ctx, dedupKey, hash)
		return false, fmt.Errorf("failed to enqueue: %w", lPushCmd.Err())
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
