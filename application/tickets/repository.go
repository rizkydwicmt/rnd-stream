package tickets

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

// Repository handles data access for tickets
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ExecuteQuery executes a SELECT query and returns rows
func (r *Repository) ExecuteQuery(ctx context.Context, query string, args []interface{}) (*sql.Rows, error) {
	sqlDB, err := r.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return rows, nil
}

// ExecuteCount executes a COUNT query and returns the count
func (r *Repository) ExecuteCount(ctx context.Context, query string, args []interface{}) (int64, error) {
	sqlDB, err := r.db.DB()
	if err != nil {
		return 0, fmt.Errorf("failed to get database connection: %w", err)
	}

	var count int64
	err = sqlDB.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// FetchRows fetches all rows from a sql.Rows and returns them as RowData slice
func (r *Repository) FetchRows(rows *sql.Rows) ([]RowData, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []RowData

	for rows.Next() {
		row, err := ScanRowGeneric(rows, columns)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// FetchRowsStreaming fetches rows in batches and sends them to a channel
// batchSize controls how many rows to fetch at a time (for memory efficiency)
func (r *Repository) FetchRowsStreaming(rows *sql.Rows, batchSize int) (<-chan []RowData, <-chan error) {
	rowsChan := make(chan []RowData, 2)
	errChan := make(chan error, 1)

	go func() {
		defer close(rowsChan)
		defer close(errChan)
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			errChan <- fmt.Errorf("failed to get columns: %w", err)
			return
		}

		batch := make([]RowData, 0, batchSize)

		for rows.Next() {
			row, err := ScanRowGeneric(rows, columns)
			if err != nil {
				errChan <- fmt.Errorf("failed to scan row: %w", err)
				return
			}

			batch = append(batch, row)

			// Send batch when it reaches batchSize
			if len(batch) >= batchSize {
				// Create a copy to avoid race conditions
				batchCopy := make([]RowData, len(batch))
				copy(batchCopy, batch)
				rowsChan <- batchCopy
				batch = batch[:0] // Reset batch
			}
		}

		// Send remaining rows
		if len(batch) > 0 {
			rowsChan <- batch
		}

		if err := rows.Err(); err != nil {
			errChan <- fmt.Errorf("error iterating rows: %w", err)
		}
	}()

	return rowsChan, errChan
}

// GetColumnMetadataFromQuery executes a LIMIT 1 query to get column metadata
func (r *Repository) GetColumnMetadataFromQuery(ctx context.Context, query string, args []interface{}) ([]ColumnMetadata, error) {
	rows, err := r.ExecuteQuery(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata, err := GetColumnMetadata(rows)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}
