package domain

import (
	"context"
	"database/sql"
	"stream/middleware"
)

// Repository defines the interface for data access operations
type Repository interface {
	// ExecuteQuery executes a SELECT query and returns sql.Rows
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// ExecuteCountQuery executes a COUNT query and returns the count
	ExecuteCountQuery(ctx context.Context, query string, args ...interface{}) (int64, error)

	// GetColumnNames extracts column names from sql.Rows
	GetColumnNames(rows *sql.Rows) ([]string, []Formula, error)

	// GetColumnMetadata extracts column metadata from sql.Rows
	GetColumnMetadata(rows *sql.Rows) ([]ColumnMetadata, error)

	// Close closes the underlying database connection
	Close() error
}

// QueryBuilder defines the interface for building SQL queries
type QueryBuilder interface {
	// SetSelectColumns sets the columns to select
	SetSelectColumns(cols []string)

	// BuildSelectQuery builds the main SELECT query with parameters
	BuildSelectQuery() (string, []interface{})

	// BuildCountQuery builds a COUNT query
	BuildCountQuery() (string, []interface{})

	// BuildSampleQuery builds a LIMIT 1 query for metadata sampling
	BuildSampleQuery() (string, []interface{})
}

// Validator defines the interface for payload validation
type Validator interface {
	// Validate validates the query payload
	Validate(payload *QueryPayload) error

	// NormalizeFormulas normalizes formulas (auto-fill empty fields)
	NormalizeFormulas(formulas []Formula) []Formula

	// SortFormulas sorts formulas by position
	SortFormulas(formulas []Formula) []Formula
}

// Transformer defines the interface for data transformation
type Transformer interface {
	// TransformRow applies formulas to a RowData to produce TransformedRow
	TransformRow(row RowData, formulas []Formula, isFormatDate bool) (TransformedRow, error)

	// GetOperatorRegistry returns the map of all available operators
	GetOperatorRegistry() map[string]OperatorFunc
}

// RowScanner defines the interface for scanning database rows
type RowScanner interface {
	// ScanRow scans a single row into a RowData map
	ScanRow(rows *sql.Rows, columns []string) (RowData, error)
}

// Service defines the interface for business logic operations
type Service interface {
	// StreamTickets streams ticket data using the internal/stream package
	StreamTickets(ctx context.Context, payload *QueryPayload) middleware.StreamResponse

	// StreamTicketsBatch streams ticket data using batch processing for better performance
	StreamTicketsBatch(ctx context.Context, payload *QueryPayload) middleware.StreamResponse

	// LogRequest logs request information
	LogRequest(requestID string, payload *QueryPayload, duration interface{}, err error)
}
