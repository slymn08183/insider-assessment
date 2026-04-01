package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slymn08183/insider-assessment/internal/cache"
	"github.com/slymn08183/insider-assessment/internal/model"
)

type EventHandler struct {
	queue *cache.EventQueue
}

func NewEventHandler(queue *cache.EventQueue) *EventHandler {
	return &EventHandler{queue: queue}
}

func (h *EventHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/events", h.Create)
	r.POST("/events/bulk", h.BulkCreate)
}

// processEvent validate and enqueue for single event
// returns true/nil if duplicate, returns false/err on error and returns false/nil on success
func (h *EventHandler) processEvent(ctx context.Context, event *model.Event) (duplicate bool, err error) {
	if err := event.ValidateTimestamp(); err != nil {
		return false, err
	}

	event.EventHash = event.GenerateHash()

	return h.queue.CheckAndEnqueue(ctx, event.EventHash, event)
}

// Create — POST /events
func (h *EventHandler) Create(c *gin.Context) {
	var event model.Event

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isDup, err := h.processEvent(c.Request.Context(), &event)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if isDup {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate event"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":     "accepted",
		"event_hash": event.EventHash,
	})
}

// BulkCreate — POST /events/bulk
func (h *EventHandler) BulkCreate(c *gin.Context) {
	var events []model.Event

	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accepted := 0
	rejected := 0
	duplicates := 0

	for i := range events {
		isDup, err := h.processEvent(c.Request.Context(), &events[i])
		if err != nil {
			rejected++
			continue
		}
		if isDup {
			duplicates++
			continue
		}
		accepted++
	}

	c.JSON(http.StatusAccepted, gin.H{
		"accepted":   accepted,
		"rejected":   rejected,
		"duplicates": duplicates,
	})
}
