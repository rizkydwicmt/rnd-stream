package tickets

import (
	"context"
	"database/sql"
	"fmt"
	"stream/middleware"
	"sync"
	"time"

	json "github.com/json-iterator/go"
)

// Service handles business logic for tickets streaming
type Service struct {
	repo      *Repository
	operators map[string]OperatorFunc
}

// NewService creates a new Service
func NewService(repo *Repository) *Service {
	return &Service{
		repo:      repo,
		operators: GetOperatorRegistry(),
	}
}

// StreamTickets processes the query payload and streams results
func (s *Service) StreamTickets(ctx context.Context, payload *QueryPayload) middleware.StreamResponse {
	// Validate payload
	if err := ValidatePayload(payload); err != nil {
		return middleware.StreamResponse{
			Code:  400,
			Error: fmt.Errorf("validation failed: %w", err),
		}
	}

	// Sort formulas by position
	sortedFormulas := SortFormulas(payload.Formulas)

	// Generate unique select list from formulas
	selectCols := GenerateUniqueSelectList(sortedFormulas)

	// Build queries
	qb := NewQueryBuilder(payload)
	qb.SetSelectColumns(selectCols)

	// Get total count
	countQuery, countArgs := qb.BuildCountQuery()
	totalCount, err := s.repo.ExecuteCount(ctx, countQuery, countArgs)
	if err != nil {
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to get count: %w", err),
		}
	}

	// Log query info
	actualLimit := payload.GetLimit()
	//limitStr := "unlimited"
	//if actualLimit > 0 {
	//	limitStr = fmt.Sprintf("%d", actualLimit)
	//}
	//fmt.Printf("Query: table=%s, limit=%s, offset=%d, where=%d conditions\n",
	//	payload.TableName, limitStr, payload.GetOffset(), len(payload.Where))

	// Build main query
	mainQuery, mainArgs := qb.BuildSelectQuery()

	// Execute main query
	rows, err := s.repo.ExecuteQuery(ctx, mainQuery, mainArgs)
	if err != nil {
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to execute query: %w", err),
		}
	}

	// Handle empty formulas: auto-generate pass-through formulas for all columns
	// This enables SELECT * behavior when formulas is null or empty
	if len(sortedFormulas) == 0 {
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return middleware.StreamResponse{
				Code:  500,
				Error: fmt.Errorf("failed to get columns for auto-formula generation: %w", err),
			}
		}

		// Generate pass-through formulas (empty operator) for each column
		sortedFormulas = make([]Formula, len(columns))
		for i, colName := range columns {
			sortedFormulas[i] = Formula{
				Params:   []string{colName},
				Field:    colName,
				Operator: "", // Empty operator = pass-through
				Position: i + 1,
			}
		}
	}

	// Stream processing with batching
	batchSize := 100 // Process 100 rows at a time
	if actualLimit > 0 && actualLimit < batchSize {
		batchSize = actualLimit
	}

	chunkChan := s.streamProcessing(ctx, rows, sortedFormulas, batchSize, payload.IsFormatDate)

	return middleware.StreamResponse{
		TotalCount: totalCount,
		ChunkChan:  chunkChan,
		Code:       200,
	}
}

// streamProcessing processes rows in batches and sends JSON chunks
func (s *Service) streamProcessing(
	ctx context.Context,
	rows *sql.Rows,
	formulas []Formula,
	batchSize int,
	isFormatDate bool,
) <-chan middleware.StreamChunk {
	chunkChan := make(chan middleware.StreamChunk, 4)

	go func() {
		defer close(chunkChan)
		defer rows.Close()

		// Get buffer from pool for accumulation
		jsonBuf := jsonBufferPool.Get().(*[]byte)
		*jsonBuf = (*jsonBuf)[:0]
		defer jsonBufferPool.Put(jsonBuf)

		// Start JSON array
		*jsonBuf = append(*jsonBuf, '[')

		// Get rows streaming channel
		rowsChan, errChan := s.repo.FetchRowsStreaming(rows, batchSize)

		for {
			select {
			case <-ctx.Done():
				// Context cancelled, stop processing
				return

			case err := <-errChan:
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: err,
					}
					return
				}

			case batch, ok := <-rowsChan:
				if !ok {
					// Channel closed, all rows processed
					// Close JSON array
					*jsonBuf = append(*jsonBuf, ']')

					// Flush final buffer
					chunkChan <- middleware.StreamChunk{
						JSONBuf: jsonBuf,
					}
					// Don't put back to pool, already in defer
					jsonBuf = nil
					return
				}

				// Transform batch
				transformed, err := BatchTransformRows(batch, formulas, s.operators, isFormatDate)
				if err != nil {
					chunkChan <- middleware.StreamChunk{
						Error: fmt.Errorf("transformation failed: %w", err),
					}
					return
				}

				// Accumulate rows into buffer
				for _, row := range transformed {
					// Marshal JSON
					jsonData, err := json.Marshal(row)
					if err != nil {
						chunkChan <- middleware.StreamChunk{
							Error: fmt.Errorf("JSON marshal failed: %w", err),
						}
						return
					}

					// Add comma separator if not first row (length > 1 because of '[')
					if len(*jsonBuf) > 1 {
						*jsonBuf = append(*jsonBuf, ',')
					}
					*jsonBuf = append(*jsonBuf, jsonData...)

					// Send chunk if buffer exceeds 32KB
					if len(*jsonBuf) > 32*1024 {
						chunkChan <- middleware.StreamChunk{
							JSONBuf: jsonBuf,
						}

						// Get new buffer from pool for next chunk
						jsonBuf = jsonBufferPool.Get().(*[]byte)
						*jsonBuf = (*jsonBuf)[:0]
					}
				}
			}
		}
	}()

	return chunkChan
}

// jsonBufferPool is a sync.Pool for JSON encoding buffers
var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 4096) // 4KB initial capacity
		return &buf
	},
}

// LogRequest logs request details for observability
func (s *Service) LogRequest(requestID string, payload *QueryPayload, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	fmt.Printf(
		"[%s] RequestID=%s Table=%s Limit=%d Offset=%d Formulas=%d Duration=%v Status=%s\n",
		time.Now().Format(time.RFC3339),
		requestID,
		payload.TableName,
		payload.GetLimit(),
		payload.GetOffset(),
		len(payload.Formulas),
		duration,
		status,
	)

	if err != nil {
		fmt.Printf("[%s] RequestID=%s Error=%v\n", time.Now().Format(time.RFC3339), requestID, err)
	}
}
