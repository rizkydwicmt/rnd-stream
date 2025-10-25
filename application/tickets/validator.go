package tickets

import (
	"fmt"
	"strings"
)

// ValidatePayload validates the incoming query payload
func ValidatePayload(payload *QueryPayload) error {
	// Validate table name against whitelist
	if !AllowedTables[payload.TableName] {
		return fmt.Errorf("table '%s' is not allowed", payload.TableName)
	}

	// Validate limit if provided (only check minimum)
	if payload.Limit != nil {
		if *payload.Limit < 1 {
			return fmt.Errorf("limit must be >= 1, got %d", *payload.Limit)
		}
	}
	// If limit is null, GetLimit() will return 0 (unlimited)

	// Validate offset
	if payload.Offset < 0 {
		return fmt.Errorf("offset must be >= 0, got %d", payload.Offset)
	}

	// Validate orderBy format
	if len(payload.OrderBy) > 0 {
		if err := validateOrderBy(payload.OrderBy); err != nil {
			return fmt.Errorf("invalid orderBy: %w", err)
		}
	}

	// Validate WHERE clauses
	for i, where := range payload.Where {
		if err := validateWhereClause(&where); err != nil {
			return fmt.Errorf("invalid where clause at index %d: %w", i, err)
		}
	}

	// Validate formulas
	for i, formula := range payload.Formulas {
		if err := validateFormula(&formula); err != nil {
			return fmt.Errorf("invalid formula at index %d: %w", i, err)
		}
	}

	// Note: Duplicate positions are now auto-fixed during SortFormulas()
	// No need to validate for unique positions

	// Check for duplicate formula field names
	if err := validateUniqueFieldNames(payload.Formulas); err != nil {
		return err
	}

	return nil
}

// validateOrderBy validates the orderBy array
// Expected format: ["field_name", "asc|desc"]
func validateOrderBy(orderBy []string) error {
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

	// Basic SQL injection protection: reject suspicious characters
	if containsSuspiciousChars(field) {
		return fmt.Errorf("orderBy field contains invalid characters: '%s'", field)
	}

	return nil
}

// validateWhereClause validates a single WHERE clause
func validateWhereClause(where *WhereClause) error {
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
func validateFormula(formula *Formula) error {
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

	// Validate params
	// Note: SQL expressions are allowed in params (e.g., "COALESCE(...) AS alias")
	// We only validate simple column names, not SQL expressions
	for _, param := range formula.Params {
		// Skip validation for SQL expressions (they contain SQL functions or AS keyword)
		if isSQLExpressionParam(param) {
			// SQL expressions are allowed - skip validation
			continue
		}
		// Regular column name - check for suspicious characters
		if containsSuspiciousChars(param) {
			return fmt.Errorf("formula param contains invalid characters: '%s'", param)
		}
	}

	return nil
}

// validateUniquePositions ensures no duplicate positions in formulas
func validateUniquePositions(formulas []Formula) error {
	positions := make(map[int]bool)
	for _, formula := range formulas {
		if positions[formula.Position] {
			return fmt.Errorf("duplicate formula position: %d", formula.Position)
		}
		positions[formula.Position] = true
	}
	return nil
}

// validateUniqueFieldNames ensures no duplicate field names in formulas
func validateUniqueFieldNames(formulas []Formula) error {
	fields := make(map[string]bool)
	for _, formula := range formulas {
		if fields[formula.Field] {
			return fmt.Errorf("duplicate formula field name: %s", formula.Field)
		}
		fields[formula.Field] = true
	}
	return nil
}

// isSQLExpressionParam checks if a param is a SQL expression
func isSQLExpressionParam(param string) bool {
	upper := strings.ToUpper(param)
	// Check for AS keyword (with spaces around it)
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

	// Check for SQL keywords as standalone words separated by spaces
	// Allow underscores in field names (e.g., created_at, user_id)
	lowerS := strings.ToLower(s)

	// Split by spaces only (not underscores)
	words := strings.Fields(lowerS)

	// If there's only one word (no spaces), check if it's a dangerous keyword
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
		// xp_ and sp_ prefixes (stored procedures)
		if strings.HasPrefix(lowerS, "xp_") || strings.HasPrefix(lowerS, "sp_") {
			return true
		}
		return false
	}

	// If multiple words, check each word
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
