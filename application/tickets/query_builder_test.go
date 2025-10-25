package tickets

import (
	"strings"
	"testing"
)

func TestQueryBuilder_BuildSelectQuery(t *testing.T) {
	limit := 100
	payload := &QueryPayload{
		TableName: "tickets",
		OrderBy:   []string{"id", "asc"},
		Limit:     &limit,
		Offset:    10,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
			{Field: "priority", Operator: ">=", Value: "medium"},
		},
	}

	qb := NewQueryBuilder(payload)
	qb.SetSelectColumns([]string{"id", "status", "priority"})

	query, args := qb.BuildSelectQuery()

	// Check query structure
	if !strings.Contains(query, "SELECT") {
		t.Error("Query should contain SELECT")
	}
	if !strings.Contains(query, "FROM `tickets`") {
		t.Error("Query should contain FROM tickets")
	}
	if !strings.Contains(query, "WHERE") {
		t.Error("Query should contain WHERE")
	}
	if !strings.Contains(query, "ORDER BY") {
		t.Error("Query should contain ORDER BY")
	}
	if !strings.Contains(query, "LIMIT ?") {
		t.Error("Query should contain LIMIT with placeholder")
	}
	if !strings.Contains(query, "OFFSET ?") {
		t.Error("Query should contain OFFSET with placeholder")
	}

	// Check args count (2 where + 1 limit + 1 offset = 4)
	expectedArgs := 4
	if len(args) != expectedArgs {
		t.Errorf("Expected %d args, got %d", expectedArgs, len(args))
	}

	// Verify no SQL injection (no direct value interpolation)
	if strings.Contains(query, "open") || strings.Contains(query, "medium") {
		t.Error("Query should not contain literal values, only placeholders")
	}
}

func TestQueryBuilder_BuildCountQuery(t *testing.T) {
	limit := 100
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
	}

	qb := NewQueryBuilder(payload)
	query, args := qb.BuildCountQuery()

	if !strings.Contains(query, "SELECT COUNT(*)") {
		t.Error("Count query should contain SELECT COUNT(*)")
	}
	if !strings.Contains(query, "FROM `tickets`") {
		t.Error("Count query should contain FROM tickets")
	}
	if strings.Contains(query, "LIMIT") {
		t.Error("Count query should not contain LIMIT")
	}

	// Should have 1 arg for the WHERE clause
	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}
}

func TestGenerateUniqueSelectList(t *testing.T) {
	formulas := []Formula{
		{
			Params:   []string{"ticket_id", "created_at"},
			Field:    "masked",
			Position: 2,
		},
		{
			Params:   []string{"ticket_id"},
			Field:    "id",
			Position: 1,
		},
		{
			Params:   []string{"status", "created_at"},
			Field:    "info",
			Position: 3,
		},
	}

	selectList := GenerateUniqueSelectList(formulas)

	// Should have unique columns in position order
	expected := []string{"ticket_id", "created_at", "status"}
	if len(selectList) != len(expected) {
		t.Errorf("Expected %d columns, got %d", len(expected), len(selectList))
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, col := range selectList {
		if seen[col] {
			t.Errorf("Duplicate column in select list: %s", col)
		}
		seen[col] = true
	}
}

func TestSortFormulas(t *testing.T) {
	formulas := []Formula{
		{Field: "third", Position: 3},
		{Field: "first", Position: 1},
		{Field: "second", Position: 2},
	}

	sorted := SortFormulas(formulas)

	if sorted[0].Field != "first" {
		t.Errorf("Expected first formula to be 'first', got %s", sorted[0].Field)
	}
	if sorted[1].Field != "second" {
		t.Errorf("Expected second formula to be 'second', got %s", sorted[1].Field)
	}
	if sorted[2].Field != "third" {
		t.Errorf("Expected third formula to be 'third', got %s", sorted[2].Field)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"field_name", "`field_name`"},
		{"table", "`table`"},
		{"`already_quoted`", "`already_quoted`"},
	}

	for _, tt := range tests {
		result := quoteIdentifier(tt.input)
		if result != tt.expected {
			t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
