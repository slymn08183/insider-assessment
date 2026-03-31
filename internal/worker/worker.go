package worker

import (
	"context"
	"log"
	"time"

	"github.com/slymn08183/insider-assessment/internal/cache"
	"github.com/slymn08183/insider-assessment/internal/model"
	"github.com/slymn08183/insider-assessment/internal/repository"
)

type Worker struct {
	queue         *cache.EventQueue
	repo          *repository.EventRepository
	batchSize     int
	flushInterval time.Duration
	stopCh        chan struct{} // channel for stop signal
}

func New(queue *cache.EventQueue, repo *repository.EventRepository, batchSize int, flushInterval time.Duration) *Worker {
	return &Worker{
		queue:         queue,
		repo:          repo,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}
}

// Start worker as goroutine
func (w *Worker) Start() {
	go w.run()
	log.Printf("Worker started (batchSize=%d, flushInterval=%s)", w.batchSize, w.flushInterval)
}

func (w *Worker) run() {
	batch := make([]*model.Event, 0, w.batchSize)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			// When receive the stop signal, flush and exit
			if len(batch) > 0 {
				w.flush(batch)
			}
			return

		case <-ticker.C:
			// When the time is up, flush
			if len(batch) > 0 {
				w.flush(batch)
				batch = make([]*model.Event, 0, w.batchSize)
			}

		default:
			// Get event from queue(1 sec timeout)
			event, err := w.queue.Dequeue(context.Background(), 1*time.Second)
			if err != nil {
				continue
			}

			batch = append(batch, event)

			// Batch is full. Flush
			if len(batch) >= w.batchSize {
				w.flush(batch)
				batch = make([]*model.Event, 0, w.batchSize)
			}
		}
	}
}

func (w *Worker) flush(batch []*model.Event) {
	if err := w.repo.BatchInsert(batch); err != nil {
		log.Printf("Failed to flush batch (%d events): %v", len(batch), err)
		return
	}
	log.Printf("Flushed %d events to database", len(batch))
}

func (w *Worker) Stop() {
	close(w.stopCh)
	log.Println("Worker stopped")
}
