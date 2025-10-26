package tickets

import (
	"testing"
)

func TestValidatePayload(t *testing.T) {
	limit100 := 100
	limit20000 := 20000

	tests := []struct {
		name      string
		payload   *QueryPayload
		wantError bool
	}{
		{
			name: "valid payload",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   []string{"ticket_id", "asc"},
				Limit:     &limit100,
				Offset:    0,
				Where: []WhereClause{
					{Field: "status", Operator: "=", Value: "open"},
				},
				Formulas: []Formula{
					{
						Params:   []string{"ticket_id"},
						Field:    "id",
						Operator: "",
						Position: 2,
					},
					{
						Params:   []string{"ticket_id", "date_create"},
						Field:    "ticket_id_masked",
						Operator: "ticketIdMasking",
						Position: 1,
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid table name",
			payload: &QueryPayload{
				TableName: "users",
				Limit:     &limit100,
			},
			wantError: true,
		},
		{
			name: "high limit is now valid (unlimited supported)",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit20000,
			},
			wantError: false, // Changed: no max limit anymore
		},
		{
			name: "negative offset",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Offset:    -1,
			},
			wantError: true,
		},
		{
			name: "invalid orderBy format",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   []string{"field"},
				Limit:     &limit100,
			},
			wantError: true,
		},
		{
			name: "invalid orderBy direction",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   []string{"field", "invalid"},
				Limit:     &limit100,
			},
			wantError: true,
		},
		{
			name: "invalid where operator",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Where: []WhereClause{
					{Field: "status", Operator: "INVALID", Value: "open"},
				},
			},
			wantError: true,
		},
		{
			name: "SQL injection attempt in orderBy",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   []string{"field; DROP TABLE tickets", "asc"},
				Limit:     &limit100,
			},
			wantError: true,
		},
		{
			name: "duplicate formula positions now auto-fixed",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Formulas: []Formula{
					{Params: []string{"field1"}, Field: "out1", Position: 1},
					{Params: []string{"field2"}, Field: "out2", Position: 1},
				},
			},
			wantError: false, // Changed: duplicates are auto-fixed now
		},
		{
			name: "duplicate formula field names",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Formulas: []Formula{
					{Params: []string{"field1"}, Field: "output", Position: 1},
					{Params: []string{"field2"}, Field: "output", Position: 2},
				},
			},
			wantError: true,
		},
		{
			name: "auto-fill Field from Operator when Field is empty",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Formulas: []Formula{
					{
						Params:   []string{"ticket_id"},
						Field:    "", // Empty field
						Operator: "ticketIdMasking",
						Position: 1,
					},
				},
			},
			wantError: false, // Should pass: Field will be auto-filled with Operator
		},
		{
			name: "empty Field and empty Operator should fail",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Formulas: []Formula{
					{
						Params:   []string{"ticket_id"},
						Field:    "", // Empty field
						Operator: "", // Empty operator (pass-through)
						Position: 1,
					},
				},
			},
			wantError: true, // Should fail: both Field and Operator are empty
		},
		{
			name: "explicit Field takes precedence over Operator",
			payload: &QueryPayload{
				TableName: "tickets",
				Limit:     &limit100,
				Formulas: []Formula{
					{
						Params:   []string{"ticket_id"},
						Field:    "custom_field_name",
						Operator: "ticketIdMasking",
						Position: 1,
					},
				},
			},
			wantError: false, // Should pass: explicit Field is preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePayload(tt.payload)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePayload() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestNormalizeFormulas(t *testing.T) {
	tests := []struct {
		name     string
		input    []Formula
		expected []Formula
	}{
		{
			name: "auto-fill empty Field with Operator",
			input: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "",
					Operator: "ticketIdMasking",
					Position: 1,
				},
			},
			expected: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "ticketIdMasking", // Should be filled
					Operator: "ticketIdMasking",
					Position: 1,
				},
			},
		},
		{
			name: "preserve explicit Field value",
			input: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "custom_name",
					Operator: "ticketIdMasking",
					Position: 1,
				},
			},
			expected: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "custom_name", // Should remain unchanged
					Operator: "ticketIdMasking",
					Position: 1,
				},
			},
		},
		{
			name: "leave empty when both Field and Operator are empty",
			input: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "",
					Operator: "",
					Position: 1,
				},
			},
			expected: []Formula{
				{
					Params:   []string{"ticket_id"},
					Field:    "", // Should remain empty
					Operator: "",
					Position: 1,
				},
			},
		},
		{
			name: "mixed scenarios",
			input: []Formula{
				{
					Params:   []string{"field1"},
					Field:    "",
					Operator: "upper",
					Position: 1,
				},
				{
					Params:   []string{"field2"},
					Field:    "explicit",
					Operator: "lower",
					Position: 2,
				},
				{
					Params:   []string{"field3"},
					Field:    "",
					Operator: "",
					Position: 3,
				},
			},
			expected: []Formula{
				{
					Params:   []string{"field1"},
					Field:    "upper", // Auto-filled
					Operator: "upper",
					Position: 1,
				},
				{
					Params:   []string{"field2"},
					Field:    "explicit", // Preserved
					Operator: "lower",
					Position: 2,
				},
				{
					Params:   []string{"field3"},
					Field:    "", // Remains empty
					Operator: "",
					Position: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			formulas := make([]Formula, len(tt.input))
			copy(formulas, tt.input)

			// Apply normalization
			normalizeFormulas(formulas)

			// Verify results
			for i := range formulas {
				if formulas[i].Field != tt.expected[i].Field {
					t.Errorf("Formula[%d].Field = %q, want %q",
						i, formulas[i].Field, tt.expected[i].Field)
				}
				if formulas[i].Operator != tt.expected[i].Operator {
					t.Errorf("Formula[%d].Operator = %q, want %q",
						i, formulas[i].Operator, tt.expected[i].Operator)
				}
			}
		})
	}
}

func TestContainsSuspiciousChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"normal field", "user_id", false},
		{"semicolon", "field;drop", true},
		{"sql comment", "field--comment", true},
		{"exec keyword", "exec something", true},
		{"drop keyword", "drop table", true},
		{"union keyword", "union select", true},
		{"normal underscore", "field_name", false},
		{"normal number", "field123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsSuspiciousChars(tt.input)
			if got != tt.want {
				t.Errorf("containsSuspiciousChars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
