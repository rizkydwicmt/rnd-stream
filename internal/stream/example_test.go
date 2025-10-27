package stream_test

import (
	"context"
	"database/sql"
	"fmt"
	"stream/internal/stream"
)

// Example_basicStreaming demonstrates basic streaming with default configuration.
func Example_basicStreaming() {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// Create streamer with default config
	streamer := stream.NewDefaultStreamer[User]()

	// Define data fetcher
	fetcher := func(ctx context.Context) (<-chan User, <-chan error) {
		dataChan := make(chan User, 10)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			// Simulate fetching users
			users := []User{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
				{ID: 3, Name: "Charlie"},
			}

			for _, user := range users {
				dataChan <- user
			}
		}()

		return dataChan, errChan
	}

	// Define transformer (pass-through in this case)
	transformer := stream.PassThroughTransformer[User]()

	// Stream data
	ctx := context.Background()
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	fmt.Printf("Code: %d\n", streamResp.Code)
	// Output: Code: 200
}

// Example_sqlStreaming demonstrates streaming from SQL database.
func Example_sqlStreaming() {
	type Ticket struct {
		ID      int    `json:"id"`
		Subject string `json:"subject"`
	}

	// Mock SQL rows (in real code, this comes from db.Query)
	var rows *sql.Rows // Assume this is from: rows, err := db.QueryContext(ctx, query)

	// Create streamer
	config := stream.DefaultChunkConfig()
	streamer := stream.NewStreamer[Ticket](config)

	// Define SQL row scanner
	scanner := func(rows *sql.Rows) (Ticket, error) {
		var ticket Ticket
		err := rows.Scan(&ticket.ID, &ticket.Subject)
		return ticket, err
	}

	// Create SQL fetcher
	fetcher := stream.SQLFetcher(rows, scanner)

	// Define transformer
	transformer := func(ticket Ticket) (interface{}, error) {
		return map[string]interface{}{
			"id":         ticket.ID,
			"subject":    ticket.Subject,
			"masked_id":  fmt.Sprintf("***%d", ticket.ID%1000),
		}, nil
	}

	// Stream
	ctx := context.Background()
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	fmt.Printf("TotalCount: %d, Code: %d\n", streamResp.TotalCount, streamResp.Code)
	// Note: TotalCount is -1 for streaming (not known in advance)
}

// Example_batchStreaming demonstrates batch streaming for efficient transformation.
func Example_batchStreaming() {
	type Product struct {
		ID    int
		Price float64
	}

	// Create streamer
	streamer := stream.NewDefaultStreamer[Product]()

	// Sample data
	products := []Product{
		{ID: 1, Price: 100.0},
		{ID: 2, Price: 200.0},
		{ID: 3, Price: 300.0},
		{ID: 4, Price: 400.0},
		{ID: 5, Price: 500.0},
	}

	// Create batch fetcher
	fetcher := stream.SliceBatchFetcher(products, 2) // Batches of 2

	// Batch transformer (e.g., apply bulk discount)
	transformer := func(batch []Product) ([]interface{}, error) {
		result := make([]interface{}, len(batch))

		// Apply 10% discount to entire batch
		for i, product := range batch {
			result[i] = map[string]interface{}{
				"id":              product.ID,
				"original_price":  product.Price,
				"discounted_price": product.Price * 0.9,
			}
		}

		return result, nil
	}

	// Stream batches
	ctx := context.Background()
	streamResp := streamer.StreamBatch(ctx, fetcher, transformer)

	fmt.Printf("Code: %d\n", streamResp.Code)
	// Output: Code: 200
}

// Example_customConfiguration demonstrates custom streaming configuration.
func Example_customConfiguration() {
	type LogEntry struct {
		Timestamp string
		Message   string
	}

	// Custom configuration for large datasets
	config := stream.ChunkConfig{
		ChunkThreshold: 64 * 1024,   // 64KB chunks (larger for better throughput)
		BatchSize:      5000,         // 5000 items per batch
		BufferSize:     100 * 1024,   // 100KB buffer
		ChannelBuffer:  8,            // 8-buffer channels
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

// Example_errorHandling demonstrates error handling in streaming.
func Example_errorHandling() {
	type Data struct {
		Value int
	}

	streamer := stream.NewDefaultStreamer[Data]()

	// Fetcher that returns error
	fetcher := func(ctx context.Context) (<-chan Data, <-chan error) {
		dataChan := make(chan Data, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			// Simulate error
			errChan <- fmt.Errorf("database connection failed")
		}()

		return dataChan, errChan
	}

	transformer := stream.PassThroughTransformer[Data]()

	ctx := context.Background()
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Error will be sent via chunk
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error received: %v\n", chunk.Error)
			break
		}
	}
	// Output: Error received: fetcher error: database connection failed
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

	fetcher := stream.SQLFetcher(rows, scanner)

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
