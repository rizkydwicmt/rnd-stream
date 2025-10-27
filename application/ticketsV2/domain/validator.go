package domain

import (
	"fmt"
	"sort"
	"strings"
)

// validator implements the Validator interface
type validator struct{}

// NewValidator creates a new Validator instance
func NewValidator() Validator {
	return &validator{}
}

// Validate validates the query payload
func (v *validator) Validate(payload *QueryPayload) error {
	// Normalize formulas before validation
	payload.Formulas = v.NormalizeFormulas(payload.Formulas)

	// Validate table name against whitelist
	if !AllowedTables[payload.TableName] {
		return fmt.Errorf("table '%s' is not allowed", payload.TableName)
	}

	// Validate limit if provided
	if payload.Limit != nil {
		if *payload.Limit < 1 {
			return fmt.Errorf("limit must be >= 1, got %d", *payload.Limit)
		}
	}

	// Validate offset
	if payload.Offset < 0 {
		return fmt.Errorf("offset must be >= 0, got %d", payload.Offset)
	}

	// Validate orderBy format
	if len(payload.OrderBy) > 0 {
		if err := v.validateOrderBy(payload.OrderBy); err != nil {
			return fmt.Errorf("invalid orderBy: %w", err)
		}
	}

	// Validate WHERE clauses
	for i, where := range payload.Where {
		if err := v.validateWhereClause(&where); err != nil {
			return fmt.Errorf("invalid where clause at index %d: %w", i, err)
		}
	}

	// Validate formulas
	for i, formula := range payload.Formulas {
		if err := v.validateFormula(&formula); err != nil {
			return fmt.Errorf("invalid formula at index %d: %w", i, err)
		}
	}

	// Check for duplicate formula field names
	if err := v.validateUniqueFieldNames(payload.Formulas); err != nil {
		return err
	}

	return nil
}

// NormalizeFormulas normalizes formulas by auto-filling empty Field with Operator value
func (v *validator) NormalizeFormulas(formulas []Formula) []Formula {
	normalized := make([]Formula, len(formulas))
	copy(normalized, formulas)

	for i := range normalized {
		// If Field is empty but Operator has a value, set Field = Operator
		if normalized[i].Field == "" && normalized[i].Operator != "" {
			normalized[i].Field = normalized[i].Operator
		}
	}

	return normalized
}

// SortFormulas sorts formulas by position and auto-repositions duplicates
func (v *validator) SortFormulas(formulas []Formula) []Formula {
	sorted := make([]Formula, len(formulas))
	copy(sorted, formulas)

	// Sort by position
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	// Auto-reposition: assign sequential positions starting from 1
	for i := range sorted {
		sorted[i].Position = i + 1
	}

	return sorted
}

// validateOrderBy validates the orderBy array
func (v *validator) validateOrderBy(orderBy []string) error {
	if len(orderBy) != 2 {
		return fmt.Errorf("orderBy must have exactly 2 elements [field, direction], got %d", len(orderBy))
	}

	field := orderBy[0]
	direction := strings.ToUpper(orderBy[1])

	if field == "" {
		return fmt.Errorf("orderBy field cannot be empty")
	}

	if direction != "ASC" && direction != "DESC" {
		return fmt.Errorf("orderBy direction must be 'asc' or 'desc', got '%s'", orderBy[1])
	}

	// Basic SQL injection protection
	if containsSuspiciousChars(field) {
		return fmt.Errorf("orderBy field contains invalid characters: '%s'", field)
	}

	return nil
}

// validateWhereClause validates a single WHERE clause
func (v *validator) validateWhereClause(where *WhereClause) error {
	if where.Field == "" {
		return fmt.Errorf("where field cannot be empty")
	}

	if where.Operator == "" {
		return fmt.Errorf("where operator cannot be empty")
	}

	// Validate operator against whitelist
	upperOp := strings.ToUpper(where.Operator)
	if !AllowedOperators[upperOp] {
		return fmt.Errorf("operator '%s' is not allowed", where.Operator)
	}

	// Basic SQL injection protection
	if containsSuspiciousChars(where.Field) {
		return fmt.Errorf("where field contains invalid characters: '%s'", where.Field)
	}

	return nil
}

// validateFormula validates a single formula
func (v *validator) validateFormula(formula *Formula) error {
	if len(formula.Params) == 0 {
		return fmt.Errorf("formula params cannot be empty")
	}

	if formula.Field == "" {
		return fmt.Errorf("formula field cannot be empty")
	}

	if formula.Position < 0 {
		return fmt.Errorf("formula position must be >= 0, got %d", formula.Position)
	}

	// Validate operator against whitelist
	if !AllowedFormulaOperators[formula.Operator] {
		return fmt.Errorf("formula operator '%s' is not allowed", formula.Operator)
	}

	// Validate params (skip SQL expressions)
	for _, param := range formula.Params {
		if isSQLExpression(param) {
			continue
		}
		if containsSuspiciousChars(param) {
			return fmt.Errorf("formula param contains invalid characters: '%s'", param)
		}
	}

	return nil
}

// validateUniqueFieldNames ensures no duplicate field names in formulas
func (v *validator) validateUniqueFieldNames(formulas []Formula) error {
	fields := make(map[string]bool)
	for _, formula := range formulas {
		if fields[formula.Field] {
			return fmt.Errorf("duplicate formula field name: %s", formula.Field)
		}
		fields[formula.Field] = true
	}
	return nil
}

// isSQLExpression checks if a param is a SQL expression
func isSQLExpression(param string) bool {
	upper := strings.ToUpper(param)

	// Check for AS keyword
	if strings.Contains(upper, " AS ") {
		return true
	}

	// Check for common SQL functions
	sqlFunctions := []string{
		"COALESCE(", "CONCAT(", "UPPER(", "LOWER(", "TRIM(",
		"SUBSTR(", "SUBSTRING(", "LENGTH(", "ABS(", "ROUND(",
		"FLOOR(", "CEIL(", "SEC_TO_TIME(", "TIME_TO_SEC(",
		"DATE(", "TIME(", "DATETIME(", "STRFTIME(",
		"IFNULL(", "NULLIF(", "CAST(", "CASE ",
	}

	for _, fn := range sqlFunctions {
		if strings.Contains(upper, fn) {
			return true
		}
	}

	// Check for arithmetic operations
	if strings.ContainsAny(param, "+-*/") {
		return true
	}

	return false
}

// containsSuspiciousChars checks for common SQL injection patterns
func containsSuspiciousChars(s string) bool {
	// Check for dangerous special characters
	dangerousChars := []string{";", "--", "/*", "*/", "'", "\""}
	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return true
		}
	}

	lowerS := strings.ToLower(s)
	words := strings.Fields(lowerS)

	// Single word check
	if len(words) == 1 {
		dangerousSingle := []string{
			"exec", "execute", "drop", "alter",
			"insert", "update", "delete", "union",
			"select", "from", "where",
		}
		for _, keyword := range dangerousSingle {
			if lowerS == keyword {
				return true
			}
		}
		if strings.HasPrefix(lowerS, "xp_") || strings.HasPrefix(lowerS, "sp_") {
			return true
		}
		return false
	}

	// Multiple words check
	dangerousKeywords := []string{
		"exec", "execute", "drop", "alter",
		"insert", "update", "delete", "union",
	}

	for _, word := range words {
		for _, keyword := range dangerousKeywords {
			if word == keyword {
				return true
			}
		}
		if strings.HasPrefix(word, "xp_") || strings.HasPrefix(word, "sp_") {
			return true
		}
	}

	return false
}
