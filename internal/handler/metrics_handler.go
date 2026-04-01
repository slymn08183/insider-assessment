package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slymn08183/insider-assessment/internal/model"
	"github.com/slymn08183/insider-assessment/internal/repository"
)

type MetricsHandler struct {
	repo *repository.EventRepository
}

func NewMetricsHandler(repo *repository.EventRepository) *MetricsHandler {
	return &MetricsHandler{repo: repo}
}

func (h *MetricsHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/metrics", h.Get)
}

// Get — GET /metrics
func (h *MetricsHandler) Get(c *gin.Context) {
	var query model.MetricsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := query.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.repo.GetMetrics(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics"})
		return
	}

	c.JSON(http.StatusOK, result)
}
