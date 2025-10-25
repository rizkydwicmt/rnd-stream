package middleware

import (
	"sync"
	"time"
)

type Response struct {
	Data    any
	Message string
	Code    int
	Error   error
}

type ResponseAPIDebug struct {
	Version   string    `json:"version"`
	Error     *string   `json:"error"`
	StartTime time.Time `json:"startTime"` // ISO8601 format, e.g., "2025-01-09T15:04:05Z07:00"
	EndTime   time.Time `json:"endTime"`   // ISO8601 format for consistency with StartTime
	RuntimeMs int64     `json:"runtimeMs"` // Runtime in milliseconds for better precision
}

type ResponseAPI struct {
	RequestID string            `json:"requestId"`
	Data      any               `json:"data"`
	Message   string            `json:"message"`
	Debug     *ResponseAPIDebug `json:"debug,omitempty"`
}

type StreamChunk struct {
	JSONBuf *[]byte // Pointer to pooled buffer (STACK-FRIENDLY)
	Error   error   // Error if any occurred during processing
}

// StreamResponse represents a streaming response configuration
type StreamResponse struct {
	TotalCount int64              // Total count of records (sent as X-Total-Count header)
	ChunkChan  <-chan StreamChunk // Channel to receive data chunks
	Error      error              // Error to return if streaming fails before starting
	Code       int                // HTTP status code (default 200)
}

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate 4KB buffer (enough for ~10 tickets)
		buf := make([]byte, 0, 4096)
		return &buf
	},
}
