package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"stream/application/ticketsV2/domain"
	"stream/application/ticketsV2/repository"
	"stream/internal/stream"
	"stream/middleware"
	"time"
)

// service implements the Service interface
type service struct {
	repo        domain.Repository
	validator   domain.Validator
	transformer domain.Transformer
	scanner     domain.RowScanner
}

// NewService creates a new Service instance
func NewService(repo domain.Repository) domain.Service {
	operators := repository.GetOperatorRegistry()

	return &service{
		repo:        repo,
		validator:   domain.NewValidator(),
		transformer: repository.NewTransformer(operators),
		scanner:     repository.NewRowScanner(),
	}
}

// StreamTickets streams ticket data using the internal/stream package
func (s *service) StreamTickets(ctx context.Context, payload *domain.QueryPayload) middleware.StreamResponse {
	// Step 1: Validate payload
	if err := s.validator.Validate(payload); err != nil {
		return middleware.StreamResponse{
			Code:  400,
			Error: fmt.Errorf("validation error: %w", err),
		}
	}

	// Step 2: Sort formulas by position
	sortedFormulas := s.validator.SortFormulas(payload.Formulas)

	// Step 3: Generate SELECT list from formulas
	selectList := repository.GenerateUniqueSelectList(sortedFormulas)

	// Step 4: Build queries
	qb := repository.NewQueryBuilder(payload)
	qb.SetSelectColumns(selectList)

	mainQuery, mainArgs := qb.BuildSelectQuery()

	// Step 5: Execute count query (if not disabled)
	var totalCount int64 = -1
	if !payload.IsDisableCount {
		countQuery, countArgs := qb.BuildCountQuery()
		count, err := s.repo.ExecuteCountQuery(ctx, countQuery, countArgs...)
		if err != nil {
			return middleware.StreamResponse{
				Code:  500,
				Error: fmt.Errorf("failed to execute count query: %w", err),
			}
		}
		totalCount = count
	}

	// Step 6: Execute main query
	rows, err := s.repo.ExecuteQuery(ctx, mainQuery, mainArgs...)
	if err != nil {
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to execute main query: %w", err),
		}
	}

	// Step 7: Get column names
	columns, err := s.repo.GetColumnNames(rows)
	if err != nil {
		rows.Close()
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to get column names: %w", err),
		}
	}

	// Step 8: Create streamer with default configuration
	streamer := stream.NewDefaultStreamer[domain.RowData]()

	// Step 9: Define data fetcher
	fetcher := s.createFetcher(ctx, rows, columns)

	// Step 10: Define transformer
	transformer := s.createTransformer(sortedFormulas, payload.IsFormatDate)

	// Step 11: Stream using internal/stream package
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Step 12: Set total count
	streamResp.TotalCount = totalCount

	return streamResp
}

// createFetcher creates a DataFetcher for streaming rows
func (s *service) createFetcher(ctx context.Context, rows *sql.Rows, columns []string) stream.DataFetcher[domain.RowData] {
	return func(ctx context.Context) (<-chan domain.RowData, <-chan error) {
		dataChan := make(chan domain.RowData, 10)
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
				row, err := s.scanner.ScanRow(rows, columns)
				if err != nil {
					errChan <- fmt.Errorf("failed to scan row: %w", err)
					return
				}

				// Send row to channel
				select {
				case dataChan <- row:
				case <-ctx.Done():
					return
				}
			}

			// Check for errors during iteration
			if err := rows.Err(); err != nil {
				errChan <- fmt.Errorf("error iterating rows: %w", err)
			}
		}()

		return dataChan, errChan
	}
}

// createTransformer creates a Transformer function
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

// LogRequest logs request information
func (s *service) LogRequest(requestID string, payload *domain.QueryPayload, duration interface{}, err error) {
	var durationMs int64
	switch v := duration.(type) {
	case time.Duration:
		durationMs = v.Milliseconds()
	case int64:
		durationMs = v
	default:
		durationMs = 0
	}

	status := "success"
	errorMsg := ""
	if err != nil {
		status = "error"
		errorMsg = err.Error()
	}

	log.Printf("[%s] table=%s limit=%d offset=%d formulas=%d duration=%dms status=%s error=%s",
		requestID,
		payload.TableName,
		payload.GetLimit(),
		payload.GetOffset(),
		len(payload.Formulas),
		durationMs,
		status,
		errorMsg,
	)
}
