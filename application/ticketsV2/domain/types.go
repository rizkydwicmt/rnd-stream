package domain

import (
	json "github.com/json-iterator/go"
	"github.com/guregu/null/v5"
)

// QueryPayload represents the incoming request payload
// Maintains full compatibility with tickets v1
type QueryPayload struct {
	TableName      string        `json:"tableName" binding:"required"`
	OrderBy        []string      `json:"orderBy"`
	Limit          *int          `json:"limit" binding:"omitempty,min=1"`
	Offset         int           `json:"offset" binding:"min=0"`
	Where          []WhereClause `json:"where"`
	Formulas       []Formula     `json:"formulas"`
	IsFormatDate   bool          `json:"isFormatDate"`
	IsDisableCount bool          `json:"isDisableCount"`
}

// GetLimit returns the limit value, defaulting to 0 (unlimited) if not set
func (q *QueryPayload) GetLimit() int {
	if q.Limit == nil {
		return 0
	}
	return *q.Limit
}

// GetOffset returns the offset value (always set, defaults to 0)
func (q *QueryPayload) GetOffset() int {
	return q.Offset
}

// WhereClause represents a single WHERE condition
type WhereClause struct {
	Field    string      `json:"field" binding:"required"`
	Operator string      `json:"op" binding:"required"`
	Value    interface{} `json:"value" binding:"required"`
}

// Formula represents a transformation formula
type Formula struct {
	Params   []string `json:"params" binding:"required"`
	Field    string   `json:"field" binding:"required"`
	Operator string   `json:"operator"`
	Position int      `json:"position" binding:"required"`
}

// ColumnMetadata holds metadata about a column from the database
type ColumnMetadata struct {
	Name         string
	DatabaseType string
	IsNullable   bool
}

// RowData represents a generic row from database
type RowData map[string]interface{}

// TransformedRow represents the final output after formula transformations
// Uses ordered key-value pairs to maintain field order based on formula position
type TransformedRow struct {
	fields []TransformedField
}

// TransformedField represents a single field in the transformed row
type TransformedField struct {
	Key   string
	Value interface{}
}

// MarshalJSON implements custom JSON marshaling to preserve field order
func (tr TransformedRow) MarshalJSON() ([]byte, error) {
	if len(tr.fields) == 0 {
		return []byte("{}"), nil
	}

	var buf []byte
	buf = append(buf, '{')

	for i, field := range tr.fields {
		if i > 0 {
			buf = append(buf, ',')
		}

		// Marshal key
		keyJSON, err := json.Marshal(field.Key)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyJSON...)
		buf = append(buf, ':')

		// Marshal value
		valueJSON, err := json.Marshal(field.Value)
		if err != nil {
			return nil, err
		}
		buf = append(buf, valueJSON...)
	}

	buf = append(buf, '}')
	return buf, nil
}

// NewTransformedRow creates a new TransformedRow with given fields
func NewTransformedRow(fields []TransformedField) TransformedRow {
	return TransformedRow{fields: fields}
}

// Get returns the value for a given field key
func (tr TransformedRow) Get(key string) (interface{}, bool) {
	for _, field := range tr.fields {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

// Fields returns all fields in order
func (tr TransformedRow) Fields() []TransformedField {
	return tr.fields
}

// OperatorFunc represents a formula operator function signature
type OperatorFunc func(params []interface{}) (interface{}, error)

// ScanValue represents a scannable value that can handle NULL
type ScanValue struct {
	Value interface{}
}

// Scan implements sql.Scanner interface for dynamic scanning
func (sv *ScanValue) Scan(src interface{}) error {
	if src == nil {
		sv.Value = null.String{}
		return nil
	}
	sv.Value = src
	return nil
}

// Configuration constants
const (
	DefaultBatchSize      = 1000
	DefaultChunkThreshold = 32 * 1024 // 32KB
	DefaultBufferSize     = 50 * 1024 // 50KB
)

// Security whitelists
var (
	// AllowedTables is a whitelist of allowed table names
	AllowedTables = map[string]bool{
		"tickets":       true,
		"report_ticket": true,
	}

	// AllowedOperators is a whitelist of allowed WHERE operators
	AllowedOperators = map[string]bool{
		"=":        true,
		"!=":       true,
		">":        true,
		">=":       true,
		"<":        true,
		"<=":       true,
		"LIKE":     true,
		"NOT LIKE": true,
		"IN":       true,
		"NOT IN":   true,
	}

	// AllowedFormulaOperators is a whitelist of allowed formula operators
	AllowedFormulaOperators = map[string]bool{
		"":                    true,
		"ticketIdMasking":     true,
		"difftime":            true,
		"sentimentMapping":    true,
		"escalatedMapping":    true,
		"formatTime":          true,
		"stripHTML":           true,
		"contacts":            true,
		"ticketDate":          true,
		"additionalData":      true,
		"decrypt":             true,
		"stripDecrypt":        true,
		"concat":              true,
		"upper":               true,
		"lower":               true,
		"formatDate":          true,
		"transactionState":    true,
		"length":              true,
		"processSurveyAnswer": true,
	}
)
