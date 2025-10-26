package tickets

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ScanRowGeneric scans a single row into a RowData map using column metadata
func ScanRowGeneric(rows *sql.Rows, columns []string) (RowData, error) {
	// Create slice of interface{} to hold column values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	// Create pointers to scan into
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Build the result map
	result := make(RowData, len(columns))
	for i, colName := range columns {
		result[colName] = values[i]
	}

	return result, nil
}

// GetColumnMetadata extracts column metadata from sql.Rows
func GetColumnMetadata(rows *sql.Rows) ([]ColumnMetadata, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	metadata := make([]ColumnMetadata, len(columns))
	for i, col := range columns {
		nullable, ok := columnTypes[i].Nullable()
		metadata[i] = ColumnMetadata{
			Name:         col,
			DatabaseType: columnTypes[i].DatabaseTypeName(),
			IsNullable:   ok && nullable,
		}
	}

	return metadata, nil
}

// extractAliasFromParam extracts the alias from a SQL expression param
// Returns the alias if the param contains "AS alias", otherwise returns empty string
func extractAliasFromParam(param string) string {
	// Look for " AS alias" pattern (case insensitive)
	upper := strings.ToUpper(param)
	asIndex := strings.LastIndex(upper, " AS ")
	if asIndex == -1 {
		return ""
	}

	// Extract everything after " AS "
	alias := strings.TrimSpace(param[asIndex+4:])

	// Remove any trailing characters that aren't valid in identifiers
	// Stop at first space, comma, or parenthesis
	for i, ch := range alias {
		if ch == ' ' || ch == ',' || ch == ')' {
			alias = alias[:i]
			break
		}
	}

	return alias
}

// TransformRow applies formulas to a RowData to produce TransformedRow
// Formulas MUST be sorted by position before calling this function
func TransformRow(row RowData, formulas []Formula, operators map[string]OperatorFunc) (TransformedRow, error) {
	// Pre-allocate slice with exact size (formulas already sorted by position)
	fields := make([]TransformedField, len(formulas))

	for i, formula := range formulas {
		// Extract parameter values from the row
		paramValues := make([]interface{}, len(formula.Params))
		for j, paramName := range formula.Params {
			// Check if this param is a SQL expression with an alias
			lookupKey := paramName
			if alias := extractAliasFromParam(paramName); alias != "" {
				lookupKey = alias
			}

			val, exists := row[lookupKey]
			if !exists {
				return TransformedRow{}, fmt.Errorf("parameter '%s' (lookup key: '%s') not found in row data", paramName, lookupKey)
			}
			paramValues[j] = val
		}

		// Get the operator function
		operatorFunc, exists := operators[formula.Operator]
		if !exists {
			return TransformedRow{}, fmt.Errorf("operator '%s' not found in registry", formula.Operator)
		}

		// Execute the operator
		transformedValue, err := operatorFunc(paramValues)
		if err != nil {
			return TransformedRow{}, fmt.Errorf("failed to execute operator '%s': %w", formula.Operator, err)
		}

		// Store in ordered slice (maintains position order)
		fields[i] = TransformedField{
			Key:   formula.Field,
			Value: transformedValue,
		}
	}

	return TransformedRow{fields: fields}, nil
}

// BatchTransformRows transforms multiple rows in batch
func BatchTransformRows(rows []RowData, formulas []Formula, operators map[string]OperatorFunc, isFormatDate bool) ([]TransformedRow, error) {
	results := make([]TransformedRow, len(rows))

	for i, row := range rows {
		transformed, err := TransformRow(row, formulas, operators)
		if err != nil {
			return nil, fmt.Errorf("failed to transform row %d: %w", i, err)
		}

		// Post-process: format date* fields if requested
		if isFormatDate {
			transformed = formatDateFields(transformed)
		}

		results[i] = transformed
	}

	return results, nil
}

// formatDateFields formats all fields with "date" prefix to ISO 8601 GMT+7
// Uses stack-allocated timezone for efficiency
func formatDateFields(row TransformedRow) TransformedRow {
	// Stack-allocated GMT+7 timezone (no heap allocation)
	gmt7 := time.FixedZone("GMT+7", 7*3600)

	// Modify fields in-place for efficiency
	for i := range row.fields {
		field := &row.fields[i]

		// Check if field key starts with "date" prefix (case-insensitive)
		if !strings.HasPrefix(strings.ToLower(field.Key), "date") {
			continue
		}

		// Try to convert value to Unix timestamp
		timestamp := toInt64(field.Value)
		if timestamp == 0 {
			// Not a valid timestamp, skip
			continue
		}

		// Convert Unix timestamp to ISO 8601 with GMT+7
		t := time.Unix(timestamp, 0).In(gmt7)
		field.Value = t.Format(time.RFC3339)
	}

	return row
}

// toInt64 converts various numeric types to int64
// Returns 0 if conversion fails or value is 0
func toInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}
