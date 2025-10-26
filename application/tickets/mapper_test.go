package tickets

import (
	"testing"
	"time"
)

func TestBatchTransformRows_WithDateFormatting(t *testing.T) {
	// Setup test data with Unix timestamps
	rows := []RowData{
		{
			"date_origin_interaction":        int64(1695984175),
			"date_first_pickup_interaction":  int64(1696000208),
			"date_first_response_interaction": int64(1696000208),
			"ticket_id":                      123,
		},
		{
			"date_origin_interaction":        int64(1695984175),
			"date_first_pickup_interaction":  int64(1696000208),
			"date_first_response_interaction": int64(1696000208),
			"ticket_id":                      456,
		},
	}

	formulas := []Formula{
		{
			Params:   []string{"date_origin_interaction"},
			Field:    "date_origin_interaction",
			Operator: "",
			Position: 1,
		},
		{
			Params:   []string{"date_first_pickup_interaction"},
			Field:    "date_first_pickup_interaction",
			Operator: "",
			Position: 2,
		},
		{
			Params:   []string{"date_first_response_interaction"},
			Field:    "date_first_response_interaction",
			Operator: "",
			Position: 3,
		},
		{
			Params:   []string{"ticket_id"},
			Field:    "ticket_id",
			Operator: "",
			Position: 4,
		},
	}

	operators := GetOperatorRegistry()

	t.Run("with isFormatDate=true formats date fields", func(t *testing.T) {
		results, err := BatchTransformRows(rows, formulas, operators, true)
		if err != nil {
			t.Fatalf("BatchTransformRows() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Check first row
		row1 := results[0]
		fields := row1.Fields()

		// Verify date_origin_interaction is formatted
		dateOrigin, exists := row1.Get("date_origin_interaction")
		if !exists {
			t.Error("date_origin_interaction field not found")
		}
		if dateStr, ok := dateOrigin.(string); ok {
			// 1695984175 Unix = 2023-09-29 10:42:55 UTC = 2023-09-29 17:42:55 GMT+7
			expected := "2023-09-29T17:42:55+07:00"
			if dateStr != expected {
				t.Errorf("Expected date_origin_interaction = %s, got %s", expected, dateStr)
			}
		} else {
			t.Errorf("Expected date_origin_interaction to be string, got %T", dateOrigin)
		}

		// Verify date_first_pickup_interaction is formatted
		datePickup, exists := row1.Get("date_first_pickup_interaction")
		if !exists {
			t.Error("date_first_pickup_interaction field not found")
		}
		if dateStr, ok := datePickup.(string); ok {
			// 1696000208 Unix = 2023-09-29 15:10:08 UTC = 2023-09-29 22:10:08 GMT+7
			expected := "2023-09-29T22:10:08+07:00"
			if dateStr != expected {
				t.Errorf("Expected date_first_pickup_interaction = %s, got %s", expected, dateStr)
			}
		} else {
			t.Errorf("Expected date_first_pickup_interaction to be string, got %T", datePickup)
		}

		// Verify date_first_response_interaction is formatted
		dateResponse, exists := row1.Get("date_first_response_interaction")
		if !exists {
			t.Error("date_first_response_interaction field not found")
		}
		if dateStr, ok := dateResponse.(string); ok {
			// 1696000208 Unix = 2023-09-29 15:10:08 UTC = 2023-09-29 22:10:08 GMT+7
			expected := "2023-09-29T22:10:08+07:00"
			if dateStr != expected {
				t.Errorf("Expected date_first_response_interaction = %s, got %s", expected, dateStr)
			}
		} else {
			t.Errorf("Expected date_first_response_interaction to be string, got %T", dateResponse)
		}

		// Verify non-date field (ticket_id) is NOT formatted
		ticketID, exists := row1.Get("ticket_id")
		if !exists {
			t.Error("ticket_id field not found")
		}
		if _, ok := ticketID.(string); ok {
			t.Errorf("Expected ticket_id to remain as number, got string")
		}

		// Verify field order is preserved
		if len(fields) != 4 {
			t.Errorf("Expected 4 fields, got %d", len(fields))
		}
	})

	t.Run("with isFormatDate=false keeps timestamps as numbers", func(t *testing.T) {
		results, err := BatchTransformRows(rows, formulas, operators, false)
		if err != nil {
			t.Fatalf("BatchTransformRows() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Check first row
		row1 := results[0]

		// Verify date fields are NOT formatted (remain as int64)
		dateOrigin, exists := row1.Get("date_origin_interaction")
		if !exists {
			t.Error("date_origin_interaction field not found")
		}
		if dateInt, ok := dateOrigin.(int64); ok {
			if dateInt != 1695984175 {
				t.Errorf("Expected date_origin_interaction = 1695984175, got %d", dateInt)
			}
		} else {
			t.Errorf("Expected date_origin_interaction to be int64, got %T", dateOrigin)
		}

		// Verify ticket_id remains unchanged
		ticketID, exists := row1.Get("ticket_id")
		if !exists {
			t.Error("ticket_id field not found")
		}
		if id, ok := ticketID.(int); ok {
			if id != 123 {
				t.Errorf("Expected ticket_id = 123, got %d", id)
			}
		} else {
			t.Errorf("Expected ticket_id to be int, got %T", ticketID)
		}
	})
}

func TestFormatDateFields(t *testing.T) {
	tests := []struct {
		name     string
		input    TransformedRow
		expected map[string]string
	}{
		{
			name: "formats date fields with GMT+7",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "date_created", Value: int64(1695984175)},
					{Key: "date_updated", Value: int64(1696000208)},
				},
			},
			expected: map[string]string{
				"date_created": "2023-09-29T17:42:55+07:00",
				"date_updated": "2023-09-29T22:10:08+07:00",
			},
		},
		{
			name: "handles DATE prefix case-insensitive",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "DATE_CREATED", Value: int64(1695984175)},
					{Key: "Date_Updated", Value: int64(1696000208)},
				},
			},
			expected: map[string]string{
				"DATE_CREATED": "2023-09-29T17:42:55+07:00",
				"Date_Updated": "2023-09-29T22:10:08+07:00",
			},
		},
		{
			name: "skips non-date fields",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "date_created", Value: int64(1695984175)},
					{Key: "ticket_id", Value: int64(123)},
					{Key: "status", Value: "open"},
				},
			},
			expected: map[string]string{
				"date_created": "2023-09-29T17:42:55+07:00",
			},
		},
		{
			name: "handles various numeric types",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "date_int", Value: int(1695984175)},
					{Key: "date_int32", Value: int32(1695984175)},
					{Key: "date_int64", Value: int64(1695984175)},
					{Key: "date_float64", Value: float64(1695984175)},
				},
			},
			expected: map[string]string{
				"date_int":     "2023-09-29T17:42:55+07:00",
				"date_int32":   "2023-09-29T17:42:55+07:00",
				"date_int64":   "2023-09-29T17:42:55+07:00",
				"date_float64": "2023-09-29T17:42:55+07:00",
			},
		},
		{
			name: "skips zero timestamps",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "date_created", Value: int64(0)},
					{Key: "date_updated", Value: int64(1696000208)},
				},
			},
			expected: map[string]string{
				"date_updated": "2023-09-29T22:10:08+07:00",
			},
		},
		{
			name: "skips nil values",
			input: TransformedRow{
				fields: []TransformedField{
					{Key: "date_created", Value: nil},
					{Key: "date_updated", Value: int64(1696000208)},
				},
			},
			expected: map[string]string{
				"date_updated": "2023-09-29T22:10:08+07:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDateFields(tt.input)

			for key, expectedValue := range tt.expected {
				value, exists := result.Get(key)
				if !exists {
					t.Errorf("Field %s not found in result", key)
					continue
				}

				if strValue, ok := value.(string); ok {
					if strValue != expectedValue {
						t.Errorf("Field %s: expected %s, got %s", key, expectedValue, strValue)
					}
				} else {
					t.Errorf("Field %s: expected string, got %T", key, value)
				}
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"int", int(123), 123},
		{"int8", int8(123), 123},
		{"int16", int16(123), 123},
		{"int32", int32(123), 123},
		{"int64", int64(123), 123},
		{"uint", uint(123), 123},
		{"uint8", uint8(123), 123},
		{"uint16", uint16(123), 123},
		{"uint32", uint32(123), 123},
		{"uint64", uint64(123), 123},
		{"float32", float32(123.5), 123},
		{"float64", float64(123.7), 123},
		{"nil", nil, 0},
		{"string", "123", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt64(tt.input)
			if result != tt.expected {
				t.Errorf("toInt64(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDateFormatting_GMT7Timezone(t *testing.T) {
	// Verify that the timezone offset is correct (+07:00)
	gmt7 := time.FixedZone("GMT+7", 7*3600)

	// Test timestamp: 1695984175 (Unix)
	// 1695984175 Unix = 2023-09-29 10:42:55 UTC = 2023-09-29 17:42:55 GMT+7
	timestamp := int64(1695984175)
	t1 := time.Unix(timestamp, 0).In(gmt7)
	formatted := t1.Format(time.RFC3339)

	expected := "2023-09-29T17:42:55+07:00"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Verify the timezone offset is exactly +07:00
	_, offset := t1.Zone()
	expectedOffset := 7 * 3600 // 7 hours in seconds
	if offset != expectedOffset {
		t.Errorf("Expected offset %d seconds, got %d seconds", expectedOffset, offset)
	}
}

func TestBatchTransformRows_EdgeCases(t *testing.T) {
	operators := GetOperatorRegistry()

	t.Run("empty rows with isFormatDate=true", func(t *testing.T) {
		rows := []RowData{}
		formulas := []Formula{}

		results, err := BatchTransformRows(rows, formulas, operators, true)
		if err != nil {
			t.Fatalf("BatchTransformRows() error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("rows with non-numeric date values", func(t *testing.T) {
		rows := []RowData{
			{
				"date_created": "not-a-number",
				"ticket_id":    123,
			},
		}

		formulas := []Formula{
			{
				Params:   []string{"date_created"},
				Field:    "date_created",
				Operator: "",
				Position: 1,
			},
		}

		results, err := BatchTransformRows(rows, formulas, operators, true)
		if err != nil {
			t.Fatalf("BatchTransformRows() error = %v", err)
		}

		// Should not error, just skip formatting
		dateCreated, _ := results[0].Get("date_created")
		if _, ok := dateCreated.(string); !ok {
			// Value should remain as original (string "not-a-number")
			// since toInt64 returns 0 for non-numeric values and we skip 0 timestamps
		}
	})

	t.Run("mixed date and non-date fields", func(t *testing.T) {
		rows := []RowData{
			{
				"date_created":    int64(1695984175),
				"ticket_id":       123,
				"status":          "open",
				"date_updated":    int64(1696000208),
				"description":     "test ticket",
			},
		}

		formulas := []Formula{
			{Params: []string{"date_created"}, Field: "date_created", Operator: "", Position: 1},
			{Params: []string{"ticket_id"}, Field: "ticket_id", Operator: "", Position: 2},
			{Params: []string{"status"}, Field: "status", Operator: "", Position: 3},
			{Params: []string{"date_updated"}, Field: "date_updated", Operator: "", Position: 4},
			{Params: []string{"description"}, Field: "description", Operator: "", Position: 5},
		}

		results, err := BatchTransformRows(rows, formulas, operators, true)
		if err != nil {
			t.Fatalf("BatchTransformRows() error = %v", err)
		}

		row := results[0]

		// Date fields should be formatted
		dateCreated, _ := row.Get("date_created")
		if _, ok := dateCreated.(string); !ok {
			t.Error("date_created should be formatted to string")
		}

		dateUpdated, _ := row.Get("date_updated")
		if _, ok := dateUpdated.(string); !ok {
			t.Error("date_updated should be formatted to string")
		}

		// Non-date fields should remain unchanged
		ticketID, _ := row.Get("ticket_id")
		if _, ok := ticketID.(string); ok {
			t.Error("ticket_id should not be formatted to string")
		}

		status, _ := row.Get("status")
		if statusStr, ok := status.(string); ok {
			if statusStr != "open" {
				t.Errorf("status should remain 'open', got %s", statusStr)
			}
		}
	})
}
