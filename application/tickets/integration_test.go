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
