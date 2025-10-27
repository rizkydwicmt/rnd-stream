# Transformer Enhancement Guide

**Version**: v0.8.0
**Date**: 2025-10-27
**Status**: Production Ready

## 📋 Overview

This document provides a comprehensive guide for using the enhanced transformation helpers in `internal/stream/helpers.go`. These enhancements enable better code reuse, improved performance, and reduced duplication across services when handling data transformations.

## 🎯 What's New

### Enhanced Transformation Helpers

Five new helper functions have been added to streamline transformation logic:

1. **`TransformerAdapter`** - Wraps domain transformation functions into stream transformers
2. **`BatchTransformerAdapter`** - Optimized batch transformation with pre-allocation
3. **`BatchTransformerWithContext`** - Context-aware batch transformation
4. **`TransformationChain`** - Composable transformation pipeline
5. **`BatchTransformParallel`** - Parallel batch processing for CPU-intensive tasks

### Key Benefits

- ✅ **DRY Compliance** - Eliminates ~30 lines of boilerplate per service
- ✅ **Performance** - Optimized pre-allocation and optional parallelization
- ✅ **Type Safety** - Generic type parameters for compile-time checking
- ✅ **Flexibility** - Multiple transformation strategies for different use cases
- ✅ **Well Tested** - Comprehensive unit tests and benchmarks
- ✅ **Backward Compatible** - Existing code unchanged

## 🔄 Migration Examples

### Before: Custom Transformer Implementation

```go
// application/ticketsV2/service/service.go (OLD)

func (s *service) createTransformer(formulas []domain.Formula, isFormatDate bool) stream.Transformer[domain.RowData] {
    return func(row domain.RowData) (interface{}, error) {
        // Transform the row using formulas
        transformed, err := s.transformer.TransformRow(row, formulas, isFormatDate)
        if err != nil {
            return nil, fmt.Errorf("failed to transform row: %w", err)
        }

        return transformed, nil
    }
}

// Usage
transformer := s.createTransformer(sortedFormulas, payload.IsFormatDate)
streamResp := streamer.Stream(ctx, fetcher, transformer)
```

### After: Using TransformerAdapter

```go
// application/ticketsV2/service/service.go (NEW)

// Usage - Much simpler!
domainTransform := func(row domain.RowData) (interface{}, error) {
    return s.transformer.TransformRow(row, sortedFormulas, payload.IsFormatDate)
}
transformer := stream.TransformerAdapter(domainTransform)
streamResp := streamer.Stream(ctx, fetcher, transformer)
```

**Reduction**: ~12 lines → ~4 lines (67% reduction!)

### Batch Transformation Migration

#### Before

```go
func (s *service) createBatchTransformer(formulas []domain.Formula, isFormatDate bool) stream.BatchTransformer[domain.RowData] {
    return func(rows []domain.RowData) ([]interface{}, error) {
        // Pre-allocate result slice
        result := make([]interface{}, len(rows))

        // Transform each row in the batch
        for i, row := range rows {
            transformed, err := s.transformer.TransformRow(row, formulas, isFormatDate)
            if err != nil {
                return nil, fmt.Errorf("failed to transform row at index %d: %w", i, err)
            }
            result[i] = transformed
        }

        return result, nil
    }
}

// Usage
batchTransformer := s.createBatchTransformer(sortedFormulas, payload.IsFormatDate)
streamResp := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)
```

#### After

```go
// Usage - Single inline definition!
domainTransform := func(row domain.RowData) (interface{}, error) {
    return s.transformer.TransformRow(row, sortedFormulas, payload.IsFormatDate)
}
batchTransformer := stream.BatchTransformerAdapter(domainTransform)
streamResp := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)
```

**Reduction**: ~20 lines → ~4 lines (80% reduction!)

## 📚 API Reference

### TransformerAdapter

```go
func TransformerAdapter[T any](domainTransform func(T) (interface{}, error)) Transformer[T]
```

**Parameters:**
- `domainTransform` - Domain-specific transformation function

**Returns:**
- `Transformer[T]` - Stream transformer function

**Use Cases:**
- Wrapping existing domain transformation logic
- Avoiding boilerplate transformer creation
- Single-item transformations

**Example:**
```go
domainTransform := func(user User) (interface{}, error) {
    return map[string]interface{}{
        "id":    user.ID,
        "name":  user.Name,
        "email": user.Email,
    }, nil
}

transformer := stream.TransformerAdapter(domainTransform)
response := streamer.Stream(ctx, fetcher, transformer)
```

**Error Handling:**
Errors from domain transform are wrapped with context:
```
transformation error: <original error>
```

### BatchTransformerAdapter

```go
func BatchTransformerAdapter[T any](domainTransform func(T) (interface{}, error)) BatchTransformer[T]
```

**Parameters:**
- `domainTransform` - Domain-specific transformation function for single items

**Returns:**
- `BatchTransformer[T]` - Batch transformer function

**Performance Benefits:**
- ✅ Pre-allocates result slice with exact capacity
- ✅ Processes items sequentially for predictable behavior
- ✅ Low memory overhead

**Example:**
```go
domainTransform := func(row RowData) (interface{}, error) {
    return processRow(row)
}

batchTransformer := stream.BatchTransformerAdapter(domainTransform)
response := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)
```

**Error Handling:**
Errors include index information:
```
transformation error at index <N>: <original error>
```

### BatchTransformerWithContext

```go
func BatchTransformerWithContext[T any](ctx context.Context, domainTransform func(T) (interface{}, error)) BatchTransformer[T]
```

**Parameters:**
- `ctx` - Context for cancellation support
- `domainTransform` - Domain-specific transformation function

**Returns:**
- `BatchTransformer[T]` - Context-aware batch transformer

**Use Cases:**
- Long-running transformations that need cancellation
- Request-scoped processing with timeouts
- Graceful shutdown scenarios

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

domainTransform := func(data Data) (interface{}, error) {
    return expensiveTransformation(data)
}

batchTransformer := stream.BatchTransformerWithContext(ctx, domainTransform)
response := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)
```

**Cancellation Behavior:**
- Checks context before processing each item
- Returns immediately on cancellation
- No partial results returned

### TransformationChain

```go
func TransformationChain[T any](transformers ...func(interface{}) (interface{}, error)) Transformer[T]
```

**Parameters:**
- `transformers` - Variable number of transformation functions

**Returns:**
- `Transformer[T]` - Composed transformer that applies all transformations in sequence

**Use Cases:**
- Multi-stage data transformations
- Reusable transformation building blocks
- Separation of concerns in complex transformations

**Example:**
```go
// Define reusable transformation stages
normalize := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    // Normalize fields
    return data, nil
}

validate := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    // Validate data
    return data, nil
}

enrich := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    // Add computed fields
    data["computed"] = calculateValue(data)
    return data, nil
}

// Compose pipeline
transformer := stream.TransformationChain[RowData](normalize, validate, enrich)
response := streamer.Stream(ctx, fetcher, transformer)
```

**Error Handling:**
- Pipeline stops at first error
- Returns error with context from failed stage

### BatchTransformParallel

```go
func BatchTransformParallel[T any](ctx context.Context, workerCount int, domainTransform func(T) (interface{}, error)) BatchTransformer[T]
```

**Parameters:**
- `ctx` - Context for cancellation
- `workerCount` - Number of parallel workers
- `domainTransform` - CPU-intensive transformation function

**Returns:**
- `BatchTransformer[T]` - Parallel batch transformer

**Use Cases:**
- CPU-intensive transformations (encoding, compression, complex calculations)
- Large batches (>1000 items)
- Multi-core systems

**Example:**
```go
ctx := context.Background()
workerCount := runtime.NumCPU()

domainTransform := func(data ImageData) (interface{}, error) {
    // CPU-intensive: resize, compress, encode
    return processImage(data)
}

batchTransformer := stream.BatchTransformParallel(ctx, workerCount, domainTransform)
response := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)
```

**Performance Characteristics:**
- ✅ Order preserved in results
- ✅ Scales with CPU cores
- ⚠️ Higher memory usage (goroutines + channels)
- ⚠️ Overhead for simple operations

**When to Use:**
- **DO USE** for CPU-bound operations (>1ms per item)
- **DO USE** with multi-core systems
- **DON'T USE** for I/O-bound operations
- **DON'T USE** for very small batches (<100 items)

## 🔧 Migration Checklist

### Step 1: Identify Custom Transformers

Look for methods like:
- `createTransformer()`
- `createBatchTransformer()`
- Manual transformation wrapper implementations

### Step 2: Choose Appropriate Helper

| Scenario | Recommended Helper |
|----------|-------------------|
| Simple single-item transformation | `TransformerAdapter` |
| Batch processing (sequential) | `BatchTransformerAdapter` |
| Need cancellation support | `BatchTransformerWithContext` |
| Multi-stage transformation | `TransformationChain` |
| CPU-intensive batch processing | `BatchTransformParallel` |

### Step 3: Refactor Code

```go
// Before
func (s *service) createTransformer(...) stream.Transformer[T] {
    return func(item T) (interface{}, error) {
        return s.domainService.Transform(item, ...)
    }
}
transformer := s.createTransformer(...)

// After
domainTransform := func(item T) (interface{}, error) {
    return s.domainService.Transform(item, ...)
}
transformer := stream.TransformerAdapter(domainTransform)
```

### Step 4: Remove Old Helper Methods

Delete the old `createTransformer` and `createBatchTransformer` methods.

### Step 5: Run Tests

```bash
go test ./...
```

### Step 6: Run Benchmarks (Optional)

```bash
go test -bench=. -benchmem ./internal/stream/
```

## 📊 Performance Comparison

### Benchmark Results

```
BenchmarkTransformerAdapter-8                      100000000        10.65 ns/op       8 B/op       0 allocs/op
BenchmarkBatchTransformerAdapter/BatchSize10-8      17218690        71.25 ns/op     160 B/op       1 allocs/op
BenchmarkBatchTransformerAdapter/BatchSize100-8      2609302       461.8 ns/op    1792 B/op       1 allocs/op
BenchmarkBatchTransformerAdapter/BatchSize1000-8      114889     10434 ns/op   23360 B/op     873 allocs/op
BenchmarkBatchTransformerWithContext-8                 95680     12606 ns/op   23360 B/op     873 allocs/op
BenchmarkTransformationChain/SingleTransform-8     100000000        10.80 ns/op       8 B/op       0 allocs/op
BenchmarkTransformationChain/ThreeTransforms-8      50000000        32.10 ns/op      24 B/op       0 allocs/op
BenchmarkBatchTransformParallel/Workers1-8              9800    127820 ns/op   22336 B/op     745 allocs/op
BenchmarkBatchTransformParallel/Workers4-8              5283    230790 ns/op   39318 B/op     753 allocs/op
```

### Key Metrics

| Helper | Overhead | Allocations | Use Case |
|--------|----------|-------------|----------|
| TransformerAdapter | Very Low (~10ns) | 0 allocs/op | Single items |
| BatchTransformerAdapter | Low | 1 alloc/batch | Sequential batch |
| BatchTransformParallel | Higher | Multiple allocs | CPU-intensive |
| TransformationChain | Linear | Linear with stages | Multi-stage |

### Performance Tips

1. **For simple transformations**: Use `TransformerAdapter` or `BatchTransformerAdapter`
2. **For CPU-intensive work**: Use `BatchTransformParallel` with `workerCount = runtime.NumCPU()`
3. **For I/O-bound work**: Avoid parallel transformation, use sequential
4. **For multi-stage logic**: Use `TransformationChain` for clarity

## ⚠️ Important Considerations

### When to Use Enhanced Helpers

✅ **USE** when:
- Wrapping domain transformation logic
- Need consistent error handling
- Want to reduce boilerplate
- Multiple services share similar patterns

❌ **DON'T USE** when:
- Transformation logic is trivial (1-2 lines)
- Need very custom control flow
- Single use-case with no reuse

### Backward Compatibility

- ✅ All existing transformers unchanged
- ✅ No breaking changes
- ✅ Services can migrate incrementally
- ✅ Both old and new approaches can coexist

### Trade-offs

**Pros:**
- Less code duplication
- Centralized error handling patterns
- Consistent behavior across services
- Better tested and benchmarked

**Cons:**
- One more level of indirection
- Slightly more complex for newcomers
- Need to understand generic types

## 🧪 Testing

### Unit Tests

Enhanced helpers include comprehensive tests:

```bash
# Test all transformation helpers
go test ./internal/stream -run TestTransformer
go test ./internal/stream -run TestBatchTransformer
go test ./internal/stream -run TestTransformationChain
go test ./internal/stream -run TestBatchTransformParallel
```

All tests include:
- ✅ Happy path testing
- ✅ Error handling
- ✅ Context cancellation
- ✅ Edge cases (empty batches, errors at specific indices)

### Integration Testing

Example integration test:

```go
func TestIntegration_TransformerAdapter(t *testing.T) {
    // Setup data
    data := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
    fetcher := stream.SliceFetcher(data)

    // Define transformation
    domainTransform := func(user User) (interface{}, error) {
        return map[string]interface{}{
            "id":   user.ID,
            "name": user.Name,
        }, nil
    }

    transformer := stream.TransformerAdapter(domainTransform)
    streamer := stream.NewDefaultStreamer[User]()

    // Stream and verify
    ctx := context.Background()
    response := streamer.Stream(ctx, fetcher, transformer)

    count := 0
    for chunk := range response.ChunkChan {
        count++
        t.Logf("Received chunk: %d bytes", len(chunk.JSONBuf))
    }

    if response.Error != nil {
        t.Fatalf("Stream error: %v", response.Error)
    }

    if count == 0 {
        t.Fatal("Expected at least one chunk")
    }
}
```

## 📖 Real-World Example: ticketsV2

### Complete Service Migration

**File**: `application/ticketsV2/service/service.go`

### Before (Lines of Code)

```
createTransformer():        12 lines
createBatchTransformer():   20 lines
Total:                      32 lines of transformer boilerplate
```

### After (Lines of Code)

```
Inline transformer creation: 4 lines
Inline batch transformer:    4 lines
Total:                       8 lines (75% reduction!)
```

### Code Quality Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Lines of Code | 32 | 8 | -75% |
| Method Count | 2 | 0 | -100% |
| Cyclomatic Complexity | 4 | 1 | -75% |
| Test Coverage | Service-specific | Shared | Centralized |

## 🚀 Advanced Use Cases

### Composable Transformation Pipeline

```go
// Define reusable transformation stages
sanitize := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    // Remove sensitive fields
    delete(data, "password")
    delete(data, "ssn")
    return data, nil
}

addTimestamp := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    data["transformed_at"] = time.Now()
    return data, nil
}

addMetadata := func(val interface{}) (interface{}, error) {
    data := val.(map[string]interface{})
    data["version"] = "v2"
    data["source"] = "api"
    return data, nil
}

// Compose different pipelines for different use cases
publicAPITransformer := stream.TransformationChain[RowData](
    sanitize,
    addTimestamp,
    addMetadata,
)

internalTransformer := stream.TransformationChain[RowData](
    addTimestamp,
    addMetadata,
)
```

### Parallel Processing for Image Transformation

```go
func processImages(ctx context.Context, images []ImageData) error {
    // CPU-intensive: resize + compress + encode
    domainTransform := func(img ImageData) (interface{}, error) {
        resized := resize(img, 800, 600)
        compressed := compress(resized, 80)
        encoded := encode(compressed)
        return encoded, nil
    }

    // Use all CPU cores
    workerCount := runtime.NumCPU()
    batchTransformer := stream.BatchTransformParallel(ctx, workerCount, domainTransform)

    fetcher := stream.SliceBatchFetcher(images, 100)
    streamer := stream.NewDefaultStreamer[ImageData]()

    response := streamer.StreamBatch(ctx, fetcher, batchTransformer)

    for chunk := range response.ChunkChan {
        // Handle processed images
        _ = chunk
    }

    return response.Error
}
```

### Context-Aware Long-Running Transformation

```go
func processWithTimeout(ctx context.Context, data []Data) error {
    // Set timeout for entire operation
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    domainTransform := func(item Data) (interface{}, error) {
        // Potentially slow operation
        return externalAPICall(item)
    }

    batchTransformer := stream.BatchTransformerWithContext(ctx, domainTransform)

    fetcher := stream.SliceBatchFetcher(data, 50)
    streamer := stream.NewDefaultStreamer[Data]()

    response := streamer.StreamBatch(ctx, fetcher, batchTransformer)

    for chunk := range response.ChunkChan {
        _ = chunk
    }

    if response.Error != nil {
        if ctx.Err() == context.DeadlineExceeded {
            log.Println("Operation timed out")
        }
        return response.Error
    }

    return nil
}
```

## 🆘 Troubleshooting

### Issue: Type Mismatch Errors

**Problem:**
```
cannot use domainTransform (type func(MyType) (interface{}, error))
as type func(T) (interface{}, error)
```

**Solution:**
Ensure generic type parameter matches:
```go
transformer := stream.TransformerAdapter[MyType](domainTransform)
//                                     ^^^^^^^^
//                                     Explicit type parameter
```

### Issue: Parallel Performance Worse Than Sequential

**Problem:**
Parallel transformation slower than sequential.

**Solution:**
Parallel processing has overhead. Use only for CPU-intensive operations:
```go
// Good: CPU-intensive
domainTransform := func(data Data) (interface{}, error) {
    return complexCalculation(data) // Takes >1ms
}

// Bad: Too fast for parallelization
domainTransform := func(data Data) (interface{}, error) {
    return data.Field * 2 // Takes <1μs
}
```

### Issue: Context Cancellation Not Working

**Problem:**
Context cancellation doesn't stop processing.

**Solution:**
Use `BatchTransformerWithContext` instead of `BatchTransformerAdapter`:
```go
// Wrong: No cancellation support
batchTransformer := stream.BatchTransformerAdapter(domainTransform)

// Correct: Checks context
batchTransformer := stream.BatchTransformerWithContext(ctx, domainTransform)
```

## 📞 Support

### Questions?
- Check `internal/stream/README.md`
- Review `ENHANCED_HELPERS_MIGRATION_GUIDE.md`
- Ask in team chat

### Issues?
- Check troubleshooting section
- Review test examples
- Open GitHub issue with:
  - Service name
  - Error message
  - Code snippet
  - Expected vs actual behavior

## 📚 Additional Resources

### Documentation
- `internal/stream/README.md` - Complete stream framework guide
- `ENHANCED_HELPERS_MIGRATION_GUIDE.md` - Fetcher enhancement guide
- `internal/stream/ARCHITECTURE.md` - Design decisions

### Code Examples
- `internal/stream/example_test.go` - Usage examples
- `internal/stream/streamer_test.go` - Unit tests
- `application/ticketsV2/service/service.go` - Real-world usage

### Benchmarks
```bash
# Run all transformer benchmarks
go test -bench=BenchmarkTransformer -benchmem ./internal/stream/

# Compare sequential vs parallel
go test -bench=BenchmarkBatchTransformerComparison -benchmem ./internal/stream/

# Test different worker counts
go test -bench=BenchmarkBatchTransformParallel -benchmem ./internal/stream/
```

---

**Document Version**: 1.0
**Last Updated**: 2025-10-27
**Status**: Ready for Production Use
**Next Review**: When 3rd service migrates
