# Stream Package Architecture

## Overview

The `internal/stream` package provides a general-purpose streaming framework extracted from the `tickets` service. It abstracts the complexity of streaming large datasets while maintaining high performance and low memory usage.

## Design Principles

### 1. **Separation of Concerns**

The package separates streaming concerns into distinct components:

```
┌─────────────────────────────────────────────────┐
│                 Application                      │
│   (Defines fetcher, transformer, handles        │
│    business logic)                               │
└────────────────┬────────────────────────────────┘
                 │
                 │ Uses
                 ↓
┌─────────────────────────────────────────────────┐
│            Stream Package                        │
│                                                  │
│  ┌─────────────┐  ┌──────────────┐             │
│  │  Streamer   │→ │ BufferPool   │             │
│  └─────────────┘  └──────────────┘             │
│         │                                        │
│         ↓                                        │
│  ┌─────────────┐  ┌──────────────┐             │
│  │  Fetcher    │  │ Transformer  │             │
│  └─────────────┘  └──────────────┘             │
│                                                  │
└────────────────┬────────────────────────────────┘
                 │
                 │ Returns
                 ↓
┌─────────────────────────────────────────────────┐
│              Middleware                          │
│   (Handles HTTP response, writes chunks)         │
└─────────────────────────────────────────────────┘
```

### 2. **Generic Programming**

Uses Go 1.18+ generics for type-safe, reusable code:

```go
type Streamer[T any] interface {
    Stream(ctx context.Context,
           fetcher DataFetcher[T],
           transformer Transformer[T]) middleware.StreamResponse
}
```

**Benefits**:
- Type safety at compile time
- No interface{} casting
- Better IDE support
- Clearer API

### 3. **Functional Composition**

Uses function types for flexibility:

```go
type DataFetcher[T any] func(ctx context.Context) (<-chan T, <-chan error)
type Transformer[T any] func(item T) (interface{}, error)
```

**Benefits**:
- Easy to compose
- No need for wrapper structs
- Testable in isolation
- Flexible implementation

### 4. **Resource Management**

Explicit resource management with clear ownership:

```go
// Buffer Pool
buf := pool.Get()
defer pool.Put(buf) // Clear ownership

// Channels
defer close(dataChan) // Explicit cleanup
defer close(errChan)
```

## Component Architecture

### Streamer

**Responsibilities**:
- Orchestrate streaming pipeline
- Manage buffer pool
- Handle chunk threshold
- Coordinate goroutines
- Propagate errors

**Implementation**:
```go
type streamer[T any] struct {
    config     ChunkConfig  // Configuration
    bufferPool BufferPool   // Buffer management
}
```

**Key Methods**:
- `Stream()`: Item-by-item streaming
- `StreamBatch()`: Batch streaming
- `GetConfig()`: Configuration introspection

### BufferPool

**Responsibilities**:
- Manage byte buffer lifecycle
- Minimize allocations
- Reduce GC pressure

**Implementation**:
```go
type bufferPool struct {
    pool        *sync.Pool  // Thread-safe pool
    initialSize int         // Buffer capacity
}
```

**Performance**:
- Get/Put: ~8ns overhead
- 51% memory savings vs fresh allocations
- Zero-allocation reuse

### DataFetcher

**Responsibilities**:
- Fetch data from source
- Send items to channel
- Handle errors
- Respect context cancellation

**Contract**:
```go
// MUST:
// - Close both channels when done
// - Respect ctx.Done()
// - Send at most one error

// MUST NOT:
// - Block indefinitely
// - Send after channel close
// - Leak goroutines
```

### Transformer

**Responsibilities**:
- Transform data item
- Validate output
- Handle business logic

**Contract**:
```go
// MUST:
// - Be stateless
// - Be thread-safe
// - Return JSON-encodable output

// MUST NOT:
// - Modify input
// - Maintain state
// - Block indefinitely
```

## Data Flow

### Streaming Pipeline

```
1. Application creates Streamer
   │
   ↓
2. Defines DataFetcher
   │
   ↓
3. Defines Transformer
   │
   ↓
4. Calls streamer.Stream()
   │
   ↓
5. Streamer starts goroutine
   │
   ├─→ Gets buffer from pool
   │   │
   │   ↓
   │   Writes JSON array start '['
   │   │
   │   ↓
   │   Fetches data from DataFetcher ◄───┐
   │   │                                  │
   │   ↓                                  │
   │   Transforms each item               │
   │   │                                  │
   │   ↓                                  │
   │   Encodes to JSON                    │
   │   │                                  │
   │   ↓                                  │
   │   Appends to buffer                  │
   │   │                                  │
   │   ↓                                  │
   │   Buffer > threshold? ──Yes─→ Send chunk
   │   │                          Get new buffer
   │   │                          └───────────┘
   │   No
   │   │
   │   └───────────────────────────────────┘
   │
   ↓
   Writes JSON array end ']'
   │
   ↓
   Sends final chunk
   │
   ↓
   Returns buffer to pool
   │
   ↓
   Closes chunk channel
```

### Error Flow

```
Fetcher Error
   │
   ↓
Send to errChan
   │
   ↓
Streamer receives error
   │
   ↓
Send StreamChunk{Error: ...}
   │
   ↓
Stop processing
   │
   ↓
Cleanup resources
   │
   ↓
Close channels
```

## Memory Management

### Buffer Pooling Strategy

**Problem**: Fresh allocations create GC pressure

**Solution**: Reuse buffers via `sync.Pool`

**Flow**:
```
Request 1:
  Get(new 50KB) → Use → Put(50KB)
                          ↓
Request 2:               Pool
  Get(reuse 50KB) ← ─────┘
     ↓
  Use → Put(50KB)
          ↓
Request 3:
  Get(reuse 50KB) ← ─────┘
```

**Benefits**:
- 51% less memory
- Fewer GC cycles
- Better throughput

### Stack vs Heap Allocation

**Analysis** (from escape analysis):

```bash
$ go build -gcflags="-m" internal/stream/*.go

# Heap allocations (necessary):
buffer_pool.go: make([]byte, 0, initialSize) escapes to heap
  → Required for sync.Pool

# Stack allocations (optimized):
streamer.go: config ChunkConfig does not escape
  → Lightweight configuration
```

**Design Decision**:
- Accept heap allocation for buffers (required for pooling)
- Keep config on stack (passed by value)
- Minimize interface{} usage (forces heap)

## Concurrency Model

### Goroutine Management

**Pattern**: Single goroutine per stream

```go
chunkChan := make(chan StreamChunk, bufferSize)

go func() {
    defer close(chunkChan)  // Always close
    defer cleanup()          // Always cleanup

    // Process data
    for item := range dataChan {
        // Transform and send
    }
}()

return chunkChan
```

**Safety**:
- One owner per goroutine
- Clear channel ownership
- No shared mutable state
- Context-based cancellation

### Channel Design

**Chunk Channel**:
```go
chunkChan := make(chan middleware.StreamChunk, 4)
```
- Buffered (4) to prevent blocking
- Owned by streaming goroutine
- Closed when done

**Data Channel** (from fetcher):
```go
dataChan := make(chan T, 10)
```
- Buffered to decouple producer/consumer
- Owned by fetcher goroutine
- Closed by fetcher

**Error Channel**:
```go
errChan := make(chan error, 1)
```
- Buffered (1) to prevent goroutine leak
- Send-once pattern
- Owned by fetcher goroutine

## Configuration Design

### ChunkConfig Structure

```go
type ChunkConfig struct {
    ChunkThreshold int  // When to send chunk
    BatchSize      int  // Items per batch (if batch streaming)
    BufferSize     int  // Initial buffer capacity
    ChannelBuffer  int  // Channel buffer size
}
```

**Design Decisions**:

1. **Mutable Config**:
   - Values can be changed after creation
   - `Validate()` applies defaults
   - Allows progressive configuration

2. **Value Semantics**:
   - Passed by value (not pointer)
   - Prevents unexpected mutations
   - Clear ownership

3. **Smart Defaults**:
   - Based on benchmarks
   - Optimized for common cases
   - Can be overridden

### Configuration Validation

```go
func (c *ChunkConfig) Validate() error {
    if c.ChunkThreshold <= 0 {
        c.ChunkThreshold = 32 * 1024  // Apply default
    }
    // ... more validations
    return nil
}
```

**Strategy**:
- Fail-safe (apply defaults, don't error)
- Progressive (validate as late as possible)
- Explicit (clear error messages if needed)

## Extension Points

### 1. Custom Buffer Pools

```go
type BufferPool interface {
    Get() *[]byte
    Put(buf *[]byte)
    GetInitialSize() int
}

// Implement custom pool
type FixedSizePool struct { ... }
```

### 2. Custom Encoders

```go
// Currently hardcoded to JSON
// Future: Support custom encoders

type Encoder interface {
    Encode(v interface{}) ([]byte, error)
}
```

### 3. Custom Metrics

```go
// Hook into streaming events
type StreamMetrics interface {
    OnChunkSent(size int)
    OnError(err error)
    OnComplete(duration time.Duration)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| Stream N items | O(N) | Linear with data size |
| Buffer Get/Put | O(1) | Amortized constant |
| JSON Encode | O(M) | M = item size |
| Channel Send | O(1) | Buffered channels |

### Space Complexity

| Component | Space | Notes |
|-----------|-------|-------|
| Buffer Pool | O(C × B) | C=concurrency, B=buffer size |
| Channel Buffer | O(K) | K=channel buffer size |
| Streaming | O(B) | B=batch size (if batching) |

### Benchmarks

From `streamer_test.go`:

```
BenchmarkStreamer_Stream-8
  100 items:  ~1000 μs
  Memory:     ~54KB per request (with pool)
  Allocs:     Minimal (reused buffers)
```

## Design Patterns Used

### 1. Strategy Pattern

```go
// Different fetching strategies
fetcher := SQLFetcher(...)        // Database strategy
fetcher := SliceFetcher(...)      // In-memory strategy
fetcher := CustomFetcher(...)     // Custom strategy
```

### 2. Template Method Pattern

```go
// Streaming template in Streamer
func (s *streamer[T]) Stream(...) {
    // 1. Setup
    // 2. Fetch
    // 3. Transform
    // 4. Encode
    // 5. Buffer
    // 6. Send chunks
    // 7. Cleanup
}
```

### 3. Object Pool Pattern

```go
// Buffer pooling
buf := pool.Get()
defer pool.Put(buf)
```

### 4. Pipeline Pattern

```go
Fetch → Transform → Encode → Buffer → Send
```

## Testing Strategy

### Unit Tests

- **Component Isolation**: Test each component separately
- **Mock Dependencies**: Use test doubles for external dependencies
- **Edge Cases**: Empty data, errors, cancellation
- **Concurrency**: Goroutine safety, race detection

### Integration Tests

- **End-to-End**: Full pipeline tests
- **Real Dependencies**: Test with actual database (when feasible)
- **Performance**: Benchmark real-world scenarios

### Property-Based Tests

```go
// Future improvement
func TestStreamer_Properties(t *testing.T) {
    // Property: All items sent are received
    // Property: No items are duplicated
    // Property: Order is preserved
}
```

## Future Enhancements

### 1. Metrics & Observability

```go
type MetricsCollector interface {
    RecordChunkSent(size int)
    RecordLatency(duration time.Duration)
    RecordError(err error)
}
```

### 2. Compression

```go
type CompressedStreamer[T any] struct {
    streamer Streamer[T]
    compressor Compressor  // gzip, zstd, etc.
}
```

### 3. Custom Encoders

```go
type Encoder interface {
    Encode(v interface{}) ([]byte, error)
    ContentType() string
}

// Support JSON, MessagePack, Protocol Buffers, etc.
```

### 4. Rate Limiting

```go
config := ChunkConfig{
    RateLimit: 1000,  // items/second
}
```

### 5. Backpressure Control

```go
type BackpressureStrategy interface {
    ShouldBlock() bool
    OnChunkSent()
}
```

## Conclusion

The stream package provides a solid foundation for streaming in Go applications. It balances:

- **Performance**: Optimized buffer pooling and minimal allocations
- **Simplicity**: Clear API with sensible defaults
- **Flexibility**: Generic types and functional composition
- **Safety**: Resource cleanup and error propagation
- **Maintainability**: Well-documented and tested

The architecture is designed to be:
- **Reusable**: Works for any data type
- **Extensible**: Easy to add new features
- **Testable**: Clear component boundaries
- **Production-Ready**: Proven in tickets service

For detailed usage examples, see `README.md`.
For performance analysis, see `BUFFER_POOL_ANALYSIS.md` in tickets package.
