package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/GOPAL-YADAV-D/Soter/internal/database"
	"github.com/sirupsen/logrus"
)

type HealthHandler struct {
	db *database.DB
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Database  string    `json:"database"`
	Storage   string    `json:"storage"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// HealthCheck handles the /healthz endpoint
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	}

	// Check database connectivity
	if err := h.db.HealthCheck(); err != nil {
		logrus.WithError(err).Error("Database health check failed")
		response.Status = "degraded"
		response.Database = "unhealthy"
	} else {
		response.Database = "healthy"
	}

	// Check storage connectivity (placeholder - to be implemented with Azure Blob)
	response.Storage = "healthy" // TODO: Implement actual storage health check

	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}