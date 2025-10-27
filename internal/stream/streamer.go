package stream

import (
	"context"
	"fmt"
	"net/http"
	"stream/middleware"

	json "github.com/json-iterator/go"
)

// streamer is the default implementation of the Streamer interface.
// It provides efficient streaming with buffer pooling and chunked encoding.
//
// Architecture:
//   - Uses goroutines for concurrent processing
//   - Buffer pool minimizes GC pressure
//   - Chunks data at configurable threshold
//   - Compatible with middleware.StreamResponse
//
// Thread Safety:
//   - Safe for concurrent use
//   - Each Stream() call runs in isolation
//   - BufferPool is thread-safe via sync.Pool
type streamer[T any] struct {
	config     ChunkConfig
	bufferPool BufferPool
}

// NewStreamer creates a new Streamer with the given configuration.
//
// Parameters:
//   - config: Streaming configuration (chunk size, batch size, etc.)
//
// Returns:
//   - Streamer[T]: Ready-to-use streamer for type T
//
// Usage:
//
//	config := stream.DefaultChunkConfig()
//	streamer := stream.NewStreamer[MyDataType](config)
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
//
// Type Parameters:
//   - T: The type of data items being streamed
func NewStreamer[T any](config ChunkConfig) Streamer[T] {
	// Validate and apply defaults
	if err := config.Validate(); err != nil {
		// Should never happen with current validation logic
		panic(fmt.Sprintf("invalid config: %v", err))
	}

	return &streamer[T]{
		config:     config,
		bufferPool: NewBufferPool(config.BufferSize),
	}
}

// Stream processes individual data items and returns a StreamResponse.
// This is the main method for streaming data with item-by-item transformation.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - fetcher: Function that provides data items
//   - transformer: Function that transforms each item
//
// Returns:
//   - StreamResponse: Compatible with middleware.sendStream()
//
// Flow:
//  1. Start goroutine for processing
//  2. Fetch data from fetcher
//  3. Transform each item
//  4. Encode to JSON
//  5. Buffer until chunk threshold
//  6. Send chunk when threshold reached
//  7. Close and cleanup when done
//
// Error Handling:
//   - Stops on first error from fetcher or transformer
//   - Sends error via StreamChunk
//   - Closes all channels
//   - Cleans up resources
//
// Context Cancellation:
//   - Respects ctx.Done()
//   - Stops processing immediately
//   - Cleans up resources
func (s *streamer[T]) Stream(
	ctx context.Context,
	fetcher DataFetcher[T],
	transformer Transformer[T],
) middleware.StreamResponse {
	chunkChan := make(chan middleware.StreamChunk, s.config.ChannelBuffer)

	go func() {
		defer close(chunkChan)

		// Get buffer from pool
		jsonBuf := s.bufferPool.Get()
		defer func() {
			if jsonBuf != nil {
				s.bufferPool.Put(jsonBuf)
			}
		}()

		// Start JSON array
		*jsonBuf = append(*jsonBuf, '[')

		// Fetch data
		dataChan, errChan := fetcher(ctx)

		firstItem := true

		for {
			select {
			case <-ctx.Done():
				// Context cancelled
				return

			case err := <-errChan:
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("fetcher error: %w", err),
					}
					return
				}

			case item, ok := <-dataChan:
				if !ok {
					// Channel closed, all items processed
					// Close JSON array
					*jsonBuf = append(*jsonBuf, ']')

					// Send final chunk
					chunkChan <- middleware.StreamChunk{
						JSONBuf: jsonBuf,
					}
					jsonBuf = nil // Prevent double-put in defer
					return
				}

				// Transform item
				transformed, err := transformer(item)
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("transformer error: %w", err),
					}
					return
				}

				// Encode to JSON
				jsonData, err := json.Marshal(transformed)
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("JSON marshal error: %w", err),
					}
					return
				}

				// Add comma separator if not first item
				if !firstItem {
					*jsonBuf = append(*jsonBuf, ',')
				} else {
					firstItem = false
				}

				// Append JSON data
				*jsonBuf = append(*jsonBuf, jsonData...)

				// Send chunk if threshold exceeded
				if len(*jsonBuf) > s.config.ChunkThreshold {
					chunkChan <- middleware.StreamChunk{
						JSONBuf: jsonBuf,
					}

					// Get new buffer for next chunk
					jsonBuf = s.bufferPool.Get()
					*jsonBuf = (*jsonBuf)[:0]
				}
			}
		}
	}()

	return middleware.StreamResponse{
		TotalCount: -1, // Not known in advance for streaming
		ChunkChan:  chunkChan,
		Code:       http.StatusOK,
		Error:      nil,
	}
}

// StreamBatch processes data in batches for more efficient transformation.
// Use this when your transformer can process multiple items more efficiently.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - fetcher: Function that provides data batches
//   - transformer: Function that transforms each batch
//
// Returns:
//   - StreamResponse: Compatible with middleware.sendStream()
//
// Flow:
//  1. Start goroutine for processing
//  2. Fetch batch from fetcher
//  3. Transform entire batch
//  4. Encode each transformed item to JSON
//  5. Buffer until chunk threshold
//  6. Send chunk when threshold reached
//  7. Close and cleanup when done
//
// Use Cases:
//   - Database queries returning batches
//   - Batch API calls
//   - Expensive transformation setup
//
// Performance:
//   - More efficient when transformation has setup cost
//   - Reduces function call overhead
//   - May use more memory for large batches
func (s *streamer[T]) StreamBatch(
	ctx context.Context,
	fetcher BatchFetcher[T],
	transformer BatchTransformer[T],
) middleware.StreamResponse {
	chunkChan := make(chan middleware.StreamChunk, s.config.ChannelBuffer)

	go func() {
		defer close(chunkChan)

		// Get buffer from pool
		jsonBuf := s.bufferPool.Get()
		defer func() {
			if jsonBuf != nil {
				s.bufferPool.Put(jsonBuf)
			}
		}()

		// Start JSON array
		*jsonBuf = append(*jsonBuf, '[')

		// Fetch batches
		batchChan, errChan := fetcher(ctx)

		firstItem := true

		for {
			select {
			case <-ctx.Done():
				// Context cancelled
				return

			case err := <-errChan:
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("batch fetcher error: %w", err),
					}
					return
				}

			case batch, ok := <-batchChan:
				if !ok {
					// Channel closed, all batches processed
					// Close JSON array
					*jsonBuf = append(*jsonBuf, ']')

					// Send final chunk
					chunkChan <- middleware.StreamChunk{
						JSONBuf: jsonBuf,
					}
					jsonBuf = nil // Prevent double-put in defer
					return
				}

				// Transform batch
				transformed, err := transformer(batch)
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("batch transformer error: %w", err),
					}
					return
				}

				// Encode each transformed item
				for _, item := range transformed {
					jsonData, err := json.Marshal(item)
					if err != nil {
						chunkChan <- middleware.StreamChunk{
							Error: fmt.Errorf("JSON marshal error: %w", err),
						}
						return
					}

					// Add comma separator if not first item
					if !firstItem {
						*jsonBuf = append(*jsonBuf, ',')
					} else {
						firstItem = false
					}

					// Append JSON data
					*jsonBuf = append(*jsonBuf, jsonData...)

					// Send chunk if threshold exceeded
					if len(*jsonBuf) > s.config.ChunkThreshold {
						chunkChan <- middleware.StreamChunk{
							JSONBuf: jsonBuf,
						}

						// Get new buffer for next chunk
						jsonBuf = s.bufferPool.Get()
						*jsonBuf = (*jsonBuf)[:0]
					}
				}
			}
		}
	}()

	return middleware.StreamResponse{
		TotalCount: -1, // Not known in advance for streaming
		ChunkChan:  chunkChan,
		Code:       http.StatusOK,
		Error:      nil,
	}
}

// GetConfig returns the current streaming configuration.
//
// Returns:
//   - ChunkConfig: Current configuration
//
// Use Cases:
//   - Debugging
//   - Metrics
//   - Configuration validation
func (s *streamer[T]) GetConfig() ChunkConfig {
	return s.config
}

// NewDefaultStreamer creates a streamer with default configuration.
// Convenience wrapper for NewStreamer(DefaultChunkConfig()).
//
// Returns:
//   - Streamer[T]: Streamer with default settings
//
// Usage:
//
//	streamer := stream.NewDefaultStreamer[MyDataType]()
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
//
// Default Configuration:
//   - ChunkThreshold: 32KB
//   - BatchSize: 1000
//   - BufferSize: 50KB
//   - ChannelBuffer: 4
func NewDefaultStreamer[T any]() Streamer[T] {
	return NewStreamer[T](DefaultChunkConfig())
}
