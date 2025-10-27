package domain

import (
	"testing"
)

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	t.Run("valid payload", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "tickets",
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
				{Params: []string{"subject"}, Field: "subject", Operator: "", Position: 2},
			},
		}

		err := validator.Validate(payload)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("invalid table name", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "invalid_table",
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
			},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Expected error for invalid table name")
		}
	})

	t.Run("invalid operator", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "tickets",
			Where: []WhereClause{
				{Field: "id", Operator: "INVALID", Value: 1},
			},
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
			},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Expected error for invalid operator")
		}
	})

	t.Run("invalid formula operator", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "tickets",
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "invalidOp", Position: 1},
			},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Expected error for invalid formula operator")
		}
	})

	t.Run("duplicate field names", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "tickets",
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
				{Params: []string{"subject"}, Field: "id", Operator: "", Position: 2},
			},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Expected error for duplicate field names")
		}
	})

	t.Run("SQL injection attempt in field", func(t *testing.T) {
		payload := &QueryPayload{
			TableName: "tickets",
			Where: []WhereClause{
				{Field: "id; DROP TABLE tickets", Operator: "=", Value: 1},
			},
			Formulas: []Formula{
				{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
			},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Expected error for SQL injection attempt")
		}
	})
}

func TestValidator_NormalizeFormulas(t *testing.T) {
	validator := NewValidator()

	t.Run("auto-fill empty field with operator", func(t *testing.T) {
		formulas := []Formula{
			{Params: []string{"id"}, Field: "", Operator: "ticketIdMasking", Position: 1},
			{Params: []string{"subject"}, Field: "subject", Operator: "", Position: 2},
		}

		normalized := validator.NormalizeFormulas(formulas)

		if normalized[0].Field != "ticketIdMasking" {
			t.Errorf("Expected field to be auto-filled with operator, got %s", normalized[0].Field)
		}

		if normalized[1].Field != "subject" {
			t.Errorf("Expected field to remain unchanged, got %s", normalized[1].Field)
		}
	})
}

func TestValidator_SortFormulas(t *testing.T) {
	validator := NewValidator()

	t.Run("sort formulas by position", func(t *testing.T) {
		formulas := []Formula{
			{Params: []string{"subject"}, Field: "subject", Operator: "", Position: 3},
			{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
			{Params: []string{"status"}, Field: "status", Operator: "", Position: 2},
		}

		sorted := validator.SortFormulas(formulas)

		if sorted[0].Field != "id" {
			t.Errorf("Expected first formula to be 'id', got %s", sorted[0].Field)
		}

		if sorted[1].Field != "status" {
			t.Errorf("Expected second formula to be 'status', got %s", sorted[1].Field)
		}

		if sorted[2].Field != "subject" {
			t.Errorf("Expected third formula to be 'subject', got %s", sorted[2].Field)
		}
	})

	t.Run("auto-reposition duplicate positions", func(t *testing.T) {
		formulas := []Formula{
			{Params: []string{"id"}, Field: "id", Operator: "", Position: 1},
			{Params: []string{"subject"}, Field: "subject", Operator: "", Position: 1},
			{Params: []string{"status"}, Field: "status", Operator: "", Position: 2},
		}

		sorted := validator.SortFormulas(formulas)

		// Should be repositioned to 1, 2, 3
		if sorted[0].Position != 1 || sorted[1].Position != 2 || sorted[2].Position != 3 {
			t.Errorf("Expected positions to be auto-repositioned to 1, 2, 3, got %d, %d, %d",
				sorted[0].Position, sorted[1].Position, sorted[2].Position)
		}
	})
}

func TestContainsSuspiciousChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid field name", "user_id", false},
		{"valid field name with underscores", "created_at", false},
		{"semicolon injection", "id; DROP TABLE", true},
		{"comment injection", "id--", true},
		{"quote injection", "id'", true},
		{"exec keyword", "exec", true},
		{"drop keyword", "drop", true},
		{"union keyword", "union", true},
		{"normal word select", "selected_items", false}, // Should be false because it's not the keyword
		{"xp_ prefix", "xp_cmdshell", true},
		{"sp_ prefix", "sp_executesql", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSuspiciousChars(tt.input)
			if result != tt.expected {
				t.Errorf("containsSuspiciousChars(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
