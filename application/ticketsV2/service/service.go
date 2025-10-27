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
	columns, formulas, err := s.repo.GetColumnNames(rows)
	if err != nil {
		rows.Close()
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to get column names: %w", err),
		}
	}

	if sortedFormulas == nil || len(sortedFormulas) == 0 {
		sortedFormulas = formulas
	}

	// Step 8: Create streamer with default configuration
	streamer := stream.NewDefaultStreamer[domain.RowData]()

	// Step 9: Define data fetcher using stream.SQLFetcherWithColumns
	scanner := s.createScanner()
	fetcher := stream.SQLFetcherWithColumns(rows, columns, scanner)

	// Step 10: Define transformer using enhanced helper
	domainTransform := s.createTransformer(sortedFormulas, payload.IsFormatDate)
	transformer := stream.TransformerAdapter(domainTransform)

	// Step 11: Stream using internal/stream package
	streamResp := streamer.Stream(ctx, fetcher, transformer)

	// Step 12: Set total count
	streamResp.TotalCount = totalCount

	return streamResp
}

// createScanner creates an SQLRowScanner that wraps the domain scanner.
// This adapter allows using domain-specific scanner with stream helpers.
func (s *service) createScanner() stream.SQLRowScanner[domain.RowData] {
	return func(rows *sql.Rows, columns []string) (domain.RowData, error) {
		return s.scanner.ScanRow(rows, columns)
	}
}

// createTransformer creates a transformer function that transforms RowData using domain-specific logic.
// This adapter allows using domain-specific transformer with stream helpers.
func (s *service) createTransformer(sortedFormulas []domain.Formula, isFormatDate bool) func(domain.RowData) (interface{}, error) {
	return func(row domain.RowData) (interface{}, error) {
		return s.transformer.TransformRow(row, sortedFormulas, isFormatDate)
	}
}

// StreamTicketsBatch streams ticket data using batch processing for better performance
func (s *service) StreamTicketsBatch(ctx context.Context, payload *domain.QueryPayload) middleware.StreamResponse {
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
	columns, formulas, err := s.repo.GetColumnNames(rows)
	if err != nil {
		rows.Close()
		return middleware.StreamResponse{
			Code:  500,
			Error: fmt.Errorf("failed to get column names: %w", err),
		}
	}

	if sortedFormulas == nil || len(sortedFormulas) == 0 {
		sortedFormulas = formulas
	}

	// Step 8: Create streamer with default configuration
	streamer := stream.NewDefaultStreamer[domain.RowData]()

	// Step 9: Define batch fetcher using stream.SQLBatchFetcherWithColumns
	scanner := s.createScanner()
	batchFetcher := stream.SQLBatchFetcherWithColumns(rows, columns, streamer.GetConfig().BatchSize, scanner)

	// Step 10: Define batch transformer using enhanced helper
	domainTransform := s.createTransformer(sortedFormulas, payload.IsFormatDate)
	batchTransformer := stream.BatchTransformerAdapter(domainTransform)

	// Step 11: Stream using batch processing
	streamResp := streamer.StreamBatch(ctx, batchFetcher, batchTransformer)

	// Step 12: Set total count
	streamResp.TotalCount = totalCount

	return streamResp
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
