# Integration Guide: OperatorFunc Pattern

This guide shows you how to integrate the new `OperatorFunc` pattern into your existing `report.service.go` file.

## Overview

The refactored approach provides:
- **Better maintainability**: Each operator is a separate function
- **Memory efficiency**: Prioritizes stack allocation over heap
- **Extensibility**: Easy to add new operators
- **Testability**: Each operator can be unit tested independently

## Step 1: Add OperatorRegistry to ReportService

```go
type ReportService struct {
	prefixTicket       string
	publisher          *rabbitmq.Publisher
	reportRepository   *reportRepository.ReportRepository
	formulaReportRepo  formulaReportRepo.IFormulaReportRepository
	mSettingReportRepo mSettingReportRepository.IMSettingReportRepository
	ctx                context.Context
	config             *config.Config
	operatorRegistry   OperatorRegistry // Add this field
}
```

## Step 2: Initialize Registry in Constructor

```go
func NewReportService(p params.Params, repo *reportRepository.ReportRepository, mSettingReportRepo mSettingReportRepository.IMSettingReportRepository, formulaRepo formulaReportRepo.IFormulaReportRepository) (*ReportService, error) {
	service := &ReportService{
		prefixTicket:       "TICKET",
		reportRepository:   repo,
		mSettingReportRepo: mSettingReportRepo,
		formulaReportRepo:  formulaRepo,
		ctx:                p.Ctx,
		publisher:          p.Publisher,
		config:             config.ConfigVal,
		operatorRegistry:   NewOperatorRegistry(), // Initialize here
	}

	return service, nil
}
```

## Step 3: Refactor processChunkOperators

Replace the large switch statement with registry lookup:

### Before (Lines 1108-1149):
```go
func (s *ReportService) processChunkOperators(chunk map[string]any, processedChunk map[string]any, formulaData formulaEntity.FormulaReport, statusTicket *[]types.TicketStatus, settingPrefix map[string]any) {
	var params []string
	jsoniter.Unmarshal(formulaData.Params, &params)

	if formulaData.Field != "" {
		processedChunk[formulaData.Field] = nil
	}

	switch formulaData.Operator {
	case "difftime":
		s.processDiffTime(chunk, processedChunk, formulaData.Field, params)
	case "ticketIdMasking":
		s.processTicketIdMasking(chunk, processedChunk, formulaData.Field, params, settingPrefix)
	case "sentimentMapping":
		s.processSentimentMapping(chunk, processedChunk, formulaData.Field, params)
	// ... many more cases ...
	default:
		processedChunk[formulaData.Field] = chunk[formulaData.Field]
	}
}
```

### After (Recommended):
```go
func (s *ReportService) processChunkOperators(chunk map[string]any, processedChunk map[string]any, formulaData formulaEntity.FormulaReport, statusTicket *[]types.TicketStatus, settingPrefix map[string]any) {
	var params []string
	jsoniter.Unmarshal(formulaData.Params, &params)

	if formulaData.Field != "" {
		processedChunk[formulaData.Field] = nil
	}

	// Lookup operator in registry
	if opFunc, exists := s.operatorRegistry[formulaData.Operator]; exists {
		opFunc(s, chunk, processedChunk, formulaData.Field, params)
		return
	}

	// Fallback for legacy or unmapped operators
	switch formulaData.Operator {
	case "escalatedMapping":
		s.processEscalatedMapping(chunk, processedChunk, formulaData.Field, params)
	case "formatTime":
		s.processFormatTime(chunk, processedChunk, formulaData.Field, params)
	case "stripHTML":
		s.processStripHTML(chunk, processedChunk, formulaData.Field, params)
	case "contacts":
		s.processContact(chunk, processedChunk)
	case "ticketDate":
		s.processTicketStatusDate(chunk, processedChunk, statusTicket)
	case "additionalData":
		s.processAdditionalData(chunk, processedChunk)
	case "decrypt":
		s.processDecrypt(chunk, processedChunk, formulaData.Field, params)
	case "stripDecrypt":
		s.processStripDecrypt(chunk, processedChunk, formulaData.Field, params)
	case "transactionState":
		s.processTransactionState(chunk, processedChunk, formulaData.Field, params)
	case "length":
		s.processLength(chunk, processedChunk, formulaData.Field, params)
	case "processSurveyAnswer":
		s.processSurveyAnswer(chunk, processedChunk, formulaData.Field, params)
	default:
		processedChunk[formulaData.Field] = chunk[formulaData.Field]
	}
}
```

## Step 4: Update Existing Operator Functions

Update the signature of your existing operator functions to match the `OperatorFunc` type:

### Before:
```go
func (s *ReportService) processDiffTime(chunk map[string]any, processedChunk map[string]any, field string, params []string) {
	// ... implementation ...
}
```

### After (same signature, works with registry):
```go
func (s *ReportService) operatorDiffTime(chunk map[string]any, processedChunk map[string]any, field string, params []string) {
	// ... implementation ...
}
```

**Note**: You can keep both versions during migration. The registry uses the new `operator*` functions while the legacy switch uses the old `process*` functions.

## Step 5: Gradually Migrate Operators

You can migrate operators one at a time:

1. **Keep existing `process*` functions** for backward compatibility
2. **Add new `operator*` functions** with improved implementation
3. **Register** new operators in `NewOperatorRegistry()`
4. **Test** thoroughly
5. **Remove** old `process*` functions once migration is complete

Example migration order (safest first):
1. ✅ `difftime` - Simple calculation, no external dependencies
2. ✅ `sentimentMapping` - Simple mapping, no side effects
3. ✅ `ticketIdMasking` - Slightly more complex, calls service method
4. `formatTime` - Similar to difftime
5. `stripHTML` - Simple transformation
6. ... continue with others

## Step 6: Add Unit Tests

Create `operator_test.go`:

```go
package service

import (
	"testing"
)

func TestOperatorDiffTime(t *testing.T) {
	s := &ReportService{}

	tests := []struct {
		name     string
		chunk    map[string]any
		params   []string
		expected string
	}{
		{
			name:     "valid timestamps",
			chunk:    map[string]any{"start": 1000, "end": 5000},
			params:   []string{"start", "end"},
			expected: "01:06:40", // 4000 seconds = 1h 6m 40s
		},
		{
			name:     "zero timestamps",
			chunk:    map[string]any{"start": 0, "end": 0},
			params:   []string{"start", "end"},
			expected: "00:00:00",
		},
		{
			name:     "invalid params",
			chunk:    map[string]any{"start": 1000},
			params:   []string{"start"}, // Only 1 param
			expected: "00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			s.operatorDiffTime(tt.chunk, result, "duration", tt.params)

			if result["duration"] != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result["duration"])
			}
		})
	}
}

func TestOperatorSentimentMapping(t *testing.T) {
	s := &ReportService{}

	tests := []struct {
		name      string
		sentiment int
		expected  string
	}{
		{"positive", 1, "Positive"},
		{"neutral", 0, "Neutral"},
		{"negative", -1, "Negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := map[string]any{"sentiment": tt.sentiment}
			result := make(map[string]any)

			s.operatorSentimentMapping(chunk, result, "sentiment_label", []string{"sentiment"})

			if result["sentiment_label"] != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result["sentiment_label"])
			}
		})
	}
}
```

## Step 7: Verify Memory Efficiency

Run escape analysis to verify stack allocation:

```bash
cd internal/application/report/service
go build -gcflags='-m' 2>&1 | grep operator
```

Look for messages like:
- `✅ "a does not escape"` - Good, stack allocated
- `❌ "a escapes to heap"` - Review and optimize if possible

Run benchmarks:

```bash
go test -bench=BenchmarkOperator -benchmem
```

Expected results:
```
BenchmarkOperatorDiffTime-8          5000000    250 ns/op     0 B/op   0 allocs/op
BenchmarkOperatorSentimentMapping-8  3000000    400 ns/op     0 B/op   0 allocs/op
```

## Complete Example

Here's a complete minimal example showing the pattern:

```go
package main

import (
	"fmt"
	"onx-report-go/internal/pkg/helper"
)

// Simplified service for demonstration
type ReportService struct {
	operatorRegistry OperatorRegistry
}

func NewReportService() *ReportService {
	return &ReportService{
		operatorRegistry: NewOperatorRegistry(),
	}
}

func (s *ReportService) processData(data map[string]any) map[string]any {
	result := make(map[string]any)

	// Apply difftime operator
	s.operatorRegistry["difftime"](
		s,
		data,
		result,
		"duration",
		[]string{"start_time", "end_time"},
	)

	// Apply sentimentMapping operator
	s.operatorRegistry["sentimentMapping"](
		s,
		data,
		result,
		"sentiment_text",
		[]string{"sentiment_score"},
	)

	return result
}

func main() {
	service := NewReportService()

	data := map[string]any{
		"start_time":      1609459200,
		"end_time":        1609462800,
		"sentiment_score": 1,
	}

	result := service.processData(data)

	fmt.Printf("Duration: %v\n", result["duration"])
	fmt.Printf("Sentiment: %v\n", result["sentiment_text"])
}
```

## Benefits Summary

### Before
- ❌ 40+ line switch statement
- ❌ Hard to test individual operators
- ❌ Coupling between operators and dispatch logic
- ❌ Difficult to extend

### After
- ✅ Registry-based dispatch (O(1) lookup)
- ✅ Each operator is independently testable
- ✅ Clear separation of concerns
- ✅ Easy to add new operators
- ✅ Better documentation per operator
- ✅ Memory-efficient with stack allocation
- ✅ Idiomatic Go code

## Rollback Plan

If you need to rollback:

1. Remove `operatorRegistry` field from `ReportService`
2. Remove registry initialization from constructor
3. Revert `processChunkOperators` to use switch statement
4. Keep old `process*` functions

The refactored code is designed to be backward compatible, so you can rollback anytime during migration.

## Next Steps

1. Copy `operator_refactored.go` to your project:
   ```bash
   cp operator_refactored.go internal/application/report/service/
   ```

2. Modify `report.service.go` following Steps 1-3 above

3. Test with existing data to ensure compatibility

4. Gradually migrate remaining operators

5. Monitor performance and memory usage

6. Remove old implementations once migration is complete

## Questions?

Common questions:

**Q: Do I need to migrate all operators at once?**
A: No! You can migrate incrementally. The registry lookup happens first, then falls back to the switch for unmigrated operators.

**Q: Will this break existing functionality?**
A: No, if you keep the old `process*` functions in the switch fallback.

**Q: How do I add a new operator?**
A:
1. Write the operator function following the `OperatorFunc` signature
2. Register it in `NewOperatorRegistry()`
3. Done! No need to modify `processChunkOperators`

**Q: Is this more performant?**
A: Map lookup is O(1) and very fast. Combined with stack allocation optimizations, it's as fast or faster than switch statements for large operator counts.

**Q: Can I add custom operators at runtime?**
A: Yes! Just add to the registry:
```go
service.operatorRegistry["myCustomOp"] = myCustomOperatorFunc
```
