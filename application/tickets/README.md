# Tickets Streaming API

API endpoint untuk streaming data dari tabel `tickets` dengan transformasi formulas secara real-time.

## Endpoint

```
POST /v1/tickets/stream
```

## Features

- ✅ Dynamic query building dengan parameterized queries (SQL injection safe)
- ✅ WHERE clause filtering dengan multiple conditions
- ✅ ORDER BY support
- ✅ LIMIT dan OFFSET untuk pagination
- ✅ Formula transformations dengan multiple operators
- ✅ HTTP chunked streaming untuk memory efficiency
- ✅ Batch processing (100 rows per batch)
- ✅ Buffer pooling untuk mengurangi GC pressure
- ✅ Context cancellation support
- ✅ Total count header (`X-Total-Count`)
- ✅ Comprehensive validation dan error handling

## Request Payload

```json
{
  "tableName": "tickets",
  "orderBy": ["id", "asc"],
  "limit": 100,
  "offset": 0,
  "where": [
    {"field": "status", "op": "=", "value": "open"},
    {"field": "created_at", "op": ">=", "value": "2025-01-01"}
  ],
  "formulas": [
    {
      "params": ["id"],
      "field": "ticket_id",
      "operator": "",
      "position": 2
    },
    {
      "params": ["id", "created_at"],
      "field": "ticket_id_masked",
      "operator": "ticketIdMasking",
      "position": 1
    }
  ]
}
```

### Payload Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tableName` | string | Yes | Table name (must be in whitelist: "tickets") |
| `orderBy` | array | No | Format: `["field_name", "asc|desc"]` |
| `limit` | int | Yes | Number of records to return (1-10000) |
| `offset` | int | No | Pagination offset (default: 0) |
| `where` | array | No | WHERE conditions (see below) |
| `formulas` | array | No | Transformation formulas (see below) |

### WHERE Clause

```json
{
  "field": "status",
  "op": "=",
  "value": "open"
}
```

**Supported Operators:**
- `=`, `!=`, `>`, `>=`, `<`, `<=`
- `LIKE`, `NOT LIKE`
- `IN`, `NOT IN` (value should be array)

### Formulas

```json
{
  "params": ["id", "created_at"],
  "field": "output_field_name",
  "operator": "ticketIdMasking",
  "position": 1
}
```

| Field | Type | Description |
|-------|------|-------------|
| `params` | array of strings | Input column names from database |
| `field` | string | Output field name in response |
| `operator` | string | Transformation function (see below) |
| `position` | int | Sort order for formula execution |

**Available Operators:**

| Operator | Description | Example Input | Example Output |
|----------|-------------|---------------|----------------|
| `` (empty) | Pass-through (no transformation) | `[12345]` | `12345` |
| `ticketIdMasking` | Mask ticket ID, show last 3 digits | `[12345, "2025-01-01"]` | `"TCK-*****345"` |
| `concat` | Concatenate all params with space | `["Hello", "World"]` | `"Hello World"` |
| `upper` | Convert to uppercase | `["hello"]` | `"HELLO"` |
| `lower` | Convert to lowercase | `["HELLO"]` | `"hello"` |
| `formatDate` | Format date (default: "2006-01-02") | `[time.Time]` | `"2025-01-15"` |

## Response

### Headers

```
Content-Type: application/json
X-Total-Count: 1234
Transfer-Encoding: chunked
```

### Body (Streaming JSON Array)

```json
[
  { "ticket_id_masked": "TCK-*****345", "ticket_id": 12345 },
  { "ticket_id_masked": "TCK-*****346", "ticket_id": 12346 },
  ...
]
```

Response adalah **valid streaming JSON array**. Data dikirim secara bertahap (chunked) dalam format:
- `[` - Array opening (sent in first chunk)
- `{},{},{}` - Objects separated by commas (sent in 32KB chunks)
- `]` - Array closing (sent in final chunk)

Client receives complete valid JSON array tanpa perlu manual parsing.

## Example cURL Request

```bash
curl -X POST http://localhost:8080/v1/tickets/stream \
  -H "Content-Type: application/json" \
  -d '{
    "tableName": "tickets",
    "orderBy": ["id", "asc"],
    "limit": 10,
    "offset": 0,
    "where": [
      {"field": "status", "op": "=", "value": "open"}
    ],
    "formulas": [
      {
        "params": ["id"],
        "field": "ticket_id",
        "operator": "",
        "position": 1
      },
      {
        "params": ["id", "created_at"],
        "field": "masked_id",
        "operator": "ticketIdMasking",
        "position": 2
      },
      {
        "params": ["status", "priority"],
        "field": "info",
        "operator": "concat",
        "position": 3
      }
    ]
  }'
```

## Performance Characteristics

### Memory Usage

- **Batch Size:** 100 rows per batch (configurable)
- **Buffer Pool:** Reuses 4KB JSON encoding buffers
- **Chunk Buffer:** Accumulates data up to 32KB before sending (reduces HTTP overhead)
- **Streaming:** Data sent in optimized chunks, not all buffered in memory
- **Peak Memory:** ~10-20MB for 100K rows with 10 columns

### Query Performance

- **COUNT Query:** O(n) on filtered rows - may be slow on large tables without indexes
- **SELECT Query:** Efficient with proper indexes on WHERE/ORDER BY columns
- **Batch Processing:** Constant memory regardless of result set size

### Recommendations

1. **Indexes:** Add indexes on frequently filtered/sorted columns
2. **LIMIT:** Use reasonable limits (< 10000) for better UX
3. **Offset:** For large offsets, consider cursor-based pagination
4. **COUNT Trade-off:** COUNT(*) can be expensive on large tables - consider caching or estimated counts
5. **Chunk Size:** 32KB chunks balance network efficiency and streaming responsiveness

## Security

### SQL Injection Protection

✅ All user inputs are **parameterized** - no string interpolation
✅ Table names validated against whitelist
✅ Column names validated for suspicious patterns
✅ Operators validated against whitelist
✅ Special characters blocked in identifiers

### Validation Rules

- Table name must be in whitelist (currently: "tickets")
- Limit: 1-10000
- Offset: >= 0
- OrderBy: exactly 2 elements `["field", "asc|desc"]`
- WHERE operators: must be in allowed list
- Formula operators: must be in allowed list
- No SQL keywords in field names (drop, exec, union, etc.)
- No special characters (`;`, `--`, `/*`, `*/`)

## Limitations

1. **Single Table Only:** Currently supports "tickets" table only (whitelist can be extended)
2. **No JOINs:** Only single table queries supported
3. **Simple WHERE Logic:** Multiple WHERE conditions use AND logic only (no OR)
4. **COUNT Performance:** COUNT(*) can be slow on very large tables
5. **Column Names:** Must match database schema exactly (case-sensitive on some DB engines)
6. **NULL Handling:** Uses `github.com/guregu/null/v5` for NULL values

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /v1/tickets/stream
       ▼
┌─────────────────────┐
│     Handler         │  - Bind & validate payload
│  (handler.go)       │  - Extract request context
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│     Service         │  - Sort formulas by position
│  (service.go)       │  - Generate SELECT columns
│                     │  - Execute COUNT query
│                     │  - Execute main query
│                     │  - Stream processing
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│   Repository        │  - Raw SQL execution
│  (repository.go)    │  - Batch row fetching
│                     │  - Stream rows via channel
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│   Query Builder     │  - Build SELECT query
│ (query_builder.go)  │  - Build COUNT query
│                     │  - Parameterize all inputs
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│     Mapper          │  - Generic row scanning
│   (mapper.go)       │  - Formula transformation
│                     │  - Operator execution
└─────────────────────┘
```

## Testing

### Run All Tests

```bash
# Unit tests
go test ./application/tickets/... -v

# Integration tests only
go test ./application/tickets/... -run Integration -v

# Benchmark
go test ./application/tickets/... -bench . -benchmem
```

### Test Coverage

```bash
go test ./application/tickets/... -cover
```

## Adding New Formula Operators

1. Add operator to `AllowedFormulaOperators` in `types.go`
2. Implement operator function in `operators.go`:
   ```go
   func myCustomOperator(params []interface{}) (interface{}, error) {
       // Implementation
       return result, nil
   }
   ```
3. Register in `GetOperatorRegistry()` in `operators.go`
4. Add tests in `operators_test.go`

## Error Responses

### Validation Error (400)

```json
{
  "code": 400,
  "message": "Invalid JSON payload",
  "error": "validation failed: limit must be between 1 and 10000"
}
```

### Server Error (500)

```json
{
  "code": 500,
  "message": "Stream failed",
  "error": "failed to execute query: ..."
}
```

## Future Enhancements

- [ ] Multi-table support with JOINs
- [ ] OR logic in WHERE clauses
- [ ] Aggregate functions (SUM, AVG, GROUP BY)
- [ ] Custom operator plugins
- [ ] Query result caching
- [ ] Cursor-based pagination
- [ ] GraphQL-style field selection
- [ ] WebSocket streaming
- [ ] Estimated counts for large tables

## License

Internal use only.
