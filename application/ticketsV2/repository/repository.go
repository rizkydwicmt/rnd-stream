package repository

import (
	"context"
	"database/sql"
	"fmt"
	"stream/application/ticketsV2/domain"

	"gorm.io/gorm"
)

// repository implements the Repository interface
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new Repository instance
func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

// ExecuteQuery executes a SELECT query and returns sql.Rows
func (r *repository) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
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

// ExecuteCountQuery executes a COUNT query and returns the count
func (r *repository) ExecuteCountQuery(ctx context.Context, query string, args ...interface{}) (int64, error) {
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

// GetColumnNames extracts column names from sql.Rows
func (r *repository) GetColumnNames(rows *sql.Rows) ([]string, []domain.Formula, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	formulas := make([]domain.Formula, len(columns))
	for i, colName := range columns {
		formulas[i] = domain.Formula{
			Params:   []string{colName},
			Field:    colName,
			Operator: "", // Empty operator = pass-through
			Position: i + 1,
		}
	}
	return columns, formulas, nil
}

// GetColumnMetadata extracts column metadata from sql.Rows
func (r *repository) GetColumnMetadata(rows *sql.Rows) ([]domain.ColumnMetadata, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	metadata := make([]domain.ColumnMetadata, len(columns))
	for i, col := range columns {
		nullable, ok := columnTypes[i].Nullable()
		metadata[i] = domain.ColumnMetadata{
			Name:         col,
			DatabaseType: columnTypes[i].DatabaseTypeName(),
			IsNullable:   ok && nullable,
		}
	}

	return metadata, nil
}

// Close closes the underlying database connection
func (r *repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Close()
}
