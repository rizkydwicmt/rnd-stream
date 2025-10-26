package tickets

import (
	"net/http"
	"stream/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for tickets
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler
func NewHandler(service *Service) *Handler {
	return &Handler{svc: service}
}

// RegisterRoutes registers the handler routes
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	tickets := api.Group("/v1/tickets")
	{
		tickets.POST("/stream", h.StreamTickets)
	}
}

// RegisterRoutesWithPrefix registers the handler routes with a custom prefix
// This is used to create separate endpoints for different databases
func (h *Handler) RegisterRoutesWithPrefix(group *gin.RouterGroup) {
	group.POST("/stream", h.StreamTickets)
}

// StreamTickets handles the POST /v1/tickets/stream endpoint
func (h *Handler) StreamTickets(c *gin.Context) {
	sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))
	requestID := c.GetString("requestId")
	startTime := time.Now()

	// Parse and bind payload
	var payload QueryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		send := c.MustGet("send").(func(middleware.Response))
		send(middleware.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid JSON payload",
			Error:   err,
		})
		return
	}

	// Log request start
	h.svc.LogRequest(requestID, &payload, 0, nil)

	// Stream processing
	response := h.svc.StreamTickets(c.Request.Context(), &payload)

	// Log request completion
	duration := time.Since(startTime)
	h.svc.LogRequest(requestID, &payload, duration, response.Error)

	// Send streaming response
	sendStream(response)
}
