package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

func setResponseDefaults(r Response) {
	if r.Message == "" {
		r.Message = "Success"
	}
	if r.Code == 0 {
		r.Code = http.StatusOK
	}
}

func logResponseError(c *gin.Context, r Response) {
	if r.Error == nil {
		return
	}

	requestPath := c.Request.URL.Path
	requestID := c.GetString("requestId")
	fmt.Printf("RequestID: %v, Path: %v, ResponseCode %v, Error: %v", requestID, requestPath, r.Code, r.Error)
}

func getStartTime(c *gin.Context) time.Time {
	if value, exists := c.Get("start-time"); exists || value != nil {
		if t, ok := value.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}

func buildDebugInfo(c *gin.Context, r Response) *ResponseAPIDebug {
	startTime := getStartTime(c)
	endTime := time.Now()
	err := r.Error.Error()

	return &ResponseAPIDebug{
		Version:   c.GetString("version"),
		StartTime: startTime,
		EndTime:   endTime,
		RuntimeMs: endTime.Sub(startTime).Milliseconds(),
		Error: func() *string {
			if r.Error != nil {
				return &err
			}
			return nil
		}(),
	}
}

func buildResponseAPI(c *gin.Context, r Response, shouldDebug bool) ResponseAPI {
	response := ResponseAPI{
		RequestID: c.GetString("requestId"),
		Message:   r.Message,
		Data:      r.Data,
	}

	if shouldDebug {
		response.Debug = buildDebugInfo(c, r)
	}

	return response
}

func send(c *gin.Context, shouldDebug bool) func(r Response) {
	return func(r Response) {
		setResponseDefaults(r)
		logResponseError(c, r)
		response := buildResponseAPI(c, r, shouldDebug)

		c.Abort()
		c.JSON(r.Code, response)
	}
}

func RequestInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("requestId", uuid.New().String())
		version := c.Request.Header.Get("version")
		if version == "" {
			version = "1.0.0"
		}
		c.Set("version", version)
		c.Set("start-time", time.Now())
		c.Next()
	}
}

// sendStream handles streaming responses with proper buffer management
// Follows the same pattern as send() for consistency
func sendStream(c *gin.Context, shouldDebug bool) func(r StreamResponse) {
	return func(r StreamResponse) {
		if r.Code == 0 {
			r.Code = http.StatusOK
		}

		if r.Error != nil {
			send(c, shouldDebug)(Response{
				Code:    r.Code,
				Message: "Stream failed",
				Error:   r.Error,
			})
			return
		}

		c.Header("Content-Type", "application/json")
		c.Header("X-Total-Count", fmt.Sprintf("%d", r.TotalCount))

		writer := c.Writer
		firstRecord := true

		for chunk := range r.ChunkChan {
			select {
			case <-c.Request.Context().Done():
				requestID := c.GetString("requestId")
				fmt.Printf("RequestID: %v, Context canceled: %v\n", requestID, c.Request.Context().Err())
				return
			default:
			}

			if chunk.Error != nil {
				requestID := c.GetString("requestId")
				fmt.Printf("RequestID: %v, Stream error: %v\n", requestID, chunk.Error)
				if firstRecord {
					send(c, shouldDebug)(Response{
						Code:    r.Code,
						Message: "Stream failed",
						Error:   r.Error,
					})
					break
				}
				return
			}

			if chunk.JSONBuf != nil && len(*chunk.JSONBuf) > 0 {
				if !firstRecord && len(*chunk.JSONBuf) > 0 && (*chunk.JSONBuf)[0] == ',' {
					writer.Write(*chunk.JSONBuf)
				} else if !firstRecord {
					writer.Write([]byte(`,`))
					writer.Write(*chunk.JSONBuf)
				} else {
					c.Status(r.Code)
					writer.Write(*chunk.JSONBuf)
					firstRecord = false
				}

				jsonBufferPool.Put(chunk.JSONBuf)

				if flusher, ok := writer.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}

		if shouldDebug {
			startTime := getStartTime(c)
			endTime := time.Now()
			requestID := c.GetString("requestId")
			fmt.Printf("RequestID: %v, Stream completed, Runtime: %dms, TotalCount: %d\n",
				requestID,
				endTime.Sub(startTime).Milliseconds(),
				r.TotalCount,
			)
		}

		c.Abort()
	}
}

func ResponseInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		shouldDebug := gin.Mode() == gin.DebugMode
		c.Set("send", send(c, shouldDebug))
		c.Set("sendStream", sendStream(c, shouldDebug))
		c.Next()
	}
}
