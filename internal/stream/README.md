# Stream Package

A general-purpose, high-performance streaming framework for Go applications. This package provides reusable abstractions for streaming large datasets with efficient memory management, buffer pooling, and chunked JSON encoding.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Usage Examples](#usage-examples)
- [Performance](#performance)
- [API Reference](#api-reference)
- [Best Practices](#best-practices)
- [Migration Guide](#migration-guide)

## Features

✅ **Generic & Type-Safe**: Fully generic implementation using Go 1.18+ generics
✅ **Memory Efficient**: Buffer pooling reduces GC pressure by ~51%
✅ **Zero Dependencies**: Only depends on stdlib and json-iterator
✅ **Context-Aware**: Respects context cancellation and timeouts
✅ **Configurable**: Customizable chunk sizes, batch sizes, and buffer sizes
✅ **Middleware Compatible**: Works seamlessly with existing `middleware.StreamResponse`
✅ **Production Ready**: Comprehensive test coverage and benchmarks

## Installation

```go
import "stream/internal/stream"
```

No external dependencies required beyond what's already in your project.

## Quick Start

### Basic Streaming

```go
package main

import (
    "context"
    "stream/internal/stream"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func StreamUsers(ctx context.Context) middleware.StreamResponse {
    // Create streamer with default config
    streamer := stream.NewDefaultStreamer[User]()

    // Define data fetcher
    fetcher := func(ctx context.Context) (<-chan User, <-chan error) {
        dataChan := make(chan User, 10)
        errChan := make(chan error, 1)

        go func() {
            defer close(dataChan)
            defer close(errChan)

            // Fetch users from database
            users := fetchUsersFromDB() // Your DB logic
            for _, user := range users {
                select {
                case dataChan <- user:
                case <-ctx.Done():
                    return
                }
            }
        }()

        return dataChan, errChan
    }

    // Define transformer (optional - pass-through in this case)
    transformer := stream.PassThroughTransformer[User]()

    // Stream!
    return streamer.Stream(ctx, fetcher, transformer)
}
```

### Using in Gin Handler

```go
func (h *Handler) StreamUsersHandler(c *gin.Context) {
    ctx := c.Request.Context()

    streamResp := StreamUsers(ctx)

    // Use existing middleware
    sendStream := c.MustGet("sendStream").(func(middleware.StreamResponse))
    sendStream(streamResp)
}
```

## Core Concepts

### 1. Streamer

The main interface for streaming data. Handles buffering, chunking, and JSON encoding.

```go
type Streamer[T any] interface {
    Stream(ctx context.Context, fetcher DataFetcher[T], transformer Transformer[T]) middleware.StreamResponse
    StreamBatch(ctx context.Context, fetcher BatchFetcher[T], transformer BatchTransformer[T]) middleware.StreamResponse
    GetConfig() ChunkConfig
}
```

### 2. DataFetcher

A function that fetches data items and sends them to a channel.

```go
type DataFetcher[T any] func(ctx context.Context) (<-chan T, <-chan error)
```

**Rules**:
- MUST close both channels when done
- SHOULD respect context cancellation
- Error channel should be buffered (capacity 1)

### 3. Transformer

A function that transforms a data item into JSON-encodable output.

```go
type Transformer[T any] func(item T) (interface{}, error)
```

**Rules**:
- Should be stateless and thread-safe
- Return value MUST be JSON-encodable
- Errors cause streaming to stop immediately

### 4. BufferPool

Manages pooled byte buffers to minimize allocations.

```go
type BufferPool interface {
    Get() *[]byte
    Put(buf *[]byte)
    GetInitialSize() int
}
```

**Benefits**:
- Reduces allocations by ~51%
- Minimizes GC pressure
- Reuses memory efficiently

## Usage Examples

### Example 1: Streaming Database Query Results

```go
import (
    "context"
    "database/sql"
    "stream/internal/stream"
)

type Ticket struct {
    ID          int    `json:"id"`
    Subject     string `json:"subject"`
    Status      string `json:"status"`
}

func StreamTickets(ctx context.Context, db *sql.DB, query string, args ...interface{}) middleware.StreamResponse {
    // Execute query
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return middleware.StreamResponse{
            Code:  500,
            Error: err,
        }
    }

    // Create streamer
    config := stream.DefaultChunkConfig()
    streamer := stream.NewStreamer[Ticket](config)

    // Define scanner for SQL rows
    scanner := func(rows *sql.Rows) (Ticket, error) {
        var ticket Ticket
        err := rows.Scan(&ticket.ID, &ticket.Subject, &ticket.Status)
        return ticket, err
    }

    // Create SQL fetcher
    fetcher := stream.SQLFetcher(rows, scanner)

    // Transform (add computed fields, mask sensitive data, etc.)
    transformer := func(ticket Ticket) (interface{}, error) {
        return map[string]interface{}{
            "id":      ticket.ID,
            "subject": ticket.Subject,
            "status":  ticket.Status,
            "masked_id": fmt.Sprintf("***%d", ticket.ID % 1000),
        }, nil
    }

    return streamer.Stream(ctx, fetcher, transformer)
}
```

### Example 2: Streaming with Batch Processing

```go
func StreamTicketsBatch(ctx context.Context, db *sql.DB) middleware.StreamResponse {
    rows, err := db.QueryContext(ctx, "SELECT * FROM tickets")
    if err != nil {
        return middleware.StreamResponse{Code: 500, Error: err}
    }

    config := stream.ChunkConfig{
        ChunkThreshold: 32 * 1024,  // 32KB chunks
        BatchSize:      1000,        // 1000 rows per batch
        BufferSize:     50 * 1024,   // 50KB buffer
    }
    streamer := stream.NewStreamer[Ticket](config)

    scanner := func(rows *sql.Rows) (Ticket, error) {
        var ticket Ticket
        err := rows.Scan(&ticket.ID, &ticket.Subject, &ticket.Status)
        return ticket, err
    }

    fetcher := stream.SQLBatchFetcher(rows, 1000, scanner)

    // Batch transformer - more efficient for expensive operations
    transformer := func(tickets []Ticket) ([]interface{}, error) {
        // Example: Batch lookup enrichment data
        enrichmentData := batchLookupEnrichment(tickets) // Your logic

        result := make([]interface{}, len(tickets))
        for i, ticket := range tickets {
            result[i] = map[string]interface{}{
                "id":         ticket.ID,
                "subject":    ticket.Subject,
                "status":     ticket.Status,
                "enrichment": enrichmentData[i],
            }
        }
        return result, nil
    }

    return streamer.StreamBatch(ctx, fetcher, transformer)
}
```

### Example 3: Streaming In-Memory Data

```go
func StreamReport(ctx context.Context, reportData []ReportItem) middleware.StreamResponse {
    streamer := stream.NewDefaultStreamer[ReportItem]()

    fetcher := stream.SliceFetcher(reportData)

    transformer := func(item ReportItem) (interface{}, error) {
        return map[string]interface{}{
            "date":   item.Date.Format("2006-01-02"),
            "value":  item.Value,
            "metric": item.Metric,
        }, nil
    }

    return streamer.Stream(ctx, fetcher, transformer)
}
```

### Example 4: Custom Data Source

```go
func StreamFromAPI(ctx context.Context, apiClient *APIClient) middleware.StreamResponse {
    streamer := stream.NewDefaultStreamer[APIResponse]()

    fetcher := func(ctx context.Context) (<-chan APIResponse, <-chan error) {
        dataChan := make(chan APIResponse, 10)
        errChan := make(chan error, 1)

        go func() {
            defer close(dataChan)
            defer close(errChan)

            offset := 0
            limit := 100

            for {
                // Fetch page from API
                resp, err := apiClient.FetchPage(ctx, offset, limit)
                if err != nil {
                    errChan <- err
                    return
                }

                // Send items
                for _, item := range resp.Items {
                    select {
                    case dataChan <- item:
                    case <-ctx.Done():
                        return
                    }
                }

                // Check if more pages
                if len(resp.Items) < limit {
                    break
                }

                offset += limit
            }
        }()

        return dataChan, errChan
    }

    transformer := stream.PassThroughTransformer[APIResponse]()

    return streamer.Stream(ctx, fetcher, transformer)
}
```

## Performance

### Benchmarks

Based on comprehensive benchmarks (see `BUFFER_POOL_ANALYSIS.md` in tickets package):

| Metric | Value |
|--------|-------|
| **Buffer Pool Overhead** | 8.37 ns/op |
| **Memory Savings** | ~51% vs fresh allocations |
| **Optimal Buffer Size** | 50KB (proven via benchmarks) |
| **Optimal Chunk Size** | 32KB (balances latency/throughput) |
| **Recommended Batch Size** | 1000 items |

### Performance Tips

1. **Use Batch Processing** when transformation has setup cost
2. **Configure Buffer Size** based on your data size
3. **Adjust Chunk Threshold** for network conditions
4. **Reuse Streamer** instances when possible
5. **Profile** your specific use case

### Memory Profile

```
Without Pool:
- 100 requests × 111KB = 11.1 MB
- High GC pressure
- Frequent allocations

With Pool (50KB buffer):
- 100 requests × 54KB = 5.4 MB
- Low GC pressure
- Buffer reuse

Savings: 51% memory reduction
```

## API Reference

### Configuration

#### DefaultChunkConfig()

Returns default configuration optimized from benchmarks.

```go
config := stream.DefaultChunkConfig()
// ChunkThreshold: 32KB
// BatchSize: 1000
// BufferSize: 50KB
// ChannelBuffer: 4
```

#### Custom Configuration

```go
config := stream.ChunkConfig{
    ChunkThreshold: 64 * 1024,   // 64KB chunks
    BatchSize:      500,          // 500 items per batch
    BufferSize:     100 * 1024,   // 100KB buffer
    ChannelBuffer:  8,            // 8-buffer channels
}

err := config.Validate() // Applies defaults for zero values
```

### Streamer Creation

```go
// Default configuration
streamer := stream.NewDefaultStreamer[MyType]()

// Custom configuration
config := stream.ChunkConfig{...}
streamer := stream.NewStreamer[MyType](config)
```

### Helper Functions

#### SQL Helpers

```go
// Item-by-item SQL streaming
fetcher := stream.SQLFetcher(rows, scanner)

// Batch SQL streaming
fetcher := stream.SQLBatchFetcher(rows, batchSize, scanner)
```

#### Slice Helpers

```go
// Stream from slice
fetcher := stream.SliceFetcher(items)

// Stream batches from slice
fetcher := stream.SliceBatchFetcher(items, batchSize)
```

#### Transform Helpers

```go
// Pass-through transformer
transformer := stream.PassThroughTransformer[MyType]()

// Pass-through batch transformer
transformer := stream.PassThroughBatchTransformer[MyType]()
```

#### Buffer Pool Helpers

```go
// Global pool
buf := stream.GetBuffer()
defer stream.PutBuffer(buf)

// Custom pool
pool := stream.NewBufferPool(50 * 1024)
buf := pool.Get()
defer pool.Put(buf)
```

## Best Practices

### 1. Always Close Channels

```go
fetcher := func(ctx context.Context) (<-chan T, <-chan error) {
    dataChan := make(chan T, 10)
    errChan := make(chan error, 1)

    go func() {
        defer close(dataChan) // ✅ Always close
        defer close(errChan)  // ✅ Always close

        // Your logic
    }()

    return dataChan, errChan
}
```

### 2. Respect Context Cancellation

```go
for item := range source {
    select {
    case dataChan <- item:
    case <-ctx.Done(): // ✅ Check context
        return
    }
}
```

### 3. Buffer Error Channels

```go
errChan := make(chan error, 1) // ✅ Buffered to prevent goroutine leak
```

### 4. Keep Transformers Stateless

```go
// ❌ Bad: Stateful transformer
var counter int
transformer := func(item T) (interface{}, error) {
    counter++ // NOT thread-safe!
    return counter, nil
}

// ✅ Good: Stateless transformer
transformer := func(item T) (interface{}, error) {
    return processItem(item), nil
}
```

### 5. Choose Appropriate Config

```go
// Small responses (< 100 items)
config := stream.ChunkConfig{
    ChunkThreshold: 16 * 1024,  // 16KB
    BatchSize:      100,
    BufferSize:     32 * 1024,  // 32KB
}

// Large responses (1000+ items)
config := stream.DefaultChunkConfig() // 32KB/1000/50KB

// Very large responses (10000+ items)
config := stream.ChunkConfig{
    ChunkThreshold: 64 * 1024,   // 64KB
    BatchSize:      5000,
    BufferSize:     100 * 1024,  // 100KB
}
```

### 6. Handle Errors Properly

```go
streamResp := streamer.Stream(ctx, fetcher, transformer)

// Errors can come from:
// 1. streamResp.Error (pre-streaming error)
// 2. StreamChunk.Error (during streaming)

if streamResp.Error != nil {
    // Handle pre-streaming error
    return middleware.StreamResponse{
        Code: 500,
        Error: streamResp.Error,
    }
}

// Middleware handles chunk errors automatically
```

## Migration Guide

### From Tickets Service to Generic Stream

**Before** (tickets/service.go):

```go
func (s *Service) StreamTickets(ctx context.Context, payload *QueryPayload) middleware.StreamResponse {
    // ... query building ...

    rows, err := s.repo.ExecuteQuery(ctx, mainQuery, mainArgs)
    if err != nil {
        return middleware.StreamResponse{Code: 500, Error: err}
    }

    // ... manual streaming logic ...
    chunkChan := s.streamProcessing(ctx, rows, formulas, batchSize, isFormatDate)

    return middleware.StreamResponse{
        TotalCount: totalCount,
        ChunkChan:  chunkChan,
        Code:       200,
    }
}
```

**After** (using stream package):

```go
import "stream/internal/stream"

func (s *Service) StreamTickets(ctx context.Context, payload *QueryPayload) middleware.StreamResponse {
    // ... query building ...

    rows, err := s.repo.ExecuteQuery(ctx, mainQuery, mainArgs)
    if err != nil {
        return middleware.StreamResponse{Code: 500, Error: err}
    }

    // Create streamer
    streamer := stream.NewDefaultStreamer[RowData]()

    // Use SQL fetcher
    scanner := func(rows *sql.Rows) (RowData, error) {
        return ScanRowGeneric(rows, columns)
    }
    fetcher := stream.SQLFetcher(rows, scanner)

    // Define transformer
    transformer := func(row RowData) (interface{}, error) {
        transformed, err := TransformRow(row, formulas, s.operators)
        if err != nil {
            return nil, err
        }

        if isFormatDate {
            transformed = formatDateFields(transformed)
        }

        return transformed, nil
    }

    // Stream with totalCount
    streamResp := streamer.Stream(ctx, fetcher, transformer)
    streamResp.TotalCount = totalCount // Add count from earlier query

    return streamResp
}
```

**Benefits**:
- ✅ Reusable streaming logic
- ✅ Tested buffer pooling
- ✅ Simplified code
- ✅ Better performance monitoring
- ✅ Easier to maintain

## Testing

Run tests:

```bash
go test ./internal/stream/
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./internal/stream/
```

## Contributing

When adding new features:

1. Add comprehensive tests
2. Update this README
3. Run benchmarks to ensure no performance regression
4. Follow existing code patterns

## License

Internal package - same license as parent project.
