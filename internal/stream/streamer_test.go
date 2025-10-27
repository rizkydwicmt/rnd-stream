package stream

import (
	"context"
	"fmt"
	"testing"
	"time"

	json "github.com/json-iterator/go"
)

// TestStreamer_Stream tests basic streaming functionality
func TestStreamer_Stream(t *testing.T) {
	ctx := context.Background()
	config := DefaultChunkConfig()
	config.ChunkThreshold = 100 // Small threshold for testing
	streamer := NewStreamer[int](config)

	t.Run("streams items successfully", func(t *testing.T) {
		// Create fetcher that sends 10 items
		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 10)
			errChan := make(chan error, 1)

			go func() {
				defer close(dataChan)
				defer close(errChan)

				for i := 1; i <= 10; i++ {
					dataChan <- i
				}
			}()

			return dataChan, errChan
		}

		// Create simple transformer
		transformer := func(item int) (interface{}, error) {
			return map[string]int{"value": item}, nil
		}

		// Stream
		resp := streamer.Stream(ctx, fetcher, transformer)

		if resp.Code != 200 {
			t.Errorf("Expected code 200, got %d", resp.Code)
		}

		if resp.Error != nil {
			t.Errorf("Expected no error, got %v", resp.Error)
		}

		// Collect chunks
		var allData []byte
		for chunk := range resp.ChunkChan {
			if chunk.Error != nil {
				t.Fatalf("Chunk error: %v", chunk.Error)
			}

			if chunk.JSONBuf != nil {
				allData = append(allData, *chunk.JSONBuf...)
			}
		}

		// Parse JSON array
		var result []map[string]int
		if err := json.Unmarshal(allData, &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v\nData: %s", err, string(allData))
		}

		// Verify results
		if len(result) != 10 {
			t.Errorf("Expected 10 items, got %d", len(result))
		}

		for i, item := range result {
			expected := i + 1
			if item["value"] != expected {
				t.Errorf("Item %d: expected value %d, got %d", i, expected, item["value"])
			}
		}
	})

	t.Run("handles empty data", func(t *testing.T) {
		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 1)
			errChan := make(chan error, 1)
			close(dataChan)
			close(errChan)
			return dataChan, errChan
		}

		transformer := PassThroughTransformer[int]()
		resp := streamer.Stream(ctx, fetcher, transformer)

		var allData []byte
		for chunk := range resp.ChunkChan {
			if chunk.JSONBuf != nil {
				allData = append(allData, *chunk.JSONBuf...)
			}
		}

		// Should be empty array
		if string(allData) != "[]" {
			t.Errorf("Expected empty array [], got %s", string(allData))
		}
	})

	t.Run("handles fetcher error", func(t *testing.T) {
		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 1)
			errChan := make(chan error, 1)

			go func() {
				defer close(dataChan)
				defer close(errChan)
				errChan <- fmt.Errorf("test error")
			}()

			return dataChan, errChan
		}

		transformer := PassThroughTransformer[int]()
		resp := streamer.Stream(ctx, fetcher, transformer)

		// Should receive error in chunk
		gotError := false
		for chunk := range resp.ChunkChan {
			if chunk.Error != nil {
				gotError = true
			}
		}

		if !gotError {
			t.Error("Expected to receive error from fetcher")
		}
	})

	t.Run("handles transformer error", func(t *testing.T) {
		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 1)
			errChan := make(chan error, 1)

			go func() {
				defer close(dataChan)
				defer close(errChan)
				dataChan <- 1
			}()

			return dataChan, errChan
		}

		transformer := func(item int) (interface{}, error) {
			return nil, fmt.Errorf("transform error")
		}

		resp := streamer.Stream(ctx, fetcher, transformer)

		// Should receive error in chunk
		gotError := false
		for chunk := range resp.ChunkChan {
			if chunk.Error != nil {
				gotError = true
			}
		}

		if !gotError {
			t.Error("Expected to receive error from transformer")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 1)
			errChan := make(chan error, 1)

			go func() {
				defer close(dataChan)
				defer close(errChan)

				for i := 0; i < 1000; i++ {
					select {
					case dataChan <- i:
					case <-ctx.Done():
						return
					}
					time.Sleep(1 * time.Millisecond)
				}
			}()

			return dataChan, errChan
		}

		transformer := PassThroughTransformer[int]()
		resp := streamer.Stream(ctx, fetcher, transformer)

		// Cancel after small delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		// Should stop early
		count := 0
		for chunk := range resp.ChunkChan {
			if chunk.JSONBuf != nil {
				count++
			}
		}

		// Should not receive all 1000 items
		if count >= 100 {
			t.Errorf("Expected early termination, got %d chunks", count)
		}
	})
}

// TestStreamer_StreamBatch tests batch streaming functionality
func TestStreamer_StreamBatch(t *testing.T) {
	ctx := context.Background()
	config := DefaultChunkConfig()
	config.ChunkThreshold = 200
	streamer := NewStreamer[int](config)

	t.Run("streams batches successfully", func(t *testing.T) {
		fetcher := func(ctx context.Context) (<-chan []int, <-chan error) {
			batchChan := make(chan []int, 2)
			errChan := make(chan error, 1)

			go func() {
				defer close(batchChan)
				defer close(errChan)

				batchChan <- []int{1, 2, 3}
				batchChan <- []int{4, 5, 6}
			}()

			return batchChan, errChan
		}

		transformer := func(items []int) ([]interface{}, error) {
			result := make([]interface{}, len(items))
			for i, item := range items {
				result[i] = map[string]int{"value": item * 2}
			}
			return result, nil
		}

		resp := streamer.StreamBatch(ctx, fetcher, transformer)

		var allData []byte
		for chunk := range resp.ChunkChan {
			if chunk.Error != nil {
				t.Fatalf("Chunk error: %v", chunk.Error)
			}
			if chunk.JSONBuf != nil {
				allData = append(allData, *chunk.JSONBuf...)
			}
		}

		var result []map[string]int
		if err := json.Unmarshal(allData, &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if len(result) != 6 {
			t.Errorf("Expected 6 items, got %d", len(result))
		}

		// Verify transformation (values should be doubled)
		for i, item := range result {
			expected := (i + 1) * 2
			if item["value"] != expected {
				t.Errorf("Item %d: expected %d, got %d", i, expected, item["value"])
			}
		}
	})
}

// TestBufferPool tests buffer pool functionality
func TestBufferPool(t *testing.T) {
	t.Run("creates pool with correct size", func(t *testing.T) {
		pool := NewBufferPool(1024)

		if pool.GetInitialSize() != 1024 {
			t.Errorf("Expected initial size 1024, got %d", pool.GetInitialSize())
		}
	})

	t.Run("gets and puts buffers", func(t *testing.T) {
		pool := NewBufferPool(100)

		buf1 := pool.Get()
		if buf1 == nil {
			t.Fatal("Expected non-nil buffer")
		}

		if len(*buf1) != 0 {
			t.Errorf("Expected empty buffer, got len=%d", len(*buf1))
		}

		if cap(*buf1) < 100 {
			t.Errorf("Expected capacity >= 100, got %d", cap(*buf1))
		}

		// Use buffer
		*buf1 = append(*buf1, []byte("test")...)

		// Put back
		pool.Put(buf1)

		// Get again
		buf2 := pool.Get()

		// Should be reset to zero length
		if len(*buf2) != 0 {
			t.Errorf("Expected reset buffer, got len=%d", len(*buf2))
		}
	})

	t.Run("handles nil put gracefully", func(t *testing.T) {
		pool := NewBufferPool(100)

		// Should not panic
		pool.Put(nil)
	})

	t.Run("uses default size for invalid size", func(t *testing.T) {
		pool := NewBufferPool(-1)

		if pool.GetInitialSize() != 50*1024 {
			t.Errorf("Expected default size 50KB, got %d", pool.GetInitialSize())
		}
	})
}

// TestChunkConfig tests configuration validation
func TestChunkConfig(t *testing.T) {
	t.Run("validates and applies defaults", func(t *testing.T) {
		config := ChunkConfig{}

		err := config.Validate()
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		if config.ChunkThreshold != 32*1024 {
			t.Errorf("Expected default ChunkThreshold 32KB, got %d", config.ChunkThreshold)
		}

		if config.BatchSize != 1000 {
			t.Errorf("Expected default BatchSize 1000, got %d", config.BatchSize)
		}

		if config.BufferSize != 50*1024 {
			t.Errorf("Expected default BufferSize 50KB, got %d", config.BufferSize)
		}

		if config.ChannelBuffer != 4 {
			t.Errorf("Expected default ChannelBuffer 4, got %d", config.ChannelBuffer)
		}
	})

	t.Run("preserves valid values", func(t *testing.T) {
		config := ChunkConfig{
			ChunkThreshold: 64 * 1024,
			BatchSize:      500,
			BufferSize:     128 * 1024,
			ChannelBuffer:  8,
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		if config.ChunkThreshold != 64*1024 {
			t.Error("ChunkThreshold was changed")
		}

		if config.BatchSize != 500 {
			t.Error("BatchSize was changed")
		}
	})
}

// TestHelpers tests helper functions
func TestSliceFetcher(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	fetcher := SliceFetcher(items)
	dataChan, errChan := fetcher(ctx)

	var received []int
	for item := range dataChan {
		received = append(received, item)
	}

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	default:
	}

	if len(received) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(received))
	}

	for i, item := range received {
		if item != items[i] {
			t.Errorf("Item %d: expected %d, got %d", i, items[i], item)
		}
	}
}

func TestSliceBatchFetcher(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	fetcher := SliceBatchFetcher(items, 3)
	batchChan, errChan := fetcher(ctx)

	var allItems []int
	batchCount := 0

	for batch := range batchChan {
		batchCount++
		allItems = append(allItems, batch...)
	}

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	default:
	}

	// Should have 4 batches: [1,2,3], [4,5,6], [7,8,9], [10]
	if batchCount != 4 {
		t.Errorf("Expected 4 batches, got %d", batchCount)
	}

	if len(allItems) != len(items) {
		t.Errorf("Expected %d total items, got %d", len(items), len(allItems))
	}
}

func TestPassThroughTransformer(t *testing.T) {
	transformer := PassThroughTransformer[string]()

	result, err := transformer("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}
}

// BenchmarkStreamer benchmarks streaming performance
func BenchmarkStreamer_Stream(b *testing.B) {
	ctx := context.Background()
	config := DefaultChunkConfig()
	streamer := NewStreamer[int](config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fetcher := func(ctx context.Context) (<-chan int, <-chan error) {
			dataChan := make(chan int, 100)
			errChan := make(chan error, 1)

			go func() {
				defer close(dataChan)
				defer close(errChan)

				for j := 0; j < 100; j++ {
					dataChan <- j
				}
			}()

			return dataChan, errChan
		}

		transformer := PassThroughTransformer[int]()
		resp := streamer.Stream(ctx, fetcher, transformer)

		// Consume chunks
		for chunk := range resp.ChunkChan {
			_ = chunk
		}
	}
}
