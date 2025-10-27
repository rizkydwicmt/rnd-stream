package handler

import (
	"net/http"
	"stream/application/ticketsV2/domain"
	"stream/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for ticketsV2
type Handler struct {
	svc domain.Service
}

// NewHandler creates a new Handler
func NewHandler(service domain.Service) *Handler {
	return &Handler{svc: service}
}

// RegisterRoutes registers the handler routes
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	tickets := api.Group("/v2/tickets")
	{
		tickets.POST("/stream", h.StreamTickets)
		tickets.POST("/stream/batch", h.StreamTicketsBatch)
	}
}

// RegisterRoutesWithPrefix registers the handler routes with a custom prefix
// This is used to create separate endpoints for different databases
func (h *Handler) RegisterRoutesWithPrefix(group *gin.RouterGroup) {
	group.POST("/stream", h.StreamTickets)
	group.POST("/stream/batch", h.StreamTicketsBatch)
}

// StreamTickets handles the POST /v2/tickets/stream endpoint
func (h *Handler) StreamTickets(c *gin.Context) {
	sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))
	requestID := c.GetString("requestId")
	startTime := time.Now()

	// Parse and bind payload
	var payload domain.QueryPayload
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

	// Stream processing using internal/stream package
	response := h.svc.StreamTickets(c.Request.Context(), &payload)

	// Log request completion
	duration := time.Since(startTime)
	h.svc.LogRequest(requestID, &payload, duration, response.Error)

	// Send streaming response
	sendStream(response)
}

// StreamTicketsBatch handles the POST /v2/tickets/stream/batch endpoint
func (h *Handler) StreamTicketsBatch(c *gin.Context) {
	sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))
	requestID := c.GetString("requestId")
	startTime := time.Now()

	// Parse and bind payload
	var payload domain.QueryPayload
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

	// Stream processing using batch mode
	response := h.svc.StreamTicketsBatch(c.Request.Context(), &payload)

	// Log request completion
	duration := time.Since(startTime)
	h.svc.LogRequest(requestID, &payload, duration, response.Error)

	// Send streaming response
	sendStream(response)
}
