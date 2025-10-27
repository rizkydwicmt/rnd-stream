package stream_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"stream/internal/stream"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// ============================================================================
// Example 1: PassThroughTransformer
// ============================================================================

// Example_passThroughTransformer demonstrates using PassThroughTransformer
// for streaming data without transformation.
func Example_passThroughTransformer() {
	type Product struct {
		ID    int
		Name  string
		Price float64
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[Product]()

	// Create a simple fetcher
	fetcher := func(ctx context.Context) (<-chan Product, <-chan error) {
		dataChan := make(chan Product, 3)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			products := []Product{
				{ID: 1, Name: "Laptop", Price: 999.99},
				{ID: 2, Name: "Mouse", Price: 29.99},
			}

			for _, p := range products {
				dataChan <- p
			}
		}()

		return dataChan, errChan
	}

	// Use PassThroughTransformer - returns items unchanged
	transformer := stream.PassThroughTransformer[Product]()
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	count := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error: %v\n", chunk.Error)
			return
		}
		if chunk.JSONBuf != nil {
			count++
		}
	}

	fmt.Printf("PassThroughTransformer: streamed %d chunks successfully\n", count)
	// Output: PassThroughTransformer: streamed 1 chunks successfully
}

// ============================================================================
// Example 2: PassThroughBatchTransformer
// ============================================================================

// Example_passThroughBatchTransformer demonstrates using PassThroughBatchTransformer
// for batch streaming without transformation.
func Example_passThroughBatchTransformer() {
	type Order struct {
		OrderID int
		Amount  float64
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[Order]()

	// Create batch fetcher
	batchFetcher := func(ctx context.Context) (<-chan []Order, <-chan error) {
		batchChan := make(chan []Order, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)

			batch := []Order{
				{OrderID: 1, Amount: 100.50},
				{OrderID: 2, Amount: 250.75},
				{OrderID: 3, Amount: 99.99},
			}

			batchChan <- batch
		}()

		return batchChan, errChan
	}

	// Use PassThroughBatchTransformer
	transformer := stream.PassThroughBatchTransformer[Order]()
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	itemCount := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error: %v\n", chunk.Error)
			return
		}
		if chunk.JSONBuf != nil {
			itemCount++
		}
	}

	fmt.Printf("PassThroughBatchTransformer: processed batch with chunks=%d\n", itemCount)
	// Output: PassThroughBatchTransformer: processed batch with chunks=1
}

// ============================================================================
// Example 3: GenericRowScanner
// ============================================================================

// Example_genericRowScanner demonstrates using GenericRowScanner for
// scanning SQL rows into map[string]interface{}.
func Example_genericRowScanner() {
	// Create mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup mock expectations
	columns := []string{"id", "name", "email"}
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnRows(
			sqlmock.NewRows(columns).
				AddRow(1, "Alice", "alice@example.com").
				AddRow(2, "Bob", "bob@example.com"),
		)

	// Execute query
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// Create scanner using GenericRowScanner
	scanner := stream.GenericRowScanner()

	// Get columns
	cols, _ := rows.Columns()

	// Scan rows manually for demonstration
	count := 0
	for rows.Next() {
		rowData, err := scanner(rows, cols)
		if err != nil {
			panic(err)
		}

		if rowData["id"] != nil {
			count++
		}
	}

	fmt.Printf("GenericRowScanner: scanned %d rows\n", count)
	// Output: GenericRowScanner: scanned 2 rows
}

// ============================================================================
// Example 4: SQLFetcherWithColumns
// ============================================================================

// Example_sqlFetcherWithColumns demonstrates using SQLFetcherWithColumns
// for streaming SQL query results with column awareness.
func Example_sqlFetcherWithColumns() {
	// Use map[string]interface{} directly for this example

	// Create mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup mock expectations
	columns := []string{"user_id", "username", "status"}
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnRows(
			sqlmock.NewRows(columns).
				AddRow(101, "john_doe", "active").
				AddRow(102, "jane_doe", "active").
				AddRow(103, "admin", "active"),
		)

	// Execute query
	rows, err := db.Query("SELECT user_id, username, status FROM users")
	if err != nil {
		panic(err)
	}

	// Get columns
	cols, _ := rows.Columns()

	// Create scanner
	scanner := stream.GenericRowScanner()

	// Create fetcher using SQLFetcherWithColumns
	ctx := context.Background()
	fetcher := stream.SQLFetcherWithColumns(rows, cols, scanner)

	// Create streamer and transformer
	streamer := stream.NewDefaultStreamer[map[string]interface{}]()
	transformer := stream.PassThroughTransformer[map[string]interface{}]()

	// Stream
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	chunkCount := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error: %v\n", chunk.Error)
			return
		}
		if chunk.JSONBuf != nil {
			chunkCount++
		}
	}

	fmt.Printf("SQLFetcherWithColumns: streamed %d chunks\n", chunkCount)
	// Output: SQLFetcherWithColumns: streamed 1 chunks
}

// ============================================================================
// Example 5: SQLBatchFetcherWithColumns
// ============================================================================

// Example_sqlBatchFetcherWithColumns demonstrates using SQLBatchFetcherWithColumns
// for efficient batch streaming of SQL results.
func Example_sqlBatchFetcherWithColumns() {
	// Use map[string]interface{} directly for this example

	// Create mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup mock - 5 rows
	columns := []string{"id", "value"}
	rows := sqlmock.NewRows(columns)
	for i := 1; i <= 5; i++ {
		rows.AddRow(i, fmt.Sprintf("value_%d", i))
	}

	mock.ExpectQuery("SELECT (.+) FROM data").WillReturnRows(rows)

	// Execute query
	queryRows, err := db.Query("SELECT id, value FROM data")
	if err != nil {
		panic(err)
	}

	// Get columns
	cols, _ := queryRows.Columns()

	// Create scanner
	scanner := stream.GenericRowScanner()

	// Create batch fetcher with batch size 2
	ctx := context.Background()
	batchSize := 2
	batchFetcher := stream.SQLBatchFetcherWithColumns(queryRows, cols, batchSize, scanner)

	// Create streamer and transformer
	streamer := stream.NewDefaultStreamer[map[string]interface{}]()
	transformer := stream.PassThroughBatchTransformer[map[string]interface{}]()

	// Stream
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	batchCount := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error: %v\n", chunk.Error)
			return
		}
		if chunk.JSONBuf != nil {
			batchCount++
		}
	}

	fmt.Printf("SQLBatchFetcherWithColumns: processed batches into %d chunks\n", batchCount)
	// Output: SQLBatchFetcherWithColumns: processed batches into 1 chunks
}

// ============================================================================
// Example 6: TransformerAdapter
// ============================================================================

// Example_transformerAdapter demonstrates using TransformerAdapter to wrap
// domain transformation logic.
func Example_transformerAdapter() {
	type User struct {
		ID   int
		Name string
		Age  int
	}

	type UserDTO struct {
		UserID   int    `json:"user_id"`
		FullName string `json:"full_name"`
		IsAdult  bool   `json:"is_adult"`
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[User]()

	// Create fetcher
	fetcher := func(ctx context.Context) (<-chan User, <-chan error) {
		dataChan := make(chan User, 3)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			users := []User{
				{ID: 1, Name: "Alice", Age: 25},
				{ID: 2, Name: "Bob", Age: 17},
			}

			for _, u := range users {
				dataChan <- u
			}
		}()

		return dataChan, errChan
	}

	// Domain transformation logic
	domainTransform := func(user User) (interface{}, error) {
		return UserDTO{
			UserID:   user.ID,
			FullName: user.Name,
			IsAdult:  user.Age >= 18,
		}, nil
	}

	// Use TransformerAdapter to wrap domain logic
	transformer := stream.TransformerAdapter(domainTransform)
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("TransformerAdapter: transformation successful")
	}
	// Output: TransformerAdapter: transformation successful
}

// ============================================================================
// Example 7: BatchTransformerAdapter
// ============================================================================

// Example_batchTransformerAdapter demonstrates using BatchTransformerAdapter
// for efficient batch transformations.
func Example_batchTransformerAdapter() {
	type Temperature struct {
		City    string
		Celsius float64
	}

	type TemperatureDTO struct {
		City       string  `json:"city"`
		Celsius    float64 `json:"celsius"`
		Fahrenheit float64 `json:"fahrenheit"`
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[Temperature]()

	// Create batch fetcher
	batchFetcher := func(ctx context.Context) (<-chan []Temperature, <-chan error) {
		batchChan := make(chan []Temperature, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)

			batch := []Temperature{
				{City: "Tokyo", Celsius: 25.0},
				{City: "London", Celsius: 15.0},
				{City: "New York", Celsius: 20.0},
			}

			batchChan <- batch
		}()

		return batchChan, errChan
	}

	// Domain transformation - single item
	domainTransform := func(temp Temperature) (interface{}, error) {
		return TemperatureDTO{
			City:       temp.City,
			Celsius:    temp.Celsius,
			Fahrenheit: (temp.Celsius * 9 / 5) + 32,
		}, nil
	}

	// Use BatchTransformerAdapter
	transformer := stream.BatchTransformerAdapter(domainTransform)
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("BatchTransformerAdapter: batch transformation successful")
	}
	// Output: BatchTransformerAdapter: batch transformation successful
}

// ============================================================================
// Example 8: BatchTransformerWithContext
// ============================================================================

// Example_batchTransformerWithContext demonstrates context-aware batch transformation
// with cancellation support.
func Example_batchTransformerWithContext() {
	type Job struct {
		ID   int
		Name string
	}

	type JobResult struct {
		JobID  int    `json:"job_id"`
		Status string `json:"status"`
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	streamer := stream.NewDefaultStreamer[Job]()

	// Create batch fetcher
	batchFetcher := func(ctx context.Context) (<-chan []Job, <-chan error) {
		batchChan := make(chan []Job, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)

			batch := []Job{
				{ID: 1, Name: "Job A"},
				{ID: 2, Name: "Job B"},
			}

			batchChan <- batch
		}()

		return batchChan, errChan
	}

	// Domain transformation
	domainTransform := func(job Job) (interface{}, error) {
		// Simulate processing
		return JobResult{
			JobID:  job.ID,
			Status: "completed",
		}, nil
	}

	// Use BatchTransformerWithContext for cancellation support
	transformer := stream.BatchTransformerWithContext(ctx, domainTransform)
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("BatchTransformerWithContext: context-aware transformation successful")
	}
	// Output: BatchTransformerWithContext: context-aware transformation successful
}

// ============================================================================
// Example 9: TransformationChain
// ============================================================================

// Example_transformationChain demonstrates chaining multiple transformations.
func Example_transformationChain() {
	type RawData struct {
		Value int
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[RawData]()

	// Create fetcher
	fetcher := func(ctx context.Context) (<-chan RawData, <-chan error) {
		dataChan := make(chan RawData, 2)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			dataChan <- RawData{Value: 5}
			dataChan <- RawData{Value: 10}
		}()

		return dataChan, errChan
	}

	// Step 1: Validate
	validateStep := func(item interface{}) (interface{}, error) {
		data := item.(RawData)
		if data.Value <= 0 {
			return nil, errors.New("invalid value")
		}
		return data, nil
	}

	// Step 2: Transform to map
	transformStep := func(item interface{}) (interface{}, error) {
		data := item.(RawData)
		return map[string]interface{}{
			"value":   data.Value,
			"doubled": data.Value * 2,
		}, nil
	}

	// Step 3: Enrich
	enrichStep := func(item interface{}) (interface{}, error) {
		data := item.(map[string]interface{})
		data["timestamp"] = "2025-01-27"
		return data, nil
	}

	// Create transformation chain
	transformer := stream.TransformationChain[RawData](
		validateStep,
		transformStep,
		enrichStep,
	)

	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("TransformationChain: multi-step transformation successful")
	}
	// Output: TransformationChain: multi-step transformation successful
}

// ============================================================================
// Example 10: BatchTransformParallel
// ============================================================================

// Example_batchTransformParallel demonstrates parallel batch transformation
// for CPU-intensive operations.
func Example_batchTransformParallel() {
	type DataPoint struct {
		ID    int
		Value float64
	}

	type ProcessedData struct {
		ID        int     `json:"id"`
		Original  float64 `json:"original"`
		Processed float64 `json:"processed"`
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[DataPoint]()

	// Create batch fetcher
	batchFetcher := func(ctx context.Context) (<-chan []DataPoint, <-chan error) {
		batchChan := make(chan []DataPoint, 1)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)

			// Create large batch
			batch := make([]DataPoint, 10)
			for i := 0; i < 10; i++ {
				batch[i] = DataPoint{ID: i + 1, Value: float64(i * 10)}
			}

			batchChan <- batch
		}()

		return batchChan, errChan
	}

	// CPU-intensive transformation
	domainTransform := func(dp DataPoint) (interface{}, error) {
		// Simulate CPU-intensive work
		result := dp.Value
		for i := 0; i < 100; i++ {
			result = result * 1.001
		}

		return ProcessedData{
			ID:        dp.ID,
			Original:  dp.Value,
			Processed: result,
		}, nil
	}

	// Use parallel transformation with CPU cores
	workerCount := runtime.NumCPU()
	if workerCount > 2 {
		workerCount = 2 // Limit for testing
	}

	transformer := stream.BatchTransformParallel(ctx, workerCount, domainTransform)
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("BatchTransformParallel: parallel transformation successful")
	}
	// Output: BatchTransformParallel: parallel transformation successful
}

// ============================================================================
// Example 11: Custom SQLRowScanner Implementation
// ============================================================================

// Example_customSQLRowScanner demonstrates creating a custom row scanner
// for domain-specific types.
func Example_customSQLRowScanner() {
	type Customer struct {
		ID       int
		Name     string
		Email    string
		IsActive bool
	}

	// Create mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup mock
	columns := []string{"id", "name", "email", "is_active"}
	mock.ExpectQuery("SELECT (.+) FROM customers").
		WillReturnRows(
			sqlmock.NewRows(columns).
				AddRow(1, "Alice Corp", "alice@corp.com", true).
				AddRow(2, "Bob Inc", "bob@inc.com", false),
		)

	// Execute query
	rows, err := db.Query("SELECT id, name, email, is_active FROM customers")
	if err != nil {
		panic(err)
	}

	// Create custom scanner with proper type handling
	customScanner := func(rows *sql.Rows, columns []string) (Customer, error) {
		var customer Customer
		err := rows.Scan(&customer.ID, &customer.Name, &customer.Email, &customer.IsActive)
		if err != nil {
			return Customer{}, fmt.Errorf("scan error: %w", err)
		}
		return customer, nil
	}

	// Get columns
	cols, _ := rows.Columns()

	// Create fetcher
	ctx := context.Background()
	fetcher := stream.SQLFetcherWithColumns(rows, cols, customScanner)

	// Create streamer
	streamer := stream.NewDefaultStreamer[Customer]()
	transformer := stream.PassThroughTransformer[Customer]()

	// Stream
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("CustomSQLRowScanner: custom scanner successful")
	}
	// Output: CustomSQLRowScanner: custom scanner successful
}

// ============================================================================
// Example 12: Complete Real-World Integration
// ============================================================================

// Example_realWorldIntegration demonstrates a complete real-world scenario
// combining multiple helpers.
func Example_realWorldIntegration() {
	// Use map[string]interface{} directly
	type EnrichedData struct {
		ID        int                    `json:"id"`
		Data      map[string]interface{} `json:"data"`
		ProcessAt string                 `json:"processed_at"`
	}

	// Create mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup mock with 3 rows
	columns := []string{"id", "name", "value"}
	rows := sqlmock.NewRows(columns).
		AddRow(1, "Item A", 100).
		AddRow(2, "Item B", 200).
		AddRow(3, "Item C", 300)

	mock.ExpectQuery("SELECT (.+) FROM items").WillReturnRows(rows)

	// Execute query
	queryRows, err := db.Query("SELECT id, name, value FROM items")
	if err != nil {
		panic(err)
	}

	// Step 1: Create scanner using GenericRowScanner
	scanner := stream.GenericRowScanner()

	// Step 2: Get columns and create batch fetcher
	cols, _ := queryRows.Columns()
	ctx := context.Background()
	batchFetcher := stream.SQLBatchFetcherWithColumns(queryRows, cols, 2, scanner)

	// Step 3: Create domain transformation
	domainTransform := func(row map[string]interface{}) (interface{}, error) {
		id, ok := row["id"].(int64)
		if !ok {
			return nil, fmt.Errorf("invalid id type")
		}

		return EnrichedData{
			ID:        int(id),
			Data:      row,
			ProcessAt: time.Now().Format("2006-01-02"),
		}, nil
	}

	// Step 4: Use BatchTransformerAdapter
	transformer := stream.BatchTransformerAdapter(domainTransform)

	// Step 5: Stream
	streamer := stream.NewDefaultStreamer[map[string]interface{}]()
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	totalChunks := 0
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			fmt.Printf("Error: %v\n", chunk.Error)
			return
		}
		if chunk.JSONBuf != nil {
			totalChunks++
		}
	}

	fmt.Printf("RealWorldIntegration: processed %d chunks successfully\n", totalChunks)
	// Output: RealWorldIntegration: processed 1 chunks successfully
}

// ============================================================================
// Example 13: SliceFetcher
// ============================================================================

// Example_sliceFetcher demonstrates using SliceFetcher to stream
// an in-memory slice.
func Example_sliceFetcher() {
	type Book struct {
		ISBN   string
		Title  string
		Author string
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[Book]()

	// Prepare data
	books := []Book{
		{ISBN: "123", Title: "Go Programming", Author: "John Doe"},
		{ISBN: "456", Title: "Web Development", Author: "Jane Smith"},
		{ISBN: "789", Title: "Database Design", Author: "Bob Wilson"},
	}

	// Create fetcher from slice
	fetcher := stream.SliceFetcher(books)
	transformer := stream.PassThroughTransformer[Book]()

	// Stream
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("SliceFetcher: streamed slice successfully")
	}
	// Output: SliceFetcher: streamed slice successfully
}

// ============================================================================
// Example 14: SliceBatchFetcher
// ============================================================================

// Example_sliceBatchFetcher demonstrates using SliceBatchFetcher to stream
// a slice in batches.
func Example_sliceBatchFetcher() {
	type Number struct {
		Value int
	}

	ctx := context.Background()
	streamer := stream.NewDefaultStreamer[Number]()

	// Prepare data - 10 numbers
	numbers := make([]Number, 10)
	for i := 0; i < 10; i++ {
		numbers[i] = Number{Value: i + 1}
	}

	// Create batch fetcher with batch size of 3
	// This will produce: [1,2,3], [4,5,6], [7,8,9], [10]
	batchSize := 3
	batchFetcher := stream.SliceBatchFetcher(numbers, batchSize)
	transformer := stream.PassThroughBatchTransformer[Number]()

	// Stream
	streamResp := streamer.StreamBatch(ctx, batchFetcher, transformer)

	// Consume stream
	success := true
	for chunk := range streamResp.ChunkChan {
		if chunk.Error != nil {
			success = false
			fmt.Printf("Error: %v\n", chunk.Error)
		}
	}

	if success {
		fmt.Println("SliceBatchFetcher: streamed batches successfully")
	}
	// Output: SliceBatchFetcher: streamed batches successfully
}
