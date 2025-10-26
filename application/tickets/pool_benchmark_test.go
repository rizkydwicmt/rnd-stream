package tickets

import (
	"sync"
	"testing"

	json "github.com/json-iterator/go"
)

// Benchmark different buffer pool sizes
func BenchmarkBufferPool_1KB(b *testing.B)   { benchmarkBufferPool(b, 1*1024) }
func BenchmarkBufferPool_4KB(b *testing.B)   { benchmarkBufferPool(b, 4*1024) }
func BenchmarkBufferPool_8KB(b *testing.B)   { benchmarkBufferPool(b, 8*1024) }
func BenchmarkBufferPool_16KB(b *testing.B)  { benchmarkBufferPool(b, 16*1024) }
func BenchmarkBufferPool_32KB(b *testing.B)  { benchmarkBufferPool(b, 32*1024) }
func BenchmarkBufferPool_50KB(b *testing.B)  { benchmarkBufferPool(b, 50*1024) }
func BenchmarkBufferPool_64KB(b *testing.B)  { benchmarkBufferPool(b, 64*1024) }
func BenchmarkBufferPool_128KB(b *testing.B) { benchmarkBufferPool(b, 128*1024) }
func BenchmarkBufferPool_256KB(b *testing.B) { benchmarkBufferPool(b, 256*1024) }

func benchmarkBufferPool(b *testing.B, bufferSize int) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, bufferSize)
			return &buf
		},
	}

	// Simulate realistic workload: process 100 rows
	rows := generateTestRows(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().(*[]byte)
		*buf = (*buf)[:0]

		// Start JSON array
		*buf = append(*buf, '[')

		// Simulate row processing
		for j, row := range rows {
			jsonData, _ := json.Marshal(row)
			if j > 0 {
				*buf = append(*buf, ',')
			}
			*buf = append(*buf, jsonData...)
		}

		// Close JSON array
		*buf = append(*buf, ']')

		pool.Put(buf)
	}
}

// Benchmark without pool (baseline)
func BenchmarkNoPool_1KB(b *testing.B)   { benchmarkNoPool(b, 1*1024) }
func BenchmarkNoPool_4KB(b *testing.B)   { benchmarkNoPool(b, 4*1024) }
func BenchmarkNoPool_8KB(b *testing.B)   { benchmarkNoPool(b, 8*1024) }
func BenchmarkNoPool_16KB(b *testing.B)  { benchmarkNoPool(b, 16*1024) }
func BenchmarkNoPool_32KB(b *testing.B)  { benchmarkNoPool(b, 32*1024) }
func BenchmarkNoPool_50KB(b *testing.B)  { benchmarkNoPool(b, 50*1024) }
func BenchmarkNoPool_64KB(b *testing.B)  { benchmarkNoPool(b, 64*1024) }
func BenchmarkNoPool_128KB(b *testing.B) { benchmarkNoPool(b, 128*1024) }
func BenchmarkNoPool_256KB(b *testing.B) { benchmarkNoPool(b, 256*1024) }

func benchmarkNoPool(b *testing.B, bufferSize int) {
	// Simulate realistic workload: process 100 rows
	rows := generateTestRows(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := make([]byte, 0, bufferSize)

		// Start JSON array
		buf = append(buf, '[')

		// Simulate row processing
		for j, row := range rows {
			jsonData, _ := json.Marshal(row)
			if j > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, jsonData...)
		}

		// Close JSON array
		buf = append(buf, ']')

		// Simulate usage (prevent optimization)
		_ = buf
	}
}

// Benchmark with different workload sizes
func BenchmarkBufferPool_50KB_10Rows(b *testing.B)   { benchmarkWithRows(b, 50*1024, 10) }
func BenchmarkBufferPool_50KB_100Rows(b *testing.B)  { benchmarkWithRows(b, 50*1024, 100) }
func BenchmarkBufferPool_50KB_500Rows(b *testing.B)  { benchmarkWithRows(b, 50*1024, 500) }
func BenchmarkBufferPool_50KB_1000Rows(b *testing.B) { benchmarkWithRows(b, 50*1024, 1000) }

func BenchmarkBufferPool_32KB_10Rows(b *testing.B)   { benchmarkWithRows(b, 32*1024, 10) }
func BenchmarkBufferPool_32KB_100Rows(b *testing.B)  { benchmarkWithRows(b, 32*1024, 100) }
func BenchmarkBufferPool_32KB_500Rows(b *testing.B)  { benchmarkWithRows(b, 32*1024, 500) }
func BenchmarkBufferPool_32KB_1000Rows(b *testing.B) { benchmarkWithRows(b, 32*1024, 1000) }

func benchmarkWithRows(b *testing.B, bufferSize, numRows int) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, bufferSize)
			return &buf
		},
	}

	rows := generateTestRows(numRows)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().(*[]byte)
		*buf = (*buf)[:0]

		*buf = append(*buf, '[')

		for j, row := range rows {
			jsonData, _ := json.Marshal(row)
			if j > 0 {
				*buf = append(*buf, ',')
			}
			*buf = append(*buf, jsonData...)
		}

		*buf = append(*buf, ']')

		pool.Put(buf)
	}
}

// Benchmark concurrent access (realistic scenario)
func BenchmarkBufferPool_Concurrent_50KB(b *testing.B) {
	benchmarkConcurrent(b, 50*1024)
}

func BenchmarkBufferPool_Concurrent_32KB(b *testing.B) {
	benchmarkConcurrent(b, 32*1024)
}

func benchmarkConcurrent(b *testing.B, bufferSize int) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, bufferSize)
			return &buf
		},
	}

	rows := generateTestRows(100)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get().(*[]byte)
			*buf = (*buf)[:0]

			*buf = append(*buf, '[')

			for j, row := range rows {
				jsonData, _ := json.Marshal(row)
				if j > 0 {
					*buf = append(*buf, ',')
				}
				*buf = append(*buf, jsonData...)
			}

			*buf = append(*buf, ']')

			pool.Put(buf)
		}
	})
}

// Benchmark buffer growth (when initial size is too small)
func BenchmarkBufferGrowth_1KB_100Rows(b *testing.B)  { benchmarkBufferGrowth(b, 1*1024, 100) }
func BenchmarkBufferGrowth_4KB_100Rows(b *testing.B)  { benchmarkBufferGrowth(b, 4*1024, 100) }
func BenchmarkBufferGrowth_8KB_100Rows(b *testing.B)  { benchmarkBufferGrowth(b, 8*1024, 100) }
func BenchmarkBufferGrowth_16KB_100Rows(b *testing.B) { benchmarkBufferGrowth(b, 16*1024, 100) }
func BenchmarkBufferGrowth_32KB_100Rows(b *testing.B) { benchmarkBufferGrowth(b, 32*1024, 100) }
func BenchmarkBufferGrowth_50KB_100Rows(b *testing.B) { benchmarkBufferGrowth(b, 50*1024, 100) }

func benchmarkBufferGrowth(b *testing.B, bufferSize, numRows int) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, bufferSize)
			return &buf
		},
	}

	rows := generateTestRows(numRows)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().(*[]byte)
		*buf = (*buf)[:0]

		*buf = append(*buf, '[')

		for j, row := range rows {
			jsonData, _ := json.Marshal(row)
			if j > 0 {
				*buf = append(*buf, ',')
			}
			*buf = append(*buf, jsonData...)

			// Track reallocations
			// If len > cap, Go has reallocated
		}

		*buf = append(*buf, ']')

		pool.Put(buf)
	}
}

// Benchmark memory overhead of sync.Pool itself
func BenchmarkPoolOverhead(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 50*1024)
			return &buf
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().(*[]byte)
		pool.Put(buf)
	}
}

// Test to verify stack vs heap allocation
func BenchmarkStackAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// This should allocate on stack if small enough
		var buf [64]byte
		_ = buf
	}
}

func BenchmarkHeapAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// This allocates on heap (slice)
		buf := make([]byte, 64)
		_ = buf
	}
}

func BenchmarkPointerAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Returning pointer forces heap allocation
		buf := make([]byte, 0, 50*1024)
		ptr := &buf
		_ = ptr
	}
}

// Helper function to generate test data
func generateTestRows(count int) []map[string]interface{} {
	rows := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		rows[i] = map[string]interface{}{
			"ticket_id":                      12345 + i,
			"date_origin_interaction":        1695984175,
			"date_first_pickup_interaction":  1696000208,
			"date_first_response_interaction": 1696000208,
			"status":                         "open",
			"priority":                       "high",
			"subject":                        "Test ticket subject with some text content",
			"description":                    "This is a test description that simulates real ticket data with reasonable length",
			"assignee":                       "user@example.com",
			"category":                       "technical_support",
		}
	}
	return rows
}

// Test actual JSON size for different row counts
func TestJSONSize(t *testing.T) {
	sizes := []int{1, 10, 50, 100, 500, 1000}

	for _, size := range sizes {
		rows := generateTestRows(size)

		buf := make([]byte, 0, 50*1024)
		buf = append(buf, '[')

		for j, row := range rows {
			jsonData, _ := json.Marshal(row)
			if j > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, jsonData...)
		}

		buf = append(buf, ']')

		t.Logf("%d rows: JSON size = %d bytes (%.2f KB)", size, len(buf), float64(len(buf))/1024)
	}
}

// Benchmark realistic streaming scenario
func BenchmarkRealisticStreaming_50KB(b *testing.B) {
	benchmarkRealisticStreaming(b, 50*1024)
}

func BenchmarkRealisticStreaming_32KB(b *testing.B) {
	benchmarkRealisticStreaming(b, 32*1024)
}

func benchmarkRealisticStreaming(b *testing.B, bufferSize int) {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, bufferSize)
			return &buf
		},
	}

	// Simulate batch processing: 1000 rows total, batch size 100
	allRows := generateTestRows(1000)
	batchSize := 100
	chunkThreshold := 32 * 1024 // 32KB chunk threshold (from actual code)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().(*[]byte)
		*buf = (*buf)[:0]
		*buf = append(*buf, '[')

		chunksCreated := 0

		for batchStart := 0; batchStart < len(allRows); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(allRows) {
				batchEnd = len(allRows)
			}

			batch := allRows[batchStart:batchEnd]

			for j, row := range batch {
				jsonData, _ := json.Marshal(row)
				if len(*buf) > 1 {
					*buf = append(*buf, ',')
				}
				*buf = append(*buf, jsonData...)

				// Simulate chunk sending (like actual code)
				if len(*buf) > chunkThreshold {
					// Simulate sending chunk
					chunksCreated++

					// Get new buffer
					pool.Put(buf)
					buf = pool.Get().(*[]byte)
					*buf = (*buf)[:0]
				}

				_ = j // prevent unused warning
			}
		}

		*buf = append(*buf, ']')
		pool.Put(buf)
	}
}
