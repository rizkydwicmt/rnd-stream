package tickets

import (
	"strings"
	"testing"
	"time"
)

func TestTicketIdMasking(t *testing.T) {
	tests := []struct {
		name   string
		params []interface{}
		want   string
	}{
		{
			name:   "normal ticket ID",
			params: []interface{}{12345, "2025-01-01"},
			want:   "TICKET-0000012345",
		},
		{
			name:   "short ID with zero padding",
			params: []interface{}{12, nil},
			want:   "TICKET-0000000012",
		},
		{
			name:   "large ID",
			params: []interface{}{9876543210, "2025-01-01"},
			want:   "TICKET-9876543210",
		},
		{
			name:   "ID without date parameter",
			params: []interface{}{456},
			want:   "TICKET-0000000456",
		},
		{
			name:   "ID as float",
			params: []interface{}{123.0},
			want:   "TICKET-0000000123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ticketIdMasking(tt.params)
			if err != nil {
				t.Errorf("ticketIdMasking() error = %v", err)
				return
			}
			if result != tt.want {
				t.Errorf("ticketIdMasking() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestDifftime(t *testing.T) {
	tests := []struct {
		name   string
		params []interface{}
		want   string
	}{
		{
			name:   "1 hour difference",
			params: []interface{}{1609459200, 1609462800},
			want:   "01:00:00",
		},
		{
			name:   "4000 seconds difference",
			params: []interface{}{1000, 5000},
			want:   "01:06:40",
		},
		{
			name:   "reverse order (absolute value)",
			params: []interface{}{5000, 1000},
			want:   "01:06:40",
		},
		{
			name:   "same timestamp",
			params: []interface{}{1000, 1000},
			want:   "00:00:00",
		},
		{
			name:   "zero timestamp",
			params: []interface{}{0, 1000},
			want:   "00:00:00",
		},
		{
			name:   "both zero",
			params: []interface{}{0, 0},
			want:   "00:00:00",
		},
		{
			name:   "large difference (more than 24h)",
			params: []interface{}{100, 90100},
			want:   "25:00:00",
		},
		{
			name:   "invalid params (only 1 param)",
			params: []interface{}{1000},
			want:   "00:00:00",
		},
		{
			name:   "invalid params (3 params)",
			params: []interface{}{1000, 2000, 3000},
			want:   "00:00:00",
		},
		{
			name:   "float timestamps",
			params: []interface{}{1000.0, 5000.0},
			want:   "01:06:40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := difftime(tt.params)
			if err != nil {
				t.Errorf("difftime() error = %v", err)
				return
			}
			if result != tt.want {
				t.Errorf("difftime() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestSentimentMapping(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "positive sentiment",
			params: []interface{}{1},
			want:   "Positive",
		},
		{
			name:   "neutral sentiment",
			params: []interface{}{0},
			want:   "Neutral",
		},
		{
			name:   "negative sentiment",
			params: []interface{}{-1},
			want:   "Negative",
		},
		{
			name:      "invalid sentiment (2)",
			params:    []interface{}{2},
			expectNil: true,
		},
		{
			name:      "invalid sentiment (-5)",
			params:    []interface{}{-5},
			expectNil: true,
		},
		{
			name:   "sentiment as float",
			params: []interface{}{1.0},
			want:   "Positive",
		},
		{
			name:   "sentiment as string",
			params: []interface{}{"1"},
			want:   "Positive",
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sentimentMapping(tt.params)
			if err != nil {
				t.Errorf("sentimentMapping() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("sentimentMapping() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("sentimentMapping() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestConcat(t *testing.T) {
	params := []interface{}{"Hello", "World", 123}
	result, err := concat(params)
	if err != nil {
		t.Errorf("concat() error = %v", err)
		return
	}

	expected := "Hello World 123"
	if result != expected {
		t.Errorf("concat() = %v, want %v", result, expected)
	}
}

func TestUpper(t *testing.T) {
	params := []interface{}{"hello world"}
	result, err := upper(params)
	if err != nil {
		t.Errorf("upper() error = %v", err)
		return
	}

	expected := "HELLO WORLD"
	if result != expected {
		t.Errorf("upper() = %v, want %v", result, expected)
	}
}

func TestLower(t *testing.T) {
	params := []interface{}{"HELLO WORLD"}
	result, err := lower(params)
	if err != nil {
		t.Errorf("lower() error = %v", err)
		return
	}

	expected := "hello world"
	if result != expected {
		t.Errorf("lower() = %v, want %v", result, expected)
	}
}

func TestFormatDate(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		params []interface{}
		want   string
	}{
		{
			name:   "time.Time with default format",
			params: []interface{}{now},
			want:   "2025-01-15",
		},
		{
			name:   "time.Time with custom format",
			params: []interface{}{now, "2006-01-02 15:04:05"},
			want:   "2025-01-15 10:30:00",
		},
		{
			name:   "SQLite byte array format",
			params: []interface{}{[]uint8("2025-01-15 10:30:00")},
			want:   "2025-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatDate(tt.params)
			if err != nil {
				t.Errorf("formatDate() error = %v", err)
				return
			}
			if result != tt.want {
				t.Errorf("formatDate() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestPassThrough(t *testing.T) {
	params := []interface{}{42, "ignored"}
	result, err := passThrough(params)
	if err != nil {
		t.Errorf("passThrough() error = %v", err)
		return
	}

	if result != 42 {
		t.Errorf("passThrough() = %v, want 42", result)
	}
}

func TestEscalatedMapping(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "escalated",
			params: []interface{}{1},
			want:   "escalated",
		},
		{
			name:   "not escalated",
			params: []interface{}{0},
			want:   "not escalated",
		},
		{
			name:      "invalid value (2)",
			params:    []interface{}{2},
			expectNil: true,
		},
		{
			name:      "invalid value (-1)",
			params:    []interface{}{-1},
			expectNil: true,
		},
		{
			name:   "escalated as float",
			params: []interface{}{1.0},
			want:   "escalated",
		},
		{
			name:   "escalated as string",
			params: []interface{}{"1"},
			want:   "escalated",
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := escalatedMapping(tt.params)
			if err != nil {
				t.Errorf("escalatedMapping() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("escalatedMapping() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("escalatedMapping() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "1 hour 1 minute 1 second",
			params: []interface{}{3661},
			want:   "01:01:01",
		},
		{
			name:   "2 hours",
			params: []interface{}{7200},
			want:   "02:00:00",
		},
		{
			name:   "zero seconds",
			params: []interface{}{0},
			want:   "00:00:00",
		},
		{
			name:   "59 seconds",
			params: []interface{}{59},
			want:   "00:00:59",
		},
		{
			name:   "more than 24 hours",
			params: []interface{}{90000},
			want:   "25:00:00",
		},
		{
			name:   "seconds as float",
			params: []interface{}{3661.0},
			want:   "01:01:01",
		},
		{
			name:   "seconds as string",
			params: []interface{}{"3661"},
			want:   "01:01:01",
		},
		{
			name:      "nil param",
			params:    []interface{}{nil},
			expectNil: true,
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatTime(tt.params)
			if err != nil {
				t.Errorf("formatTime() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("formatTime() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("formatTime() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "simple paragraph",
			params: []interface{}{"<p>Hello</p>"},
			want:   "Hello",
		},
		{
			name:   "bold text",
			params: []interface{}{"<b>Bold</b> text"},
			want:   "Bold text",
		},
		{
			name:   "nested tags",
			params: []interface{}{"<div><p><b>Nested</b> content</p></div>"},
			want:   "Nested content",
		},
		{
			name:   "plain text (no HTML)",
			params: []interface{}{"Plain text"},
			want:   "Plain text",
		},
		{
			name:   "empty string",
			params: []interface{}{""},
			want:   "",
		},
		{
			name:   "multiple tags",
			params: []interface{}{"<h1>Title</h1><p>Paragraph</p>"},
			want:   "TitleParagraph",
		},
		{
			name:   "self-closing tags",
			params: []interface{}{"Line 1<br/>Line 2"},
			want:   "Line 1Line 2",
		},
		{
			name:   "tags with attributes",
			params: []interface{}{"<a href='url'>Link</a>"},
			want:   "Link",
		},
		{
			name:   "mixed content",
			params: []interface{}{"Text <b>bold</b> and <i>italic</i> text"},
			want:   "Text bold and italic text",
		},
		{
			name:      "nil param",
			params:    []interface{}{nil},
			expectNil: true,
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
		{
			name:   "numeric input converted to string",
			params: []interface{}{12345},
			want:   "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stripHTML(tt.params)
			if err != nil {
				t.Errorf("stripHTML() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("stripHTML() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("stripHTML() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestContacts(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		expectErr bool
		checkFunc func(t *testing.T, result interface{})
	}{
		{
			name:   "JSON array of contacts",
			params: []interface{}{`[{"contact_type":"email","contact_value":"test@example.com"}]`},
			checkFunc: func(t *testing.T, result interface{}) {
				// contacts() returns []map[string]interface{}
				contacts, ok := result.([]map[string]interface{})
				if !ok || len(contacts) == 0 {
					t.Error("Expected contacts array")
					return
				}
				if contacts[0]["contact_type"] != "email" {
					t.Errorf("Expected email contact type, got %v", contacts[0]["contact_type"])
				}
			},
		},
		{
			name:   "JSON object with contacts key",
			params: []interface{}{`{"contacts":[{"contact_type":"phone","contact_value":"123456"}]}`},
			checkFunc: func(t *testing.T, result interface{}) {
				// contacts() returns []map[string]interface{}
				contacts, ok := result.([]map[string]interface{})
				if !ok || len(contacts) == 0 {
					t.Error("Expected contacts array")
					return
				}
				if contacts[0]["contact_type"] != "phone" {
					t.Errorf("Expected phone contact type, got %v", contacts[0]["contact_type"])
				}
			},
		},
		{
			name:   "empty string",
			params: []interface{}{""},
			checkFunc: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				if len(resultMap) != 0 {
					t.Error("Expected empty map for empty string")
				}
			},
		},
		{
			name:   "nil param",
			params: []interface{}{nil},
			checkFunc: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				if len(resultMap) != 0 {
					t.Error("Expected empty map for nil")
				}
			},
		},
		{
			name:   "no params",
			params: []interface{}{},
			checkFunc: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				if len(resultMap) != 0 {
					t.Error("Expected empty map for no params")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := contacts(tt.params)
			if (err != nil) != tt.expectErr {
				t.Errorf("contacts() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestTicketDate(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		expectErr bool
		checkFunc func(t *testing.T, result interface{})
	}{
		{
			name:   "JSON array with status dates",
			params: []interface{}{`[{"status_id":1,"date_create":"2024-01-15 10:30:00"}]`},
			checkFunc: func(t *testing.T, result interface{}) {
				// ticketDate() returns []map[string]interface{}
				statusDates, ok := result.([]map[string]interface{})
				if !ok || len(statusDates) == 0 {
					t.Error("Expected status_dates array")
					return
				}
				if statusDates[0]["date_create"] == nil {
					t.Error("Expected date_create to be formatted")
				}
			},
		},
		{
			name:   "with custom date format",
			params: []interface{}{`[{"status_id":1,"date_create":"2024-01-15 10:30:00"}]`, "2006-01-02"},
			checkFunc: func(t *testing.T, result interface{}) {
				// ticketDate() returns []map[string]interface{}
				statusDates, ok := result.([]map[string]interface{})
				if !ok || len(statusDates) == 0 {
					t.Error("Expected status_dates array")
					return
				}
				dateStr, ok := statusDates[0]["date_create"].(string)
				if !ok {
					t.Error("Expected string date")
					return
				}
				if !strings.Contains(dateStr, "2024-01-15") {
					t.Errorf("Expected formatted date, got %s", dateStr)
				}
			},
		},
		{
			name:   "empty string",
			params: []interface{}{""},
			checkFunc: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				if len(resultMap) != 0 {
					t.Error("Expected empty map for empty string")
				}
			},
		},
		{
			name:   "nil param",
			params: []interface{}{nil},
			checkFunc: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				if len(resultMap) != 0 {
					t.Error("Expected empty map for nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ticketDate(tt.params)
			if (err != nil) != tt.expectErr {
				t.Errorf("ticketDate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestAdditionalData(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		expectErr bool
		want      map[string]interface{}
	}{
		{
			name:   "JSON with fields",
			params: []interface{}{`{"field1":"value1","field2":"value2"}`},
			want: map[string]interface{}{
				"additional_field1": "value1",
				"additional_field2": "value2",
			},
		},
		{
			name:   "with custom prefix",
			params: []interface{}{`{"field1":"value1"}`, "custom"},
			want: map[string]interface{}{
				"custom_field1": "value1",
			},
		},
		{
			name:   "with spaces in keys",
			params: []interface{}{`{"Customer Name":"John Doe"}`},
			want: map[string]interface{}{
				"additional_Customer_Name": "John Doe",
			},
		},
		{
			name:   "empty string",
			params: []interface{}{""},
			want:   map[string]interface{}{},
		},
		{
			name:   "nil param",
			params: []interface{}{nil},
			want:   map[string]interface{}{},
		},
		{
			name:   "no params",
			params: []interface{}{},
			want:   map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := additionalData(tt.params)
			if (err != nil) != tt.expectErr {
				t.Errorf("additionalData() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Error("Expected map result")
				return
			}

			if len(resultMap) != len(tt.want) {
				t.Errorf("Expected %d fields, got %d", len(tt.want), len(resultMap))
				return
			}

			for key, expectedValue := range tt.want {
				actualValue, exists := resultMap[key]
				if !exists {
					t.Errorf("Expected key %s not found", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Key %s: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestGetOperatorRegistry(t *testing.T) {
	registry := GetOperatorRegistry()

	requiredOps := []string{
		"",
		"ticketIdMasking",
		"difftime",
		"sentimentMapping",
		"escalatedMapping",
		"formatTime",
		"stripHTML",
		"contacts",
		"ticketDate",
		"additionalData",
		"decrypt",
		"stripDecrypt",
		"transactionState",
		"length",
		"processSurveyAnswer",
		"concat",
		"upper",
		"lower",
		"formatDate",
	}

	for _, op := range requiredOps {
		if _, exists := registry[op]; !exists {
			t.Errorf("Operator '%s' not found in registry", op)
		}
	}

	// Verify all operators are callable
	t.Run("verify operators are callable", func(t *testing.T) {
		// Test difftime is callable
		if op, exists := registry["difftime"]; exists {
			result, err := op([]interface{}{1000, 2000})
			if err != nil {
				t.Errorf("difftime operator failed: %v", err)
			}
			if result != "00:16:40" {
				t.Errorf("difftime returned unexpected result: %v", result)
			}
		}

		// Test sentimentMapping is callable
		if op, exists := registry["sentimentMapping"]; exists {
			result, err := op([]interface{}{1})
			if err != nil {
				t.Errorf("sentimentMapping operator failed: %v", err)
			}
			if result != "Positive" {
				t.Errorf("sentimentMapping returned unexpected result: %v", result)
			}
		}

		// Test escalatedMapping is callable
		if op, exists := registry["escalatedMapping"]; exists {
			result, err := op([]interface{}{1})
			if err != nil {
				t.Errorf("escalatedMapping operator failed: %v", err)
			}
			if result != "escalated" {
				t.Errorf("escalatedMapping returned unexpected result: %v", result)
			}
		}

		// Test formatTime is callable
		if op, exists := registry["formatTime"]; exists {
			result, err := op([]interface{}{3661})
			if err != nil {
				t.Errorf("formatTime operator failed: %v", err)
			}
			if result != "01:01:01" {
				t.Errorf("formatTime returned unexpected result: %v", result)
			}
		}

		// Test stripHTML is callable
		if op, exists := registry["stripHTML"]; exists {
			result, err := op([]interface{}{"<p>Test</p>"})
			if err != nil {
				t.Errorf("stripHTML operator failed: %v", err)
			}
			if result != "Test" {
				t.Errorf("stripHTML returned unexpected result: %v", result)
			}
		}

		// Test decrypt is callable
		if op, exists := registry["decrypt"]; exists {
			result, err := op([]interface{}{"encrypted_value"})
			if err != nil {
				t.Errorf("decrypt operator failed: %v", err)
			}
			// Placeholder returns same value
			if result != "encrypted_value" {
				t.Errorf("decrypt returned unexpected result: %v", result)
			}
		}

		// Test stripDecrypt is callable
		if op, exists := registry["stripDecrypt"]; exists {
			result, err := op([]interface{}{"<p>Test</p>"})
			if err != nil {
				t.Errorf("stripDecrypt operator failed: %v", err)
			}
			// Should decrypt (placeholder = same) and strip HTML
			if result != "Test" {
				t.Errorf("stripDecrypt returned unexpected result: %v", result)
			}
		}

		// Test transactionState is callable
		if op, exists := registry["transactionState"]; exists {
			result, err := op([]interface{}{0})
			if err != nil {
				t.Errorf("transactionState operator failed: %v", err)
			}
			if result != "primary" {
				t.Errorf("transactionState returned unexpected result: %v", result)
			}

			result2, err2 := op([]interface{}{1})
			if err2 != nil {
				t.Errorf("transactionState operator failed: %v", err2)
			}
			if result2 != "flow 1" {
				t.Errorf("transactionState returned unexpected result: %v", result2)
			}
		}

		// Test length is callable
		if op, exists := registry["length"]; exists {
			result, err := op([]interface{}{[]interface{}{1, 2, 3}})
			if err != nil {
				t.Errorf("length operator failed: %v", err)
			}
			if result != 3 {
				t.Errorf("length returned unexpected result: %v", result)
			}
		}

		// Test processSurveyAnswer is callable
		if op, exists := registry["processSurveyAnswer"]; exists {
			result, err := op([]interface{}{
				`{"q1":"value"}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Question 1"}]}]}`,
			})
			if err != nil {
				t.Errorf("processSurveyAnswer operator failed: %v", err)
			}
			resultStr, ok := result.(string)
			if !ok {
				t.Errorf("processSurveyAnswer returned non-string: %v", result)
			}
			if !strings.Contains(resultStr, "Question 1") {
				t.Errorf("processSurveyAnswer returned unexpected result: %v", resultStr)
			}
		}
	})
}

func TestToString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello", "hello"},
		{"int", 123, "123"},
		{"byte array", []uint8("test"), "test"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if !strings.Contains(result, tt.want) && result != tt.want {
				t.Errorf("toString(%v) = %v, want %v", tt.input, result, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int
	}{
		{"int", 42, 42},
		{"int8", int8(42), 42},
		{"int16", int16(42), 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"uint", uint(42), 42},
		{"uint8", uint8(42), 42},
		{"uint16", uint16(42), 42},
		{"uint32", uint32(42), 42},
		{"uint64", uint64(42), 42},
		{"float32", float32(42.7), 42},
		{"float64", float64(42.7), 42},
		{"string number", "42", 42},
		{"byte array number", []uint8("42"), 42},
		{"nil", nil, 0},
		{"invalid string", "abc", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt(tt.input)
			if result != tt.want {
				t.Errorf("toInt(%v) = %v, want %v", tt.input, result, tt.want)
			}
		})
	}
}

func TestSecondsToHHMMSS(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"zero", 0, "00:00:00"},
		{"1 second", 1, "00:00:01"},
		{"59 seconds", 59, "00:00:59"},
		{"1 minute", 60, "00:01:00"},
		{"1 hour", 3600, "01:00:00"},
		{"1 hour 1 minute 1 second", 3661, "01:01:01"},
		{"24 hours", 86400, "24:00:00"},
		{"more than 24 hours", 90000, "25:00:00"},
		{"4000 seconds", 4000, "01:06:40"},
		{"negative (absolute)", -3661, "01:01:01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := secondsToHHMMSS(tt.seconds)
			if result != tt.want {
				t.Errorf("secondsToHHMMSS(%d) = %v, want %v", tt.seconds, result, tt.want)
			}
		})
	}
}

// ========================================================================
// BENCHMARK TESTS - Memory Efficiency Verification
// ========================================================================

func BenchmarkDifftime(b *testing.B) {
	params := []interface{}{1000, 5000}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = difftime(params)
	}
}

func BenchmarkSentimentMapping(b *testing.B) {
	params := []interface{}{1}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = sentimentMapping(params)
	}
}

func BenchmarkTicketIdMasking(b *testing.B) {
	params := []interface{}{12345, time.Now()}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ticketIdMasking(params)
	}
}

func BenchmarkToInt(b *testing.B) {
	testCases := []interface{}{
		42,
		"42",
		42.5,
		[]uint8("42"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = toInt(tc)
		}
	}
}

func BenchmarkSecondsToHHMMSS(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = secondsToHHMMSS(3661)
	}
}

func BenchmarkEscalatedMapping(b *testing.B) {
	params := []interface{}{1}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = escalatedMapping(params)
	}
}

func BenchmarkFormatTime(b *testing.B) {
	params := []interface{}{3661}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = formatTime(params)
	}
}

func BenchmarkStripHTML(b *testing.B) {
	params := []interface{}{"<div><p><b>Nested</b> content with <i>multiple</i> tags</p></div>"}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = stripHTML(params)
	}
}

// BenchmarkStripHTMLLarge tests performance with larger HTML content
func BenchmarkStripHTMLLarge(b *testing.B) {
	largeHTML := strings.Repeat("<div><p>Content</p></div>", 100)
	params := []interface{}{largeHTML}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = stripHTML(params)
	}
}

// BenchmarkContacts tests performance of contacts operator
func BenchmarkContacts(b *testing.B) {
	params := []interface{}{`[{"contact_type":"email","contact_value":"test@example.com"},{"contact_type":"phone","contact_value":"1234567890"}]`}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = contacts(params)
	}
}

// BenchmarkTicketDate tests performance of ticketDate operator
func BenchmarkTicketDate(b *testing.B) {
	params := []interface{}{`[{"status_id":1,"date_create":"2024-01-15 10:30:00"},{"status_id":2,"date_create":"2024-01-16 14:20:00"}]`}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ticketDate(params)
	}
}

// BenchmarkAdditionalData tests performance of additionalData operator
func BenchmarkAdditionalData(b *testing.B) {
	params := []interface{}{`{"field1":"value1","field2":"value2","field3":"value3","field4":"value4"}`}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = additionalData(params)
	}
}

func TestDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "encrypted string (placeholder returns same)",
			params: []interface{}{"encrypted_value_123"},
			want:   "encrypted_value_123", // Placeholder implementation returns same value
		},
		{
			name:   "empty string",
			params: []interface{}{""},
			expectNil: true,
		},
		{
			name:      "nil param",
			params:    []interface{}{nil},
			expectNil: true,
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
		{
			name:   "numeric input converted to string",
			params: []interface{}{12345},
			want:   "12345",
		},
		{
			name:   "boolean input converted to string",
			params: []interface{}{true},
			want:   "true",
		},
		{
			name:   "encrypted email example",
			params: []interface{}{"base64_encrypted_email_here"},
			want:   "base64_encrypted_email_here", // Placeholder
		},
		{
			name:   "encrypted phone example",
			params: []interface{}{"base64_encrypted_phone_here"},
			want:   "base64_encrypted_phone_here", // Placeholder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decrypt(tt.params)
			if err != nil {
				t.Errorf("decrypt() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("decrypt() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("decrypt() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestStripDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "encrypted HTML - simple paragraph",
			params: []interface{}{"<p>Hello</p>"},
			want:   "Hello", // Placeholder decrypts as-is, then strips HTML
		},
		{
			name:   "encrypted HTML - bold text",
			params: []interface{}{"<b>Bold</b> text"},
			want:   "Bold text",
		},
		{
			name:   "encrypted HTML - nested tags",
			params: []interface{}{"<div><p><b>Nested</b> content</p></div>"},
			want:   "Nested content",
		},
		{
			name:   "encrypted HTML - plain text (no HTML)",
			params: []interface{}{"Plain text"},
			want:   "Plain text",
		},
		{
			name:      "empty string",
			params:    []interface{}{""},
			expectNil: true,
		},
		{
			name:   "encrypted HTML - multiple tags",
			params: []interface{}{"<h1>Title</h1><p>Paragraph</p>"},
			want:   "TitleParagraph",
		},
		{
			name:   "encrypted HTML - tags with attributes",
			params: []interface{}{"<a href='url'>Link</a>"},
			want:   "Link",
		},
		{
			name:   "encrypted HTML - mixed content",
			params: []interface{}{"Text <b>bold</b> and <i>italic</i> text"},
			want:   "Text bold and italic text",
		},
		{
			name:      "nil param",
			params:    []interface{}{nil},
			expectNil: true,
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
		{
			name:   "numeric input converted and treated",
			params: []interface{}{12345},
			want:   "12345",
		},
		{
			name:   "encrypted email body example",
			params: []interface{}{"<p>Dear customer,</p><p>Thank you for <b>contacting</b> us.</p>"},
			want:   "Dear customer,Thank you for contacting us.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stripDecrypt(tt.params)
			if err != nil {
				t.Errorf("stripDecrypt() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("stripDecrypt() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("stripDecrypt() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

// BenchmarkDecrypt tests performance of decrypt operator
func BenchmarkDecrypt(b *testing.B) {
	params := []interface{}{"base64_encrypted_test_data_here"}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = decrypt(params)
	}
}

// BenchmarkStripDecrypt tests performance of stripDecrypt operator
func BenchmarkStripDecrypt(b *testing.B) {
	params := []interface{}{"<div><p>Encrypted <b>HTML</b> content with <i>multiple</i> tags</p></div>"}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = stripDecrypt(params)
	}
}

// BenchmarkStripDecryptLarge tests performance with larger encrypted HTML content
func BenchmarkStripDecryptLarge(b *testing.B) {
	largeHTML := "<html><body>" + strings.Repeat("<div><p>Content</p></div>", 100) + "</body></html>"
	params := []interface{}{largeHTML}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = stripDecrypt(params)
	}
}

// ========================================================================
// NEW OPERATORS: transactionState & length - Unit Tests
// ========================================================================

func TestTransactionState(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		want      string
		expectNil bool
	}{
		{
			name:   "primary state - integer 0",
			params: []interface{}{0},
			want:   "primary",
		},
		{
			name:   "primary state - string '0'",
			params: []interface{}{"0"},
			want:   "primary",
		},
		{
			name:   "primary state - float 0.0",
			params: []interface{}{0.0},
			want:   "primary",
		},
		{
			name:   "flow state - integer 1",
			params: []interface{}{1},
			want:   "flow 1",
		},
		{
			name:   "flow state - integer 2",
			params: []interface{}{2},
			want:   "flow 2",
		},
		{
			name:   "flow state - integer 3",
			params: []interface{}{3},
			want:   "flow 3",
		},
		{
			name:   "flow state - string '1'",
			params: []interface{}{"1"},
			want:   "flow 1",
		},
		{
			name:   "flow state - string '2'",
			params: []interface{}{"2"},
			want:   "flow 2",
		},
		{
			name:   "flow state - float 1.0",
			params: []interface{}{1.0},
			want:   "flow 1",
		},
		{
			name:   "flow state - large number",
			params: []interface{}{99},
			want:   "flow 99",
		},
		{
			name:   "flow state - negative number",
			params: []interface{}{-1},
			want:   "flow -1",
		},
		{
			name:   "flow state - string with text",
			params: []interface{}{"active"},
			want:   "flow active",
		},
		{
			name:      "nil param",
			params:    []interface{}{nil},
			expectNil: true,
		},
		{
			name:      "no params",
			params:    []interface{}{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transactionState(tt.params)
			if err != nil {
				t.Errorf("transactionState() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value)
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("transactionState() = %v, want null value", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("transactionState() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestLength(t *testing.T) {
	tests := []struct {
		name   string
		params []interface{}
		want   int
	}{
		{
			name:   "array with 3 elements",
			params: []interface{}{[]interface{}{1, 2, 3}},
			want:   3,
		},
		{
			name:   "array with 2 string elements",
			params: []interface{}{[]interface{}{"a", "b"}},
			want:   2,
		},
		{
			name:   "empty array",
			params: []interface{}{[]interface{}{}},
			want:   0,
		},
		{
			name:   "array with 1 element",
			params: []interface{}{[]interface{}{42}},
			want:   1,
		},
		{
			name:   "array with 10 elements",
			params: []interface{}{[]interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			want:   10,
		},
		{
			name:   "array with mixed types",
			params: []interface{}{[]interface{}{1, "two", 3.0, true, nil}},
			want:   5,
		},
		{
			name:   "array with nested arrays",
			params: []interface{}{[]interface{}{[]interface{}{1, 2}, []interface{}{3, 4}}},
			want:   2,
		},
		{
			name:   "[]any type with elements",
			params: []interface{}{[]any{1, 2, 3}},
			want:   3,
		},
		{
			name:   "[]any type empty",
			params: []interface{}{[]any{}},
			want:   0,
		},
		{
			name:   "string (not an array)",
			params: []interface{}{"string"},
			want:   0,
		},
		{
			name:   "integer (not an array)",
			params: []interface{}{123},
			want:   0,
		},
		{
			name:   "map (not an array)",
			params: []interface{}{map[string]interface{}{"key": "value"}},
			want:   0,
		},
		{
			name:   "nil param",
			params: []interface{}{nil},
			want:   0,
		},
		{
			name:   "no params",
			params: []interface{}{},
			want:   0,
		},
		{
			name:   "boolean (not an array)",
			params: []interface{}{true},
			want:   0,
		},
		{
			name:   "large array (100 elements)",
			params: []interface{}{make([]interface{}, 100)},
			want:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := length(tt.params)
			if err != nil {
				t.Errorf("length() error = %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("length() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestProcessSurveyAnswer tests comprehensive survey answer transformation
func TestProcessSurveyAnswer(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		expectNil bool
		checkFunc func(t *testing.T, result interface{})
	}{
		{
			name: "choice question - single select",
			params: []interface{}{
				`{"q1":"choice_a"}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Favorite Color","choices":[{"value":"choice_a","text":"Red"},{"value":"choice_b","text":"Blue"}]}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Favorite Color":"Red"}
				if !strings.Contains(resultStr, "Favorite Color") || !strings.Contains(resultStr, "Red") {
					t.Errorf("Expected transformed answer with title and choice text, got: %s", resultStr)
				}
			},
		},
		{
			name: "choice question - multi select (array)",
			params: []interface{}{
				`{"q1":["choice_a","choice_b"]}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Favorite Colors","choices":[{"value":"choice_a","text":"Red"},{"value":"choice_b","text":"Blue"}]}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Favorite Colors":"Red,Blue"}
				if !strings.Contains(resultStr, "Favorite Colors") {
					t.Errorf("Expected title 'Favorite Colors', got: %s", resultStr)
				}
				if !strings.Contains(resultStr, "Red") || !strings.Contains(resultStr, "Blue") {
					t.Errorf("Expected both choice texts, got: %s", resultStr)
				}
			},
		},
		{
			name: "boolean question - true",
			params: []interface{}{
				`{"q2":true}`,
				`{"pages":[{"elements":[{"name":"q2","title":"Do you agree?","labelTrue":"Yes","labelFalse":"No"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Do you agree?":"Yes"}
				if !strings.Contains(resultStr, "Do you agree?") || !strings.Contains(resultStr, "Yes") {
					t.Errorf("Expected boolean mapped to 'Yes', got: %s", resultStr)
				}
			},
		},
		{
			name: "boolean question - false",
			params: []interface{}{
				`{"q2":false}`,
				`{"pages":[{"elements":[{"name":"q2","title":"Do you agree?","labelTrue":"Yes","labelFalse":"No"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Do you agree?":"No"}
				if !strings.Contains(resultStr, "Do you agree?") || !strings.Contains(resultStr, "No") {
					t.Errorf("Expected boolean mapped to 'No', got: %s", resultStr)
				}
			},
		},
		{
			name: "multipletext question",
			params: []interface{}{
				`{"q3":{"field1":"John","field2":"Doe"}}`,
				`{"pages":[{"elements":[{"name":"q3","title":"Full Name","type":"multipletext"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Full Name":"John,Doe"} or {"Full Name":"Doe,John"}
				if !strings.Contains(resultStr, "Full Name") {
					t.Errorf("Expected title 'Full Name', got: %s", resultStr)
				}
				// Values concatenated with comma (order may vary due to map iteration)
				if !(strings.Contains(resultStr, "John") && strings.Contains(resultStr, "Doe")) {
					t.Errorf("Expected both field values, got: %s", resultStr)
				}
			},
		},
		{
			name: "matrixdynamic question",
			params: []interface{}{
				`{"q4":[{"col1":"val1","col2":"val2"}]}`,
				`{"pages":[{"elements":[{"name":"q4","title":"Matrix Data","type":"matrixdynamic"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"Matrix Data":"[{...}]"} (JSON string)
				if !strings.Contains(resultStr, "Matrix Data") {
					t.Errorf("Expected title 'Matrix Data', got: %s", resultStr)
				}
			},
		},
		{
			name: "multi-language title",
			params: []interface{}{
				`{"q5":"answer"}`,
				`{"pages":[{"elements":[{"name":"q5","title":{"default":"English Title","id":"Indonesian Title"}}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should use default language
				if !strings.Contains(resultStr, "English Title") {
					t.Errorf("Expected default language title, got: %s", resultStr)
				}
			},
		},
		{
			name: "comment field",
			params: []interface{}{
				`{"q6-Comment":"Additional notes"}`,
				`{"pages":[{"elements":[{"name":"q6","title":"Question 6","commentText":"Comments"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should transform to: {"q6-Comments":"Additional notes"}
				if !strings.Contains(resultStr, "q6-Comments") {
					t.Errorf("Expected comment field with commentText, got: %s", resultStr)
				}
			},
		},
		{
			name: "no questions metadata - return original",
			params: []interface{}{
				`{"q1":"value"}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should return original answer
				if !strings.Contains(resultStr, "q1") {
					t.Errorf("Expected original key preserved, got: %s", resultStr)
				}
			},
		},
		{
			name: "empty answer",
			params: []interface{}{
				`{}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Question"}]}]}`,
			},
			expectNil: true,
		},
		{
			name: "nil answer",
			params: []interface{}{
				nil,
				`{"pages":[{"elements":[{"name":"q1","title":"Question"}]}]}`,
			},
			expectNil: true,
		},
		{
			name: "empty string answer",
			params: []interface{}{
				"",
				`{"pages":[{"elements":[{"name":"q1","title":"Question"}]}]}`,
			},
			expectNil: true,
		},
		{
			name: "invalid JSON answer",
			params: []interface{}{
				`{"q1":invalid}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Question"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				// Should return original string when can't parse
				if result == nil {
					t.Error("Expected original string returned for invalid JSON")
				}
			},
		},
		{
			name: "invalid JSON questions",
			params: []interface{}{
				`{"q1":"value"}`,
				`{"pages":invalid}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// Should return answer as-is if questions invalid
				if !strings.Contains(resultStr, "q1") {
					t.Errorf("Expected original answer preserved, got: %s", resultStr)
				}
			},
		},
		{
			name: "no params",
			params: []interface{}{},
			expectNil: true,
		},
		{
			name: "map input types",
			params: []interface{}{
				map[string]interface{}{"q1": "value"},
				map[string]interface{}{
					"pages": []interface{}{
						map[string]interface{}{
							"elements": []interface{}{
								map[string]interface{}{
									"name":  "q1",
									"title": "Question 1",
								},
							},
						},
					},
				},
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				if !strings.Contains(resultStr, "Question 1") {
					t.Errorf("Expected title transformation, got: %s", resultStr)
				}
			},
		},
		{
			name: "question not found in metadata",
			params: []interface{}{
				`{"q1":"value","q2":"value2"}`,
				`{"pages":[{"elements":[{"name":"q1","title":"Question 1"}]}]}`,
			},
			checkFunc: func(t *testing.T, result interface{}) {
				resultStr, ok := result.(string)
				if !ok {
					t.Error("Expected string result")
					return
				}
				// q1 should be transformed, q2 should remain as key
				if !strings.Contains(resultStr, "Question 1") {
					t.Errorf("Expected q1 transformed, got: %s", resultStr)
				}
				if !strings.Contains(resultStr, "q2") {
					t.Errorf("Expected q2 key preserved, got: %s", resultStr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processSurveyAnswer(tt.params)
			if err != nil {
				t.Errorf("processSurveyAnswer() error = %v", err)
				return
			}

			if tt.expectNil {
				// Check if it's null.String{} (zero value) or empty
				if str, ok := result.(string); ok && str != "" {
					t.Errorf("processSurveyAnswer() = %v, want null/empty value", result)
				}
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// ========================================================================
// NEW OPERATORS: transactionState & length - Benchmark Tests
// ========================================================================

// BenchmarkTransactionState tests performance of transactionState operator
func BenchmarkTransactionState(b *testing.B) {
	b.Run("primary state (0)", func(b *testing.B) {
		params := []interface{}{0}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = transactionState(params)
		}
	})

	b.Run("flow state (1)", func(b *testing.B) {
		params := []interface{}{1}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = transactionState(params)
		}
	})

	b.Run("flow state string", func(b *testing.B) {
		params := []interface{}{"2"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = transactionState(params)
		}
	})

	b.Run("flow state large number", func(b *testing.B) {
		params := []interface{}{999}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = transactionState(params)
		}
	})
}

// BenchmarkLength tests performance of length operator
func BenchmarkLength(b *testing.B) {
	b.Run("small array (3 elements)", func(b *testing.B) {
		params := []interface{}{[]interface{}{1, 2, 3}}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})

	b.Run("medium array (50 elements)", func(b *testing.B) {
		arr := make([]interface{}, 50)
		for i := 0; i < 50; i++ {
			arr[i] = i
		}
		params := []interface{}{arr}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})

	b.Run("large array (1000 elements)", func(b *testing.B) {
		arr := make([]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			arr[i] = i
		}
		params := []interface{}{arr}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})

	b.Run("empty array", func(b *testing.B) {
		params := []interface{}{[]interface{}{}}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})

	b.Run("non-array input", func(b *testing.B) {
		params := []interface{}{"not an array"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})

	b.Run("[]any type", func(b *testing.B) {
		params := []interface{}{[]any{1, 2, 3, 4, 5}}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = length(params)
		}
	})
}

// BenchmarkProcessSurveyAnswer tests performance of processSurveyAnswer operator
func BenchmarkProcessSurveyAnswer(b *testing.B) {
	b.Run("simple choice question", func(b *testing.B) {
		params := []interface{}{
			`{"q1":"choice_a"}`,
			`{"pages":[{"elements":[{"name":"q1","title":"Favorite Color","choices":[{"value":"choice_a","text":"Red"}]}]}]}`,
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("multi-select with 5 choices", func(b *testing.B) {
		params := []interface{}{
			`{"q1":["choice_a","choice_b","choice_c","choice_d","choice_e"]}`,
			`{"pages":[{"elements":[{"name":"q1","title":"Colors","choices":[{"value":"choice_a","text":"Red"},{"value":"choice_b","text":"Blue"},{"value":"choice_c","text":"Green"},{"value":"choice_d","text":"Yellow"},{"value":"choice_e","text":"Purple"}]}]}]}`,
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("boolean question", func(b *testing.B) {
		params := []interface{}{
			`{"q2":true}`,
			`{"pages":[{"elements":[{"name":"q2","title":"Agree?","labelTrue":"Yes","labelFalse":"No"}]}]}`,
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("multipletext question", func(b *testing.B) {
		params := []interface{}{
			`{"q3":{"field1":"John","field2":"Doe","field3":"john@example.com"}}`,
			`{"pages":[{"elements":[{"name":"q3","title":"Contact Info","type":"multipletext"}]}]}`,
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("multiple questions (3 fields)", func(b *testing.B) {
		params := []interface{}{
			`{"q1":"choice_a","q2":true,"q3":"text answer"}`,
			`{"pages":[{"elements":[{"name":"q1","title":"Question 1","choices":[{"value":"choice_a","text":"Answer A"}]},{"name":"q2","title":"Question 2","labelTrue":"Yes","labelFalse":"No"},{"name":"q3","title":"Question 3"}]}]}`,
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("complex survey (10 questions)", func(b *testing.B) {
		// Build a complex survey with 10 questions
		answerData := `{"q1":"a","q2":"b","q3":"c","q4":"d","q5":"e","q6":"f","q7":"g","q8":"h","q9":"i","q10":"j"}`
		questionsData := `{"pages":[{"elements":[` +
			`{"name":"q1","title":"Q1","choices":[{"value":"a","text":"A1"}]},` +
			`{"name":"q2","title":"Q2","choices":[{"value":"b","text":"B1"}]},` +
			`{"name":"q3","title":"Q3","choices":[{"value":"c","text":"C1"}]},` +
			`{"name":"q4","title":"Q4","choices":[{"value":"d","text":"D1"}]},` +
			`{"name":"q5","title":"Q5","choices":[{"value":"e","text":"E1"}]},` +
			`{"name":"q6","title":"Q6","choices":[{"value":"f","text":"F1"}]},` +
			`{"name":"q7","title":"Q7","choices":[{"value":"g","text":"G1"}]},` +
			`{"name":"q8","title":"Q8","choices":[{"value":"h","text":"H1"}]},` +
			`{"name":"q9","title":"Q9","choices":[{"value":"i","text":"I1"}]},` +
			`{"name":"q10","title":"Q10","choices":[{"value":"j","text":"J1"}]}` +
			`]}]}`

		params := []interface{}{answerData, questionsData}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("map input (no JSON parsing)", func(b *testing.B) {
		params := []interface{}{
			map[string]interface{}{"q1": "value"},
			map[string]interface{}{
				"pages": []interface{}{
					map[string]interface{}{
						"elements": []interface{}{
							map[string]interface{}{
								"name":  "q1",
								"title": "Question 1",
							},
						},
					},
				},
			},
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})

	b.Run("no transformation (no questions)", func(b *testing.B) {
		params := []interface{}{`{"q1":"value"}`}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = processSurveyAnswer(params)
		}
	})
}
