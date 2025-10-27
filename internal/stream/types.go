// Package stream provides a reusable streaming framework for Go applications.
// It abstracts the complexity of streaming large datasets with efficient memory management,
// buffer pooling, and chunked JSON encoding.
//
// Key Features:
// - Generic streaming interface for any data type
// - Efficient buffer pooling to minimize GC pressure
// - Configurable chunk sizes and batch processing
// - Context-aware cancellation
// - Compatible with existing middleware.StreamResponse
//
// Usage Example:
//
//	// Create a streamer with custom configuration
//	config := stream.ChunkConfig{
//	    ChunkThreshold: 32 * 1024,  // 32KB chunks
//	    BatchSize:      1000,        // Process 1000 items at a time
//	    BufferSize:     50 * 1024,   // 50KB initial buffer
//	}
//
//	streamer := stream.NewStreamer(config)
//
//	// Define data fetcher
//	fetcher := func(ctx context.Context) (<-chan YourDataType, <-chan error) {
//	    dataChan := make(chan YourDataType, 10)
//	    errChan := make(chan error, 1)
//	    go func() {
//	        // Fetch data from your source
//	        // Send to dataChan
//	        // Close channels when done
//	    }()
//	    return dataChan, errChan
//	}
//
//	// Define transformer
//	transformer := func(item YourDataType) (interface{}, error) {
//	    // Transform item to desired output format
//	    return transformedItem, nil
//	}
//
//	// Stream data
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
package stream

import (
	"context"
	"stream/middleware"
)

// DataFetcher is a function that fetches data from a source and sends it to a channel.
// It should close both channels when done or on error.
// The data channel should send individual items of type T.
// The error channel should send at most one error.
//
// Type Parameters:
//   - T: The type of data items being fetched
//
// Returns:
//   - dataChan: Channel that receives data items
//   - errChan: Channel that receives errors (buffered with capacity 1)
//
// Implementation Notes:
//   - MUST close both channels when done
//   - Should respect context cancellation
//   - Error channel should be buffered to avoid goroutine leaks
//   - Data channel buffer size affects memory usage
type DataFetcher[T any] func(ctx context.Context) (<-chan T, <-chan error)

// Transformer is a function that transforms a single data item into JSON-encodable output.
// It receives an item of type T and returns the transformed result.
//
// Type Parameters:
//   - T: The input type (raw data item)
//
// Returns:
//   - interface{}: The transformed output (must be JSON-encodable)
//   - error: Error if transformation fails
//
// Implementation Notes:
//   - Should be stateless and thread-safe
//   - Return value MUST be JSON-encodable
//   - Errors cause streaming to stop immediately
//   - For pass-through, return input unchanged
type Transformer[T any] func(item T) (interface{}, error)

// BatchFetcher is a function that fetches data in batches for more efficient processing.
// Similar to DataFetcher but sends slices of items instead of individual items.
//
// Type Parameters:
//   - T: The type of data items being fetched
//
// Returns:
//   - batchChan: Channel that receives batches of data items
//   - errChan: Channel that receives errors (buffered with capacity 1)
//
// Use Cases:
//   - When data source naturally returns batches (e.g., database query results)
//   - When batch processing is more efficient than item-by-item
//
// Implementation Notes:
//   - Batch size can vary between sends
//   - MUST close both channels when done
//   - Should respect context cancellation
type BatchFetcher[T any] func(ctx context.Context) (<-chan []T, <-chan error)

// BatchTransformer is a function that transforms a batch of items efficiently.
// Useful when transformation can be optimized for batches (e.g., bulk operations).
//
// Type Parameters:
//   - T: The input type (raw data item)
//
// Returns:
//   - []interface{}: Slice of transformed outputs (must be JSON-encodable)
//   - error: Error if transformation fails
//
// Use Cases:
//   - When transformation involves expensive setup that can be amortized
//   - When bulk transformations are more efficient (e.g., batch API calls)
//
// Implementation Notes:
//   - Output slice length SHOULD match input slice length
//   - Should be stateless and thread-safe
//   - Errors cause streaming to stop immediately
type BatchTransformer[T any] func(items []T) ([]interface{}, error)

// Streamer is the main interface for streaming data with efficient memory management.
// It handles buffering, chunking, and JSON encoding transparently.
//
// Type Parameters:
//   - T: The type of data items being streamed
//
// Methods:
//   - Stream: Stream individual items with item-by-item transformation
//   - StreamBatch: Stream batches with batch transformation
//
// Implementation Notes:
//   - Implementations MUST be safe for concurrent use
//   - Should handle context cancellation gracefully
//   - Should close all channels when done
type Streamer[T any] interface {
	// Stream processes data items one-by-one and returns a StreamResponse.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - fetcher: Function that fetches data items
	//   - transformer: Function that transforms each item
	//
	// Returns:
	//   - StreamResponse: Response compatible with middleware.sendStream
	//
	// Behavior:
	//   - Respects context cancellation
	//   - Stops on first error from fetcher or transformer
	//   - Buffers data up to ChunkThreshold before sending
	//   - Uses buffer pool to minimize allocations
	Stream(ctx context.Context, fetcher DataFetcher[T], transformer Transformer[T]) middleware.StreamResponse

	// StreamBatch processes data in batches for more efficient transformation.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - fetcher: Function that fetches data batches
	//   - transformer: Function that transforms each batch
	//
	// Returns:
	//   - StreamResponse: Response compatible with middleware.sendStream
	//
	// Behavior:
	//   - Same as Stream() but with batch processing
	//   - Useful when transformation is more efficient in batches
	StreamBatch(ctx context.Context, fetcher BatchFetcher[T], transformer BatchTransformer[T]) middleware.StreamResponse

	// GetConfig returns the current streaming configuration
	GetConfig() ChunkConfig
}

// ChunkConfig defines configuration for chunk-based streaming.
// All fields are optional and have sensible defaults.
type ChunkConfig struct {
	// ChunkThreshold is the size in bytes at which a chunk is sent.
	// When the JSON buffer exceeds this size, it's flushed to the client.
	//
	// Default: 32 * 1024 (32KB)
	// Recommended: 16KB - 64KB
	//
	// Tradeoffs:
	//   - Smaller: More frequent flushes, lower memory, higher overhead
	//   - Larger: Fewer flushes, higher memory, lower overhead
	ChunkThreshold int

	// BatchSize is the number of items to process in a single batch.
	// Only applies when using StreamBatch().
	//
	// Default: 1000
	// Recommended: 100 - 5000 depending on item size
	//
	// Tradeoffs:
	//   - Smaller: More frequent processing, lower peak memory
	//   - Larger: Less frequent processing, higher peak memory, better throughput
	BatchSize int

	// BufferSize is the initial capacity of JSON buffers from the pool.
	//
	// Default: 50 * 1024 (50KB)
	// Recommended: Same as or slightly larger than ChunkThreshold
	//
	// Tradeoffs:
	//   - Smaller: Lower memory per request, may require reallocations
	//   - Larger: Higher memory per request, fewer reallocations
	BufferSize int

	// ChannelBuffer is the buffer size for internal channels.
	//
	// Default: 4
	// Recommended: 2 - 10
	//
	// Tradeoffs:
	//   - Smaller: Lower memory, more blocking
	//   - Larger: Higher memory, less blocking
	ChannelBuffer int
}

// DefaultChunkConfig returns the default streaming configuration.
// These values are optimized based on benchmarks in BUFFER_POOL_ANALYSIS.md
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		ChunkThreshold: 32 * 1024, // 32KB - optimal for network packets
		BatchSize:      1000,      // 1000 items - balances memory and throughput
		BufferSize:     50 * 1024, // 50KB - proven optimal in benchmarks
		ChannelBuffer:  4,         // 4 - enough to prevent blocking
	}
}

// Validate checks if the configuration is valid and applies defaults.
// It returns an error if any value is invalid.
func (c *ChunkConfig) Validate() error {
	// Apply defaults for zero values
	if c.ChunkThreshold <= 0 {
		c.ChunkThreshold = 32 * 1024
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1000
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 50 * 1024
	}
	if c.ChannelBuffer <= 0 {
		c.ChannelBuffer = 4
	}

	// No validation errors for now
	// Could add max limits if needed
	return nil
}

// BufferPool manages a pool of byte buffers to reduce allocations.
// It uses sync.Pool internally for efficient reuse.
type BufferPool interface {
	// Get retrieves a buffer from the pool.
	// The buffer is reset to zero length but retains its capacity.
	//
	// Returns:
	//   - *[]byte: Pointer to pooled buffer (capacity >= initial size)
	//
	// Usage:
	//   buf := pool.Get()
	//   defer pool.Put(buf)
	//   *buf = append(*buf, data...)
	Get() *[]byte

	// Put returns a buffer to the pool for reuse.
	// The buffer should not be used after calling Put().
	//
	// Parameters:
	//   - buf: Pointer to buffer to return to pool
	//
	// Safety:
	//   - Buffer MUST NOT be used after Put()
	//   - Calling Put() multiple times with same buffer is undefined
	//   - Passing nil is safe (no-op)
	Put(buf *[]byte)

	// GetInitialSize returns the initial capacity of buffers from this pool.
	GetInitialSize() int
}
