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

// Create godoc
// @Summary Ingest a single event
// @Description Validates the event, checks for duplicates, and queues it for processing
// @Tags events
// @Accept json
// @Produce json
// @Param event body model.Event true "Event payload"
// @Success 202 {object} model.EventResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse
// @Router /events [post]
func (h *EventHandler) Create(c *gin.Context) {
	var event model.Event

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	isDup, err := h.processEvent(c.Request.Context(), &event)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if isDup {
		c.JSON(http.StatusConflict, model.ErrorResponse{Error: "duplicate event"})
		return
	}

	c.JSON(http.StatusAccepted, model.EventResponse{
		Status:    "accepted",
		EventHash: event.EventHash,
	})
}

// BulkCreate godoc
// @Summary Ingest multiple events
// @Description Validates each event, deduplicates, and queues via Redis pipeline
// @Tags events
// @Accept json
// @Produce json
// @Param events body []model.Event true "Array of events"
// @Success 202 {object} model.BulkResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /events/bulk [post]
func (h *EventHandler) BulkCreate(c *gin.Context) {
	var events []model.Event

	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	// Validate and generate hashes
	valid := make([]model.Event, 0, len(events))
	rejected := 0
	for i := range events {
		if err := events[i].ValidateTimestamp(); err != nil {
			rejected++
			continue
		}
		events[i].EventHash = events[i].GenerateHash()
		valid = append(valid, events[i])
	}

	// Bulk pipeline — all Redis ops in 2 round-trips
	accepted, duplicates, bulkRejected := h.queue.BulkCheckAndEnqueue(c.Request.Context(), valid)
	rejected += bulkRejected

	c.JSON(http.StatusAccepted, model.BulkResponse{
		Accepted:   accepted,
		Rejected:   rejected,
		Duplicates: duplicates,
	})
}
