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

// BulkCheckAndEnqueue Bulk version of CheckAndEnqueue.
// All SetNX + LPush commands in a single pipeline (1 round-trip for N events).
// Returns accepted, duplicated and rejected counts.
func (q *EventQueue) BulkCheckAndEnqueue(ctx context.Context, events []model.Event) (accepted, duplicated, rejected int) {
	type eventData struct {
		hash string
		data []byte
	}

	// Pre-marshal all events
	items := make([]eventData, 0, len(events))
	for i := range events {
		data, err := json.Marshal(&events[i])
		if err != nil {
			rejected++
			continue
		}
		items = append(items, eventData{hash: events[i].EventHash, data: data})
	}

	// Pipeline 1: SetNX for all hashes
	pipe := q.client.Pipeline()
	setCmds := make([]*redis.StatusCmd, len(items))
	for i, item := range items {
		setCmds[i] = pipe.SetArgs(ctx, dedupPrefix+item.hash, "1", redis.SetArgs{
			Mode: "NX",
			TTL:  dedupTTL,
		})
	}
	_, execErr := pipe.Exec(ctx)
	if execErr != nil && !errors.Is(execErr, redis.Nil) {
		rejected += len(items)
		return accepted, duplicated, rejected
	}

	// Filter: only new events (SetNX returned "OK")
	newItems := make([]eventData, 0, len(items))
	for i, cmd := range setCmds {
		if cmd.Val() == "OK" {
			newItems = append(newItems, items[i])
		} else {
			duplicated++
		}
	}

	if len(newItems) == 0 {
		return accepted, duplicated, rejected
	}

	// Pipeline 2: LPush all new events
	pipe2 := q.client.Pipeline()
	for _, item := range newItems {
		pipe2.LPush(ctx, queueKey, item.data)
	}
	_, err := pipe2.Exec(ctx)
	if err != nil {
		// Rollback hashes
		pipe3 := q.client.Pipeline()
		for _, item := range newItems {
			pipe3.Del(ctx, dedupPrefix+item.hash)
		}
		_, _ = pipe3.Exec(ctx)
		rejected += len(newItems)
		return accepted, duplicated, rejected
	}

	accepted = len(newItems)
	return accepted, duplicated, rejected
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
