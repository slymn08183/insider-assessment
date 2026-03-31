package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct{}

func (h *MetricsHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/metrics", h.Get)
}

// Get — GET /metrics
// Filters by event_name, returns aggregated results
// Query params: event_name (mandatory), from, to, channel, group_by
func (h *MetricsHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "metrics endpoint ready",
	})
}
