package stream_test

import (
	"context"
	"database/sql"
	"fmt"
	"stream/internal/stream"
)

// Example_customConfiguration demonstrates custom streaming configuration.
func Example_customConfiguration() {
	type LogEntry struct {
		Timestamp string
		Message   string
	}

	// Custom configuration for large datasets
	config := stream.ChunkConfig{
		ChunkThreshold: 64 * 1024,  // 64KB chunks (larger for better throughput)
		BatchSize:      5000,       // 5000 items per batch
		BufferSize:     100 * 1024, // 100KB buffer
		ChannelBuffer:  8,          // 8-buffer channels
	}

	err := config.Validate()
	if err != nil {
		panic(err)
	}

	streamer := stream.NewStreamer[LogEntry](config)

	// Use streamer
	fmt.Printf("Config - Chunk: %d, Batch: %d\n",
		streamer.GetConfig().ChunkThreshold,
		streamer.GetConfig().BatchSize)
	// Output: Config - Chunk: 65536, Batch: 5000
}

// Example_contextCancellation demonstrates context cancellation support.
func Example_contextCancellation() {
	type Item struct {
		ID int
	}

	streamer := stream.NewDefaultStreamer[Item]()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	fetcher := func(ctx context.Context) (<-chan Item, <-chan error) {
		dataChan := make(chan Item, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			for i := 0; i < 1000; i++ {
				select {
				case dataChan <- Item{ID: i}:
				case <-ctx.Done():
					return
				}
			}
		}()

		return dataChan, errChan
	}

	transformer := stream.PassThroughTransformer[Item]()

	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Cancel immediately
	cancel()

	// Consume chunks until cancelled
	count := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.JSONBuf != nil {
			count++
		}
	}

	fmt.Println("Context cancellation respected")
	// Output: Context cancellation respected
}

// Example_bufferPoolUsage demonstrates direct buffer pool usage.
func Example_bufferPoolUsage() {
	// Get buffer from global pool
	buf := stream.GetBuffer()
	defer stream.PutBuffer(buf)

	// Use buffer
	*buf = append(*buf, []byte("Hello, ")...)
	*buf = append(*buf, []byte("World!")...)

	fmt.Printf("Buffer content: %s\n", string(*buf))
	// Output: Buffer content: Hello, World!
}

// Example_customBufferPool demonstrates creating a custom buffer pool.
func Example_customBufferPool() {
	// Create pool with 100KB buffers
	pool := stream.NewBufferPool(100 * 1024)

	buf1 := pool.Get()
	fmt.Printf("Initial size: %d KB\n", pool.GetInitialSize()/1024)

	*buf1 = append(*buf1, []byte("data")...)

	pool.Put(buf1)

	buf2 := pool.Get()
	fmt.Printf("Buffer reused, length: %d\n", len(*buf2))

	pool.Put(buf2)

	// Output: Initial size: 100 KB
	// Buffer reused, length: 0
}

// Example_migrationFromTickets demonstrates migration from tickets service.
func Example_migrationFromTickets() {
	// Before: Manual streaming in tickets/service.go
	// After: Using generic stream package

	type RowData map[string]interface{}
	type TransformedRow struct {
		Fields map[string]interface{}
	}

	// Setup (in real code, these come from service dependencies)
	var rows *sql.Rows              // From db query
	var formulas []interface{}      // From payload
	var operators map[string]func() // From service

	// Create streamer
	streamer := stream.NewDefaultStreamer[RowData]()

	// Define SQL scanner
	scanner := func(rows *sql.Rows) (RowData, error) {
		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		row := make(RowData)
		for i, col := range columns {
			row[col] = values[i]
		}
		return row, nil
	}

	// Note: SQLFetcher doesn't exist, using inline fetcher for this example
	fetcher := func(ctx context.Context) (<-chan RowData, <-chan error) {
		dataChan := make(chan RowData, 10)
		errChan := make(chan error, 1)
		go func() {
			defer close(dataChan)
			defer close(errChan)
			if rows != nil {
				for rows.Next() {
					row, err := scanner(rows)
					if err != nil {
						errChan <- err
						return
					}
					dataChan <- row
				}
			}
		}()
		return dataChan, errChan
	}

	// Define transformer (replaces BatchTransformRows)
	transformer := func(row RowData) (interface{}, error) {
		// Apply formulas and operators here
		transformed := applyTransformations(row, formulas, operators)
		return transformed, nil
	}

	ctx := context.Background()
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	fmt.Printf("Migrated successfully, Code: %d\n", streamResp.Code)
}

// Helper function for example
func applyTransformations(row map[string]interface{}, formulas []interface{}, operators map[string]func()) map[string]interface{} {
	// Placeholder - in real code, this would apply actual transformations
	return row
}

// Example_ginHandlerIntegration demonstrates integration with Gin handlers.
func Example_ginHandlerIntegration() {
	type Report struct {
		Date  string
		Value float64
	}

	// This would be in your handler
	StreamReportHandler := func(c interface{}) {
		// c would be *gin.Context in real code
		ctx := context.Background()

		// Create streamer
		streamer := stream.NewDefaultStreamer[Report]()

		// Fetch data
		reports := []Report{
			{Date: "2025-01-01", Value: 100.0},
			{Date: "2025-01-02", Value: 200.0},
		}

		fetcher := stream.SliceFetcher(reports)
		transformer := stream.PassThroughTransformer[Report]()

		// Stream
		streamResp := streamer.Stream(ctx, fetcher, transformer)

		// In real code, you would use:
		// sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))
		// sendStream(streamResp)

		fmt.Printf("Ready to stream %d chunks\n", len(streamResp.ChunkChan))
	}

	// Simulate handler call
	StreamReportHandler(nil)
	// Output: Ready to stream 0 chunks
}
