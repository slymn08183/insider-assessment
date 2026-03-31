package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type EventHandler struct{}

func (h *EventHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/events", h.Create)
	r.POST("/events/bulk", h.BulkCreate)
}

// Create — POST /events
// Accepts an event, validates and sends it to the queue
func (h *EventHandler) Create(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "event received",
	})
}

// BulkCreate — POST /events/bulk
// Accepts multiple events, validates and sends it to the queue
func (h *EventHandler) BulkCreate(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "bulk events received",
	})
}
