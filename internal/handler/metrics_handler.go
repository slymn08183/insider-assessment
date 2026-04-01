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

// Get godoc
// @Summary Get aggregated metrics
// @Description Returns event counts, unique users, and optional daily/hourly breakdown
// @Tags metrics
// @Produce json
// @Param event_name query string true "Event name to filter"
// @Param from query string false "Start time (RFC3339)"
// @Param to query string false "End time (RFC3339)"
// @Param channel query string false "Filter by channel"
// @Param group_by query string false "Breakdown period: daily or hourly"
// @Success 200 {object} model.MetricsResult
// @Failure 400 {object} model.ErrorResponse
// @Router /metrics [get]
func (h *MetricsHandler) Get(c *gin.Context) {
	var query model.MetricsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	if err := query.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.repo.GetMetrics(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get metrics"})
		return
	}

	c.JSON(http.StatusOK, result)
}
