package repository

import (
	"fmt"
	"sort"
	"stream/application/ticketsV2/domain"
	"strings"
)

// queryBuilder implements the QueryBuilder interface
type queryBuilder struct {
	tableName  string
	selectCols []string
	where      []domain.WhereClause
	orderBy    []string
	limit      int
	offset     int
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder(payload *domain.QueryPayload) domain.QueryBuilder {
	return &queryBuilder{
		tableName: payload.TableName,
		where:     payload.Where,
		orderBy:   payload.OrderBy,
		limit:     payload.GetLimit(),
		offset:    payload.GetOffset(),
	}
}

// SetSelectColumns sets the columns to select
func (qb *queryBuilder) SetSelectColumns(cols []string) {
	qb.selectCols = cols
}

// BuildSelectQuery builds the main SELECT query with parameters
func (qb *queryBuilder) BuildSelectQuery() (string, []interface{}) {
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
				quotedCols[i] = col
			} else {
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
func (qb *queryBuilder) BuildCountQuery() (string, []interface{}) {
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
func (qb *queryBuilder) BuildSampleQuery() (string, []interface{}) {
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
				quotedCols[i] = col
			} else {
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
func (qb *queryBuilder) buildWhereClause(where domain.WhereClause, args []interface{}) (string, []interface{}) {
	var clause strings.Builder

	clause.WriteString(quoteIdentifier(where.Field))
	clause.WriteString(" ")
	clause.WriteString(where.Operator)
	clause.WriteString(" ")

	// Handle IN and NOT IN operators
	upperOp := strings.ToUpper(where.Operator)
	if upperOp == "IN" || upperOp == "NOT IN" {
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
			clause.WriteString("(?)")
			args = append(args, where.Value)
		}
	} else {
		clause.WriteString("?")
		args = append(args, where.Value)
	}

	return clause.String(), args
}

// quoteIdentifier safely quotes a SQL identifier
func quoteIdentifier(identifier string) string {
	cleaned := strings.ReplaceAll(identifier, "`", "")
	return fmt.Sprintf("`%s`", cleaned)
}

// isSQLExpression checks if a param is a SQL expression
func isSQLExpression(param string) bool {
	upper := strings.ToUpper(param)

	if strings.Contains(upper, " AS ") {
		return true
	}

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

	if strings.ContainsAny(param, "+-*/") {
		return true
	}

	return false
}

// GenerateUniqueSelectList generates a unique, deterministic list of columns
// from formulas' params, sorted by formula position
func GenerateUniqueSelectList(formulas []domain.Formula) []string {
	// Sort formulas by position
	sortedFormulas := make([]domain.Formula, len(formulas))
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
