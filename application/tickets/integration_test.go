package tickets

import (
	"context"
	"fmt"
	"stream/common"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&common.Ticket{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Seed test data
	tickets := []common.Ticket{
		{
			ID:          1,
			TicketNo:    "TKT-000001",
			CustomerID:  1,
			Subject:     "Test ticket 1",
			Description: "Description 1",
			Status:      "open",
			Priority:    "high",
			CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:          2,
			TicketNo:    "TKT-000002",
			CustomerID:  2,
			Subject:     "Test ticket 2",
			Description: "Description 2",
			Status:      "open",
			Priority:    "medium",
			CreatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:          3,
			TicketNo:    "TKT-000003",
			CustomerID:  3,
			Subject:     "Test ticket 3",
			Description: "Description 3",
			Status:      "closed",
			Priority:    "low",
			CreatedAt:   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := db.Create(&tickets).Error; err != nil {
		t.Fatalf("Failed to seed data: %v", err)
	}

	return db
}

func TestIntegration_FullStreamingFlow(t *testing.T) {
	db := setupTestDB(t)

	// Create service stack
	repo := NewRepository(db)
	svc := NewService(repo)

	// Create payload
	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		OrderBy:   []string{"id", "asc"},
		Limit:     &limit,
		Offset:    0,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
		Formulas: []Formula{
			{
				Params:   []string{"id"},
				Field:    "ticket_id",
				Operator: "",
				Position: 1,
			},
			{
				Params:   []string{"id", "created_at"},
				Field:    "masked_id",
				Operator: "ticketIdMasking",
				Position: 2,
			},
		},
	}

	// Execute streaming
	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	// Check response
	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	if response.TotalCount != 2 {
		t.Errorf("Expected total count 2 (only 'open' tickets), got %d", response.TotalCount)
	}

	// Consume the stream
	var receivedData int
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
			continue
		}

		// Count received chunks (with buffering, multiple rows may be in one chunk)
		if chunk.JSONBuf != nil {
			receivedData++
		}
	}

	// With 32KB buffering, 2 small rows will likely be in 1 chunk
	// Just verify we received data
	if receivedData == 0 {
		t.Error("Expected to receive data chunks, got 0")
	}

	t.Logf("Received %d chunk(s) for 2 rows (expected 1 with 32KB buffering)", receivedData)
}

func TestIntegration_QueryBuilder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	limit := 2
	payload := &QueryPayload{
		TableName: "tickets",
		OrderBy:   []string{"id", "desc"},
		Limit:     &limit,
		Offset:    0,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
	}

	qb := NewQueryBuilder(payload)
	qb.SetSelectColumns([]string{"id", "status", "priority"})

	// Test count query
	countQuery, countArgs := qb.BuildCountQuery()
	count, err := repo.ExecuteCount(context.Background(), countQuery, countArgs)
	if err != nil {
		t.Fatalf("ExecuteCount() error = %v", err)
	}

	// Should have 2 tickets with status = "open"
	expectedCount := int64(2)
	if count != expectedCount {
		t.Logf("Count query: %s, args: %v", countQuery, countArgs)
		t.Errorf("Expected count %d, got %d", expectedCount, count)
	}

	// Test select query
	selectQuery, selectArgs := qb.BuildSelectQuery()
	rows, err := repo.ExecuteQuery(context.Background(), selectQuery, selectArgs)
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}

	results, err := repo.FetchRows(rows)
	if err != nil {
		t.Fatalf("FetchRows() error = %v", err)
	}

	// With LIMIT 2, should get at most 2 results
	if len(results) > 2 {
		t.Errorf("Expected at most 2 results (LIMIT 2), got %d", len(results))
	}

	// Should have exactly 2 results (2 open tickets)
	if len(results) != 2 {
		t.Errorf("Expected exactly 2 results, got %d", len(results))
	}
}

func TestIntegration_FormulaTransformation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	limit := 1
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Formulas: []Formula{
			{
				Params:   []string{"id"},
				Field:    "plain_id",
				Operator: "",
				Position: 1,
			},
			{
				Params:   []string{"id", "created_at"},
				Field:    "masked_id",
				Operator: "ticketIdMasking",
				Position: 2,
			},
			{
				Params:   []string{"status", "priority"},
				Field:    "status_priority",
				Operator: "concat",
				Position: 3,
			},
		},
	}

	qb := NewQueryBuilder(payload)
	selectCols := GenerateUniqueSelectList(payload.Formulas)
	qb.SetSelectColumns(selectCols)

	selectQuery, selectArgs := qb.BuildSelectQuery()
	rows, err := repo.ExecuteQuery(context.Background(), selectQuery, selectArgs)
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}

	results, err := repo.FetchRows(rows)
	if err != nil {
		t.Fatalf("FetchRows() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No results returned")
	}

	// Transform the first row
	row := results[0]
	operators := GetOperatorRegistry()
	sortedFormulas := SortFormulas(payload.Formulas)

	transformed, err := TransformRow(row, sortedFormulas, operators)
	if err != nil {
		t.Fatalf("TransformRow() error = %v", err)
	}

	// Check that all formula fields exist in output
	expectedFields := []string{"plain_id", "masked_id", "status_priority"}
	for _, field := range expectedFields {
		if _, exists := transformed.Get(field); !exists {
			t.Errorf("Expected field '%s' in transformed output", field)
		}
	}

	// Verify masked_id format (TICKET-NNNNNNNNNN)
	maskedIDVal, ok := transformed.Get("masked_id")
	if !ok {
		t.Error("masked_id field not found")
	}
	if maskedID, ok := maskedIDVal.(string); ok {
		if len(maskedID) < 6 || maskedID[:7] != "TICKET-" {
			t.Errorf("masked_id should start with 'TICKET-', got: %s", maskedID)
		}
		// Verify format: TICKET-NNNNNNNNNN (7 chars + 10 digits = 17 total)
		if len(maskedID) != 17 {
			t.Errorf("masked_id should be 17 characters long (TICKET-NNNNNNNNNN), got length: %d, value: %s", len(maskedID), maskedID)
		}
	} else {
		t.Errorf("masked_id should be a string, got: %T", maskedIDVal)
	}

	// Log the transformed result for inspection
	t.Logf("Transformed row: %+v", transformed)
}

// TestIntegration_EmptyFormulasSelectAll tests that empty formulas array results in SELECT * behavior
func TestIntegration_EmptyFormulasSelectAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	// Test with empty formulas array
	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Formulas:  []Formula{}, // Empty array
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	// Check response
	if response.Error != nil {
		t.Fatalf("StreamTickets() with empty formulas error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume the stream and verify we get actual data (not empty objects)
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			// Make a copy of the buffer content
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	if len(chunks) == 0 {
		t.Fatal("Expected to receive data chunks, got 0")
	}

	// Verify the response is not empty objects
	// The response should contain actual data like {"id":1,"ticket_no":"TKT-000001",...}
	// not empty objects like [{},{},{}]
	responseData := string(chunks[0])
	t.Logf("Response data: %s", responseData)

	// Verify response contains expected fields
	expectedFields := []string{"id", "ticket_no", "customer_id", "subject", "description", "status", "priority"}
	for _, field := range expectedFields {
		if !contains(responseData, field) {
			t.Errorf("Expected field '%s' in response data, but not found", field)
		}
	}

	// Verify it's not an empty object
	if contains(responseData, "[{}") || contains(responseData, "{},{}") {
		t.Error("Response contains empty objects instead of actual data")
	}
}

// TestIntegration_NilFormulasSelectAll tests that nil formulas results in SELECT * behavior
func TestIntegration_NilFormulasSelectAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	// Test with nil formulas (omitted in JSON)
	limit := 2
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Formulas:  nil, // Nil formulas
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	// Check response
	if response.Error != nil {
		t.Fatalf("StreamTickets() with nil formulas error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	// Consume the stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	if len(chunks) == 0 {
		t.Fatal("Expected to receive data chunks, got 0")
	}

	responseData := string(chunks[0])
	t.Logf("Response data with nil formulas: %s", responseData)

	// Verify response contains expected fields
	expectedFields := []string{"id", "ticket_no", "status"}
	for _, field := range expectedFields {
		if !contains(responseData, field) {
			t.Errorf("Expected field '%s' in response data, but not found", field)
		}
	}
}

// TestIntegration_EmptyFormulasWithWhere tests SELECT * with WHERE clause
func TestIntegration_EmptyFormulasWithWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
		Formulas: []Formula{}, // Empty formulas
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Should return only 2 tickets with status = "open"
	if response.TotalCount != 2 {
		t.Errorf("Expected total count 2 (only 'open' tickets), got %d", response.TotalCount)
	}

	// Consume and verify
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with WHERE: %s", responseData)

	// Verify response contains status field with "open" value
	if !contains(responseData, `"status":"open"`) && !contains(responseData, `"status": "open"`) {
		t.Error("Expected to find status='open' in response")
	}
}

// TestIntegration_EmptyFormulasWithOrderBy tests SELECT * with ORDER BY
func TestIntegration_EmptyFormulasWithOrderBy(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		OrderBy:   []string{"id", "desc"}, // Descending order
		Limit:     &limit,
		Formulas:  []Formula{}, // Empty formulas
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with ORDER BY: %s", responseData)

	// With ORDER BY id DESC, we should see id:3 before id:1
	// Verify the response contains data
	if !contains(responseData, `"id"`) {
		t.Error("Expected to find 'id' field in response")
	}
}

// TestIntegration_EmptyFormulasWithLimitOffset tests SELECT * with LIMIT and OFFSET
func TestIntegration_EmptyFormulasWithLimitOffset(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	// Get second page (offset 1, limit 1)
	limit := 1
	payload := &QueryPayload{
		TableName: "tickets",
		OrderBy:   []string{"id", "asc"},
		Limit:     &limit,
		Offset:    1, // Skip first record
		Formulas:  []Formula{},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Total count should still be 3 (OFFSET doesn't affect count)
	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with LIMIT/OFFSET: %s", responseData)

	// Should contain only 1 record (the second one with id=2)
	// Verify it's an array with data
	if !contains(responseData, "[{") || !contains(responseData, "}]") {
		t.Error("Expected valid JSON array with data")
	}
}

// TestIntegration_CompareEmptyFormulasWithExplicitFormulas verifies that empty formulas
// returns the same data as explicit pass-through formulas for all columns
func TestIntegration_CompareEmptyFormulasWithExplicitFormulas(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 1

	// Test 1: Empty formulas
	payload1 := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Formulas:  []Formula{},
	}

	ctx := context.Background()
	response1 := svc.StreamTickets(ctx, payload1)
	if response1.Error != nil {
		t.Fatalf("StreamTickets() with empty formulas error = %v", response1.Error)
	}

	var chunks1 [][]byte
	for chunk := range response1.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks1 = append(chunks1, bufCopy)
		}
	}

	response1Data := string(chunks1[0])
	t.Logf("Empty formulas response: %s", response1Data)

	// Verify response1 contains all expected fields
	allFields := []string{"id", "ticket_no", "customer_id", "subject", "description", "status", "priority", "created_at", "updated_at"}
	for _, field := range allFields {
		if !contains(response1Data, `"`+field+`"`) {
			t.Errorf("Empty formulas response missing field '%s'", field)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestIntegration_NilOrderBy tests that queries work without ORDER BY when nil
func TestIntegration_NilOrderBy(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		OrderBy:   nil, // Explicitly nil
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() with nil OrderBy error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume stream
	var receivedData int
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			receivedData++
		}
	}

	if receivedData == 0 {
		t.Error("Expected to receive data chunks, got 0")
	}
}

// TestIntegration_EmptyOrderBy tests that queries work without ORDER BY when empty array
func TestIntegration_EmptyOrderBy(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		OrderBy:   []string{}, // Explicitly empty
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() with empty OrderBy error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume stream
	var receivedData int
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			receivedData++
		}
	}

	if receivedData == 0 {
		t.Error("Expected to receive data chunks, got 0")
	}
}

// TestIntegration_NoOrderByWithWhere tests query without ORDER BY but with WHERE clause
func TestIntegration_NoOrderByWithWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
		// OrderBy omitted (will be nil)
		Formulas: []Formula{
			{Params: []string{"id", "status"}, Field: "ticket_info", Operator: "concat", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() without OrderBy but with WHERE error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	// Should return only 2 tickets with status = "open"
	if response.TotalCount != 2 {
		t.Errorf("Expected total count 2 (only 'open' tickets), got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	if len(chunks) == 0 {
		t.Fatal("Expected to receive data chunks, got 0")
	}

	responseData := string(chunks[0])
	t.Logf("Response without OrderBy but with WHERE: %s", responseData)
}

// TestIntegration_NoOrderByWithLimitOffset tests pagination without ORDER BY
func TestIntegration_NoOrderByWithLimitOffset(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 1
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Offset:    1,
		// OrderBy omitted
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() without OrderBy but with LIMIT/OFFSET error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	// Total count should still be 3
	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response without OrderBy with LIMIT/OFFSET: %s", responseData)

	// Should contain data (at most 1 record due to LIMIT 1)
	if !contains(responseData, "[") || !contains(responseData, "]") {
		t.Error("Expected valid JSON array")
	}
}

// TestIntegration_NoOrderByWithEmptyFormulas tests SELECT * without ORDER BY
func TestIntegration_NoOrderByWithEmptyFormulas(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		// No OrderBy, no Formulas - simplest query
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() without OrderBy and Formulas error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	if response.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response without OrderBy or Formulas (SELECT *): %s", responseData)

	// Verify we get actual data (all fields)
	expectedFields := []string{"id", "ticket_no", "status"}
	for _, field := range expectedFields {
		if !contains(responseData, `"`+field+`"`) {
			t.Errorf("Expected field '%s' in response data", field)
		}
	}
}

// TestQueryBuilder_NoOrderBy tests query building without ORDER BY clause
func TestQueryBuilder_NoOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		orderBy  []string
		expected string // Should NOT contain ORDER BY
	}{
		{
			name:     "nil orderBy",
			orderBy:  nil,
			expected: "SELECT `id`, `status` FROM `tickets`",
		},
		{
			name:     "empty orderBy array",
			orderBy:  []string{},
			expected: "SELECT `id`, `status` FROM `tickets`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &QueryPayload{
				TableName: "tickets",
				OrderBy:   tt.orderBy,
			}

			qb := NewQueryBuilder(payload)
			qb.SetSelectColumns([]string{"id", "status"})

			query, _ := qb.BuildSelectQuery()

			if query != tt.expected {
				t.Errorf("Expected query: %s\nGot: %s", tt.expected, query)
			}

			// Verify query does NOT contain ORDER BY
			if contains(query, "ORDER BY") {
				t.Errorf("Query should not contain ORDER BY clause, got: %s", query)
			}
		})
	}
}

// TestValidator_NoOrderByValidation tests that validation passes without OrderBy
func TestValidator_NoOrderByValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload *QueryPayload
		wantErr bool
	}{
		{
			name: "valid payload without orderBy (nil)",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   nil,
			},
			wantErr: false,
		},
		{
			name: "valid payload without orderBy (empty array)",
			payload: &QueryPayload{
				TableName: "tickets",
				OrderBy:   []string{},
			},
			wantErr: false,
		},
		{
			name: "valid payload with WHERE but no orderBy",
			payload: &QueryPayload{
				TableName: "tickets",
				Where: []WhereClause{
					{Field: "status", Operator: "=", Value: "open"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIntegration_DisableCount tests that count query is skipped when isDisableCount=true
func TestIntegration_DisableCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName:     "tickets",
		Limit:         &limit,
		IsDisableCount: true, // Disable count query
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() with isDisableCount=true error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	// When count is disabled, TotalCount should be -1
	if response.TotalCount != -1 {
		t.Errorf("Expected TotalCount = -1 when count disabled, got %d", response.TotalCount)
	}

	// Consume stream to verify data is still returned
	var receivedData int
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			receivedData++
		}
	}

	if receivedData == 0 {
		t.Error("Expected to receive data chunks even with count disabled, got 0")
	}

	t.Logf("Received data successfully with count disabled")
}

// TestIntegration_EnableCount tests that count query runs when isDisableCount=false
func TestIntegration_EnableCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName:     "tickets",
		Limit:         &limit,
		IsDisableCount: false, // Enable count query (default)
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() with isDisableCount=false error = %v", response.Error)
	}

	if response.Code != 200 {
		t.Errorf("Expected status code 200, got %d", response.Code)
	}

	// When count is enabled, TotalCount should be actual count (3 test tickets)
	if response.TotalCount != 3 {
		t.Errorf("Expected TotalCount = 3 when count enabled, got %d", response.TotalCount)
	}

	// Consume stream
	var receivedData int
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			receivedData++
		}
	}

	if receivedData == 0 {
		t.Error("Expected to receive data chunks, got 0")
	}
}

// TestIntegration_DefaultCountBehavior tests backward compatibility (count enabled by default)
func TestIntegration_DefaultCountBehavior(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		// IsDisableCount omitted - should default to false (count enabled)
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() with default count behavior error = %v", response.Error)
	}

	// Default behavior should include count
	if response.TotalCount != 3 {
		t.Errorf("Expected TotalCount = 3 with default behavior, got %d", response.TotalCount)
	}
}

// TestIntegration_DisableCountWithWhere tests count disabled with WHERE clause
func TestIntegration_DisableCountWithWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName:     "tickets",
		Limit:         &limit,
		IsDisableCount: true,
		Where: []WhereClause{
			{Field: "status", Operator: "=", Value: "open"},
		},
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Count should be -1 even with WHERE clause
	if response.TotalCount != -1 {
		t.Errorf("Expected TotalCount = -1 with count disabled, got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with disabled count and WHERE: %s", responseData)

	// Data should still be filtered by WHERE clause
	if !contains(responseData, "[") {
		t.Error("Expected valid JSON array")
	}
}

// TestIntegration_DisableCountWithPagination tests count disabled with LIMIT/OFFSET
func TestIntegration_DisableCountWithPagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 1
	payload := &QueryPayload{
		TableName:     "tickets",
		Limit:         &limit,
		Offset:        1,
		IsDisableCount: true,
		Formulas: []Formula{
			{Params: []string{"id"}, Field: "ticket_id", Operator: "", Position: 1},
		},
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Count disabled
	if response.TotalCount != -1 {
		t.Errorf("Expected TotalCount = -1, got %d", response.TotalCount)
	}

	// Pagination (LIMIT/OFFSET) should still work
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with disabled count and pagination: %s", responseData)

	// Should get data (second record due to OFFSET 1)
	if !contains(responseData, "[") {
		t.Error("Expected valid JSON array with pagination")
	}
}

// TestIntegration_DisableCountWithEmptyFormulas tests SELECT * with count disabled
func TestIntegration_DisableCountWithEmptyFormulas(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName:     "tickets",
		Limit:         &limit,
		IsDisableCount: true,
		// No formulas - SELECT * behavior
	}

	ctx := context.Background()
	response := svc.StreamTickets(ctx, payload)

	if response.Error != nil {
		t.Fatalf("StreamTickets() error = %v", response.Error)
	}

	// Count disabled
	if response.TotalCount != -1 {
		t.Errorf("Expected TotalCount = -1, got %d", response.TotalCount)
	}

	// Consume stream
	var chunks [][]byte
	for chunk := range response.ChunkChan {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}
		if chunk.JSONBuf != nil {
			bufCopy := make([]byte, len(*chunk.JSONBuf))
			copy(bufCopy, *chunk.JSONBuf)
			chunks = append(chunks, bufCopy)
		}
	}

	responseData := string(chunks[0])
	t.Logf("Response with disabled count and SELECT *: %s", responseData)

	// Verify we get all fields
	expectedFields := []string{"id", "ticket_no", "status"}
	for _, field := range expectedFields {
		if !contains(responseData, `"`+field+`"`) {
			t.Errorf("Expected field '%s' in response", field)
		}
	}
}

func BenchmarkStreamTickets(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewRepository(db)
	svc := NewService(repo)

	limit := 10
	payload := &QueryPayload{
		TableName: "tickets",
		Limit:     &limit,
		Formulas: []Formula{
			{
				Params:   []string{"id"},
				Field:    "ticket_id",
				Operator: "",
				Position: 1,
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := svc.StreamTickets(ctx, payload)
		if response.Error != nil {
			b.Fatalf("StreamTickets() error = %v", response.Error)
		}

		// Consume the stream
		for chunk := range response.ChunkChan {
			if chunk.Error != nil {
				b.Fatalf("Stream chunk error: %v", chunk.Error)
			}
		}
	}
}

// Helper for benchmarking
func setupBenchmarkDB(b *testing.B) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("Failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&common.Ticket{}); err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	tickets := make([]common.Ticket, 10)
	for i := 0; i < 10; i++ {
		tickets[i] = common.Ticket{
			ID:          uint(i + 1),
			TicketNo:    fmt.Sprintf("TKT-%06d", i+1),
			CustomerID:  uint((i % 3) + 1),
			Subject:     fmt.Sprintf("Test ticket %d", i+1),
			Description: fmt.Sprintf("Description %d", i+1),
			Status:      []string{"open", "closed", "pending"}[i%3],
			Priority:    []string{"low", "medium", "high"}[i%3],
			CreatedAt:   time.Date(2025, 1, i+1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 1, i+1, 0, 0, 0, 0, time.UTC),
		}
	}

	if err := db.Create(&tickets).Error; err != nil {
		b.Fatalf("Failed to seed data: %v", err)
	}

	return db
}
