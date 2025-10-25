package tickets

import (
	"fmt"
	"sort"
	"strings"
)

// QueryBuilder builds safe SQL queries with parameter binding
type QueryBuilder struct {
	tableName  string
	selectCols []string
	where      []WhereClause
	orderBy    []string
	limit      int
	offset     int
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder(payload *QueryPayload) *QueryBuilder {
	return &QueryBuilder{
		tableName: payload.TableName,
		where:     payload.Where,
		orderBy:   payload.OrderBy,
		limit:     payload.GetLimit(),   // Use getter for default handling
		offset:    payload.GetOffset(),
	}
}

// SetSelectColumns sets the columns to select
func (qb *QueryBuilder) SetSelectColumns(cols []string) {
	qb.selectCols = cols
}

// BuildSelectQuery builds the main SELECT query with parameters
func (qb *QueryBuilder) BuildSelectQuery() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	// SELECT clause
	query.WriteString("SELECT ")
	if len(qb.selectCols) == 0 {
		query.WriteString("*")
	} else {
		// Use backticks to safely quote column names, but pass through SQL expressions
		quotedCols := make([]string, len(qb.selectCols))
		for i, col := range qb.selectCols {
			if isSQLExpression(col) {
				// SQL expression - use as-is
				quotedCols[i] = col
			} else {
				// Regular column - quote it
				quotedCols[i] = quoteIdentifier(col)
			}
		}
		query.WriteString(strings.Join(quotedCols, ", "))
	}

	// FROM clause
	query.WriteString(" FROM ")
	query.WriteString(quoteIdentifier(qb.tableName))

	// WHERE clause
	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		whereParts := make([]string, len(qb.where))
		for i, where := range qb.where {
			whereParts[i], args = qb.buildWhereClause(where, args)
		}
		query.WriteString(strings.Join(whereParts, " AND "))
	}

	// ORDER BY clause
	if len(qb.orderBy) > 0 && len(qb.orderBy) == 2 {
		query.WriteString(" ORDER BY ")
		query.WriteString(quoteIdentifier(qb.orderBy[0]))
		query.WriteString(" ")
		query.WriteString(strings.ToUpper(qb.orderBy[1]))
	}

	// LIMIT clause (only if limit > 0)
	if qb.limit > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, qb.limit)
	}

	// OFFSET clause
	if qb.offset > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, qb.offset)
	}

	return query.String(), args
}

// BuildCountQuery builds a COUNT query
func (qb *QueryBuilder) BuildCountQuery() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	// SELECT COUNT(*)
	query.WriteString("SELECT COUNT(*) FROM ")
	query.WriteString(quoteIdentifier(qb.tableName))

	// WHERE clause (same as main query)
	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		whereParts := make([]string, len(qb.where))
		for i, where := range qb.where {
			whereParts[i], args = qb.buildWhereClause(where, args)
		}
		query.WriteString(strings.Join(whereParts, " AND "))
	}

	return query.String(), args
}

// BuildSampleQuery builds a LIMIT 1 query for metadata sampling
func (qb *QueryBuilder) BuildSampleQuery() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	// SELECT clause
	query.WriteString("SELECT ")
	if len(qb.selectCols) == 0 {
		query.WriteString("*")
	} else {
		quotedCols := make([]string, len(qb.selectCols))
		for i, col := range qb.selectCols {
			if isSQLExpression(col) {
				// SQL expression - use as-is
				quotedCols[i] = col
			} else {
				// Regular column - quote it
				quotedCols[i] = quoteIdentifier(col)
			}
		}
		query.WriteString(strings.Join(quotedCols, ", "))
	}

	// FROM clause
	query.WriteString(" FROM ")
	query.WriteString(quoteIdentifier(qb.tableName))

	// WHERE clause (same as main query)
	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		whereParts := make([]string, len(qb.where))
		for i, where := range qb.where {
			whereParts[i], args = qb.buildWhereClause(where, args)
		}
		query.WriteString(strings.Join(whereParts, " AND "))
	}

	// LIMIT 1
	query.WriteString(" LIMIT 1")

	return query.String(), args
}

// buildWhereClause builds a single WHERE clause with parameter binding
func (qb *QueryBuilder) buildWhereClause(where WhereClause, args []interface{}) (string, []interface{}) {
	var clause strings.Builder

	clause.WriteString(quoteIdentifier(where.Field))
	clause.WriteString(" ")
	clause.WriteString(where.Operator)
	clause.WriteString(" ")

	// Handle IN and NOT IN operators (expect array values)
	upperOp := strings.ToUpper(where.Operator)
	if upperOp == "IN" || upperOp == "NOT IN" {
		// Value should be an array
		switch v := where.Value.(type) {
		case []interface{}:
			placeholders := make([]string, len(v))
			for i, val := range v {
				placeholders[i] = "?"
				args = append(args, val)
			}
			clause.WriteString("(")
			clause.WriteString(strings.Join(placeholders, ", "))
			clause.WriteString(")")
		default:
			// Fallback: treat as single value
			clause.WriteString("(?)")
			args = append(args, where.Value)
		}
	} else {
		// Standard operators: use parameter binding
		clause.WriteString("?")
		args = append(args, where.Value)
	}

	return clause.String(), args
}

// quoteIdentifier safely quotes a SQL identifier (table or column name)
func quoteIdentifier(identifier string) string {
	// Use backticks for SQLite/MySQL compatibility
	// Remove any existing backticks first to prevent injection
	cleaned := strings.ReplaceAll(identifier, "`", "")
	return fmt.Sprintf("`%s`", cleaned)
}

// isSQLExpression checks if a param is a SQL expression (contains AS or SQL functions)
func isSQLExpression(param string) bool {
	upper := strings.ToUpper(param)
	// Check for AS keyword (with spaces around it to avoid matching column names containing "as")
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

// extractAlias extracts the alias from a SQL expression with "AS alias"
// Returns the alias if found, otherwise returns empty string
func extractAlias(expression string) string {
	// Look for " AS alias" pattern (case insensitive)
	upper := strings.ToUpper(expression)
	asIndex := strings.LastIndex(upper, " AS ")
	if asIndex == -1 {
		return ""
	}

	// Extract everything after " AS "
	alias := strings.TrimSpace(expression[asIndex+4:])

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

// GenerateUniqueSelectList generates a unique, deterministic list of columns
// from formulas' params, sorted by formula position
func GenerateUniqueSelectList(formulas []Formula) []string {
	// First, sort formulas by position
	sortedFormulas := make([]Formula, len(formulas))
	copy(sortedFormulas, formulas)
	sort.Slice(sortedFormulas, func(i, j int) bool {
		return sortedFormulas[i].Position < sortedFormulas[j].Position
	})

	// Collect unique params in order
	seen := make(map[string]bool)
	var selectList []string

	for _, formula := range sortedFormulas {
		for _, param := range formula.Params {
			if !seen[param] {
				seen[param] = true
				selectList = append(selectList, param)
			}
		}
	}

	return selectList
}

// SortFormulas sorts formulas by position (ascending) and auto-repositions duplicates
func SortFormulas(formulas []Formula) []Formula {
	sorted := make([]Formula, len(formulas))
	copy(sorted, formulas)

	// Sort by position
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	// Auto-reposition: assign sequential positions starting from 1
	// This handles duplicate positions automatically
	for i := range sorted {
		sorted[i].Position = i + 1
	}

	return sorted
}
