package repository

import (
	"database/sql"
	"fmt"
	"stream/application/ticketsV2/domain"
	"strings"
	"time"
)

// rowScanner implements the RowScanner interface
type rowScanner struct{}

// NewRowScanner creates a new RowScanner instance
func NewRowScanner() domain.RowScanner {
	return &rowScanner{}
}

// ScanRow scans a single row into a RowData map using column metadata
func (rs *rowScanner) ScanRow(rows *sql.Rows, columns []string) (domain.RowData, error) {
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
	result := make(domain.RowData, len(columns))
	for i, colName := range columns {
		result[colName] = values[i]
	}

	return result, nil
}

// transformer implements the Transformer interface
type transformer struct {
	operators map[string]domain.OperatorFunc
}

// NewTransformer creates a new Transformer instance
func NewTransformer(operators map[string]domain.OperatorFunc) domain.Transformer {
	return &transformer{
		operators: operators,
	}
}

// TransformRow applies formulas to a RowData to produce TransformedRow
func (t *transformer) TransformRow(row domain.RowData, formulas []domain.Formula, isFormatDate bool) (domain.TransformedRow, error) {
	// Pre-allocate slice with exact size
	fields := make([]domain.TransformedField, len(formulas))

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
				return domain.TransformedRow{}, fmt.Errorf("parameter '%s' (lookup key: '%s') not found in row data", paramName, lookupKey)
			}
			paramValues[j] = val
		}

		// Get the operator function
		operatorFunc, exists := t.operators[formula.Operator]
		if !exists {
			return domain.TransformedRow{}, fmt.Errorf("operator '%s' not found in registry", formula.Operator)
		}

		// Execute the operator
		transformedValue, err := operatorFunc(paramValues)
		if err != nil {
			return domain.TransformedRow{}, fmt.Errorf("failed to execute operator '%s': %w", formula.Operator, err)
		}

		// Store in ordered slice
		fields[i] = domain.TransformedField{
			Key:   formula.Field,
			Value: transformedValue,
		}
	}

	transformed := domain.NewTransformedRow(fields)

	// Post-process: format date* fields if requested
	if isFormatDate {
		transformed = formatDateFields(transformed)
	}

	return transformed, nil
}

// GetOperatorRegistry returns the map of all available operators
func (t *transformer) GetOperatorRegistry() map[string]domain.OperatorFunc {
	return t.operators
}

// extractAliasFromParam extracts the alias from a SQL expression param
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
	for i, ch := range alias {
		if ch == ' ' || ch == ',' || ch == ')' {
			alias = alias[:i]
			break
		}
	}

	return alias
}

// formatDateFields formats all fields with "date" prefix to ISO 8601 GMT+7
func formatDateFields(row domain.TransformedRow) domain.TransformedRow {
	// Stack-allocated GMT+7 timezone
	gmt7 := time.FixedZone("GMT+7", 7*3600)

	// Get fields and modify in-place
	fields := row.Fields()
	for i := range fields {
		field := &fields[i]

		// Check if field key starts with "date" prefix (case-insensitive)
		if !strings.HasPrefix(strings.ToLower(field.Key), "date") {
			continue
		}

		// Try to convert value to Unix timestamp
		timestamp := toInt64(field.Value)
		if timestamp == 0 {
			continue
		}

		// Convert Unix timestamp to ISO 8601 with GMT+7
		t := time.Unix(timestamp, 0).In(gmt7)
		field.Value = t.Format(time.RFC3339)
	}

	return domain.NewTransformedRow(fields)
}

// toInt64 converts various numeric types to int64
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
