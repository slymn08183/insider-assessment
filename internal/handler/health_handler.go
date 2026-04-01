package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slymn08183/insider-assessment/internal/model"
)

type HealthHandler struct{}

func (h *HealthHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
}

// Health godoc
// @Summary Health check
// @Description Returns OK if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} model.HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:  "ok",
		Message: "The API is up and running",
	})
}
