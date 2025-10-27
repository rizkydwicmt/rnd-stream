package repository

import (
	"stream/application/ticketsV2/domain"
	"strings"
	"testing"
)

func TestQueryBuilder_BuildSelectQuery(t *testing.T) {
	t.Run("simple SELECT query", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets`"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("SELECT with specific columns", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
		}

		qb := NewQueryBuilder(payload)
		qb.SetSelectColumns([]string{"id", "subject", "status"})
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT `id`, `subject`, `status` FROM `tickets`"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("SELECT with WHERE clause", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
			Where: []domain.WhereClause{
				{Field: "status", Operator: "=", Value: "open"},
			},
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets` WHERE `status` = ?"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 1 || args[0] != "open" {
			t.Errorf("Expected args [open], got %v", args)
		}
	})

	t.Run("SELECT with multiple WHERE clauses", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
			Where: []domain.WhereClause{
				{Field: "status", Operator: "=", Value: "open"},
				{Field: "priority", Operator: ">", Value: 2},
			},
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets` WHERE `status` = ? AND `priority` > ?"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 2 || args[0] != "open" || args[1] != 2 {
			t.Errorf("Expected args [open 2], got %v", args)
		}
	})

	t.Run("SELECT with IN operator", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
			Where: []domain.WhereClause{
				{Field: "status", Operator: "IN", Value: []interface{}{"open", "pending", "closed"}},
			},
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets` WHERE `status` IN (?, ?, ?)"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("SELECT with ORDER BY", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
			OrderBy:   []string{"created_at", "desc"},
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets` ORDER BY `created_at` DESC"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("SELECT with LIMIT and OFFSET", func(t *testing.T) {
		limit := 10
		payload := &domain.QueryPayload{
			TableName: "tickets",
			Limit:     &limit,
			Offset:    20,
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildSelectQuery()

		expectedQuery := "SELECT * FROM `tickets` LIMIT ? OFFSET ?"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 2 || args[0] != 10 || args[1] != 20 {
			t.Errorf("Expected args [10 20], got %v", args)
		}
	})

	t.Run("SELECT with SQL expression (COALESCE)", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
		}

		qb := NewQueryBuilder(payload)
		qb.SetSelectColumns([]string{"id", "COALESCE(name, 'Unknown') AS display_name"})
		query, _ := qb.BuildSelectQuery()

		// SQL expressions should not be quoted
		if !strings.Contains(query, "COALESCE(name, 'Unknown') AS display_name") {
			t.Errorf("Expected SQL expression to be preserved, got %q", query)
		}
	})
}

func TestQueryBuilder_BuildCountQuery(t *testing.T) {
	t.Run("simple COUNT query", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildCountQuery()

		expectedQuery := "SELECT COUNT(*) FROM `tickets`"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("COUNT with WHERE clause", func(t *testing.T) {
		payload := &domain.QueryPayload{
			TableName: "tickets",
			Where: []domain.WhereClause{
				{Field: "status", Operator: "=", Value: "open"},
			},
		}

		qb := NewQueryBuilder(payload)
		query, args := qb.BuildCountQuery()

		expectedQuery := "SELECT COUNT(*) FROM `tickets` WHERE `status` = ?"
		if query != expectedQuery {
			t.Errorf("Expected query %q, got %q", expectedQuery, query)
		}

		if len(args) != 1 || args[0] != "open" {
			t.Errorf("Expected args [open], got %v", args)
		}
	})
}

func TestGenerateUniqueSelectList(t *testing.T) {
	t.Run("unique columns from formulas", func(t *testing.T) {
		formulas := []domain.Formula{
			{Params: []string{"id", "subject"}, Field: "ticket", Operator: "", Position: 1},
			{Params: []string{"status", "priority"}, Field: "info", Operator: "", Position: 2},
			{Params: []string{"id", "created_at"}, Field: "created", Operator: "", Position: 3},
		}

		selectList := GenerateUniqueSelectList(formulas)

		expected := []string{"id", "subject", "status", "priority", "created_at"}
		if len(selectList) != len(expected) {
			t.Errorf("Expected %d columns, got %d", len(expected), len(selectList))
		}

		for i, col := range expected {
			if selectList[i] != col {
				t.Errorf("Expected column %d to be %s, got %s", i, col, selectList[i])
			}
		}
	})

	t.Run("maintains order by position", func(t *testing.T) {
		formulas := []domain.Formula{
			{Params: []string{"z_field"}, Field: "z", Operator: "", Position: 3},
			{Params: []string{"a_field"}, Field: "a", Operator: "", Position: 1},
			{Params: []string{"m_field"}, Field: "m", Operator: "", Position: 2},
		}

		selectList := GenerateUniqueSelectList(formulas)

		// Should be ordered by position, not alphabetically
		expected := []string{"a_field", "m_field", "z_field"}
		if len(selectList) != len(expected) {
			t.Errorf("Expected %d columns, got %d", len(expected), len(selectList))
		}

		for i, col := range expected {
			if selectList[i] != col {
				t.Errorf("Expected column %d to be %s, got %s", i, col, selectList[i])
			}
		}
	})
}

func TestIsSQLExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple column", "id", false},
		{"column with underscore", "user_id", false},
		{"COALESCE function", "COALESCE(name, 'Unknown')", true},
		{"AS keyword", "name AS display_name", true},
		{"CONCAT function", "CONCAT(first_name, ' ', last_name)", true},
		{"arithmetic", "price * quantity", true},
		{"CAST function", "CAST(value AS INTEGER)", true},
		{"CASE statement", "CASE WHEN status = 1 THEN 'active' END", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSQLExpression(tt.input)
			if result != tt.expected {
				t.Errorf("isSQLExpression(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
