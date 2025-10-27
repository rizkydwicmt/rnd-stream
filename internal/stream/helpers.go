package stream

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLRowScanner is a function that scans a SQL row into a custom type.
// It's used with SQLFetcher to abstract database row scanning.
//
// Parameters:
//   - rows: The SQL rows being scanned
//
// Returns:
//   - T: Scanned data item
//   - error: Error if scanning fails
//
// Example:
//
//	scanner := func(rows *sql.Rows) (MyStruct, error) {
//	    var item MyStruct
//	    err := rows.Scan(&item.Field1, &item.Field2)
//	    return item, err
//	}
type SQLRowScanner[T any] func(rows *sql.Rows) (T, error)

// SQLFetcher creates a DataFetcher from SQL rows using a custom scanner.
// This is a common pattern for streaming database query results.
//
// Parameters:
//   - rows: SQL rows from query execution
//   - scanner: Function to scan each row into type T
//
// Returns:
//   - DataFetcher[T]: Fetcher that streams SQL rows
//
// Usage:
//
//	rows, err := db.QueryContext(ctx, query, args...)
//	if err != nil {
//	    return err
//	}
//
//	scanner := func(rows *sql.Rows) (MyStruct, error) {
//	    var item MyStruct
//	    err := rows.Scan(&item.Field1, &item.Field2)
//	    return item, err
//	}
//
//	fetcher := stream.SQLFetcher(rows, scanner)
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
//
// Implementation Notes:
//   - Closes rows automatically when done
//   - Respects context cancellation
//   - Buffers up to 10 items in channel
//   - Sends error on scan failure
func SQLFetcher[T any](rows *sql.Rows, scanner SQLRowScanner[T]) DataFetcher[T] {
	return func(ctx context.Context) (<-chan T, <-chan error) {
		dataChan := make(chan T, 10)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)
			defer rows.Close()

			for rows.Next() {
				// Check context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Scan row
				item, err := scanner(rows)
				if err != nil {
					errChan <- fmt.Errorf("failed to scan row: %w", err)
					return
				}

				// Send item
				select {
				case dataChan <- item:
				case <-ctx.Done():
					return
				}
			}

			// Check for iteration errors
			if err := rows.Err(); err != nil {
				errChan <- fmt.Errorf("error iterating rows: %w", err)
			}
		}()

		return dataChan, errChan
	}
}

// SQLBatchScanner is a function that scans SQL rows into a batch of items.
// It continues scanning until either batchSize is reached or no more rows.
//
// Parameters:
//   - rows: The SQL rows being scanned
//   - batchSize: Maximum number of items to scan
//   - scanner: Function to scan each row
//
// Returns:
//   - []T: Batch of scanned items
//   - error: Error if scanning fails
//
// Example:
//
//	batchScanner := func(rows *sql.Rows, size int, scanner SQLRowScanner[MyStruct]) ([]MyStruct, error) {
//	    return stream.ScanBatch(rows, size, scanner)
//	}
type SQLBatchScanner[T any] func(rows *sql.Rows, batchSize int, scanner SQLRowScanner[T]) ([]T, error)

// ScanBatch is a helper function to scan a batch of SQL rows.
// It's used internally by SQLBatchFetcher.
//
// Parameters:
//   - rows: SQL rows to scan
//   - batchSize: Maximum number of rows to scan
//   - scanner: Function to scan each row
//
// Returns:
//   - []T: Batch of scanned items (may be less than batchSize at end)
//   - error: Error if scanning fails
func ScanBatch[T any](rows *sql.Rows, batchSize int, scanner SQLRowScanner[T]) ([]T, error) {
	batch := make([]T, 0, batchSize)

	for i := 0; i < batchSize && rows.Next(); i++ {
		item, err := scanner(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		batch = append(batch, item)
	}

	return batch, nil
}

// SQLBatchFetcher creates a BatchFetcher from SQL rows using a custom scanner.
// This is more efficient than SQLFetcher when processing can benefit from batching.
//
// Parameters:
//   - rows: SQL rows from query execution
//   - batchSize: Number of rows per batch
//   - scanner: Function to scan each row into type T
//
// Returns:
//   - BatchFetcher[T]: Fetcher that streams batches of SQL rows
//
// Usage:
//
//	rows, err := db.QueryContext(ctx, query, args...)
//	if err != nil {
//	    return err
//	}
//
//	scanner := func(rows *sql.Rows) (MyStruct, error) {
//	    var item MyStruct
//	    err := rows.Scan(&item.Field1, &item.Field2)
//	    return item, err
//	}
//
//	fetcher := stream.SQLBatchFetcher(rows, 1000, scanner)
//	streamResp := streamer.StreamBatch(ctx, fetcher, batchTransformer)
//
// Performance:
//   - More efficient than item-by-item when batch transformation is possible
//   - Reduces channel communication overhead
//   - Better CPU cache locality
func SQLBatchFetcher[T any](rows *sql.Rows, batchSize int, scanner SQLRowScanner[T]) BatchFetcher[T] {
	return func(ctx context.Context) (<-chan []T, <-chan error) {
		batchChan := make(chan []T, 2)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)
			defer rows.Close()

			for rows.Next() {
				// Check context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Scan batch
				batch, err := ScanBatch(rows, batchSize, scanner)
				if err != nil {
					errChan <- err
					return
				}

				if len(batch) > 0 {
					// Send batch
					select {
					case batchChan <- batch:
					case <-ctx.Done():
						return
					}
				}
			}

			// Check for iteration errors
			if err := rows.Err(); err != nil {
				errChan <- fmt.Errorf("error iterating rows: %w", err)
			}
		}()

		return batchChan, errChan
	}
}

// SliceFetcher creates a DataFetcher from a slice.
// Useful for testing or when data is already in memory.
//
// Parameters:
//   - items: Slice of items to stream
//
// Returns:
//   - DataFetcher[T]: Fetcher that streams slice items
//
// Usage:
//
//	items := []MyStruct{{Field: "value1"}, {Field: "value2"}}
//	fetcher := stream.SliceFetcher(items)
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
//
// Use Cases:
//   - Testing
//   - Streaming in-memory data
//   - Converting existing slice-based code to streaming
func SliceFetcher[T any](items []T) DataFetcher[T] {
	return func(ctx context.Context) (<-chan T, <-chan error) {
		dataChan := make(chan T, 10)
		errChan := make(chan error, 1)

		go func() {
			defer close(dataChan)
			defer close(errChan)

			for _, item := range items {
				select {
				case dataChan <- item:
				case <-ctx.Done():
					return
				}
			}
		}()

		return dataChan, errChan
	}
}

// SliceBatchFetcher creates a BatchFetcher from a slice.
// Splits the slice into batches of the specified size.
//
// Parameters:
//   - items: Slice of items to stream
//   - batchSize: Size of each batch
//
// Returns:
//   - BatchFetcher[T]: Fetcher that streams slice batches
//
// Usage:
//
//	items := []MyStruct{ /* ... */ }
//	fetcher := stream.SliceBatchFetcher(items, 100)
//	streamResp := streamer.StreamBatch(ctx, fetcher, batchTransformer)
func SliceBatchFetcher[T any](items []T, batchSize int) BatchFetcher[T] {
	return func(ctx context.Context) (<-chan []T, <-chan error) {
		batchChan := make(chan []T, 2)
		errChan := make(chan error, 1)

		go func() {
			defer close(batchChan)
			defer close(errChan)

			for i := 0; i < len(items); i += batchSize {
				select {
				case <-ctx.Done():
					return
				default:
				}

				end := i + batchSize
				if end > len(items) {
					end = len(items)
				}

				batch := items[i:end]

				select {
				case batchChan <- batch:
				case <-ctx.Done():
					return
				}
			}
		}()

		return batchChan, errChan
	}
}

// PassThroughTransformer creates a Transformer that returns items unchanged.
// Useful when data is already in the desired format.
//
// Returns:
//   - Transformer[T]: Transformer that passes through items
//
// Usage:
//
//	transformer := stream.PassThroughTransformer[MyStruct]()
//	streamResp := streamer.Stream(ctx, fetcher, transformer)
func PassThroughTransformer[T any]() Transformer[T] {
	return func(item T) (interface{}, error) {
		return item, nil
	}
}

// PassThroughBatchTransformer creates a BatchTransformer that returns items unchanged.
//
// Returns:
//   - BatchTransformer[T]: Transformer that passes through batches
//
// Usage:
//
//	transformer := stream.PassThroughBatchTransformer[MyStruct]()
//	streamResp := streamer.StreamBatch(ctx, fetcher, transformer)
func PassThroughBatchTransformer[T any]() BatchTransformer[T] {
	return func(items []T) ([]interface{}, error) {
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = item
		}
		return result, nil
	}
}
