package health

import (
	"net/http"
	"stream/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{svc: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	health := api.Group("/health")
	{
		health.GET("", h.HealthCheck)
		health.GET("/stream", h.HealthCheckStream)
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	send := c.MustGet("send").(func(middleware.Response))

	response, err := h.svc.CheckHealth()
	if err != nil {
		send(middleware.Response{
			Code:    http.StatusServiceUnavailable,
			Message: "Health check failed",
			Error:   err,
		})
		return
	}

	send(middleware.Response{
		Code:    http.StatusOK,
		Message: "Health check completed",
		Data:    response,
	})
}

func (h *Handler) HealthCheckStream(c *gin.Context) {
	sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))

	response := h.svc.CheckHealthStream()
	sendStream(middleware.StreamResponse{
		TotalCount: 0,
		ChunkChan:  response,
	})
}
