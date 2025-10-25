#!/bin/bash

# Ensure server is running on localhost:8080

BASE_URL="http://localhost:8080"

echo "================================"
echo "Testing Tickets Streaming API"
echo "================================"
echo ""

# Test 1: Basic streaming request
echo "Test 1: Basic streaming with formulas"
echo "--------------------------------------"
curl -X POST "$BASE_URL/v1/tickets/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tableName": "tickets",
    "orderBy": ["id", "asc"],
    "limit": 5,
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
      }
    ]
  }' | jq '.'

echo ""
echo ""

# Test 2: All operators
echo "Test 2: All formula operators"
echo "------------------------------"
curl -X POST "$BASE_URL/v1/tickets/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tableName": "tickets",
    "limit": 3,
    "formulas": [
      {
        "params": ["id"],
        "field": "id",
        "operator": "",
        "position": 1
      },
      {
        "params": ["id", "created_at"],
        "field": "masked",
        "operator": "ticketIdMasking",
        "position": 2
      },
      {
        "params": ["ticket_no", "subject"],
        "field": "full_info",
        "operator": "concat",
        "position": 3
      },
      {
        "params": ["status"],
        "field": "status_upper",
        "operator": "upper",
        "position": 4
      },
      {
        "params": ["priority"],
        "field": "priority_lower",
        "operator": "lower",
        "position": 5
      }
    ]
  }' | jq '.'

echo ""
echo ""

# Test 3: Validation error
echo "Test 3: Validation error (invalid table)"
echo "----------------------------------------"
curl -X POST "$BASE_URL/v1/tickets/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tableName": "invalid_table",
    "limit": 10
  }' | jq '.'

echo ""
echo ""

# Test 4: Large limit with offset
echo "Test 4: Pagination (LIMIT 5 OFFSET 10)"
echo "---------------------------------------"
curl -X POST "$BASE_URL/v1/tickets/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tableName": "tickets",
    "orderBy": ["id", "asc"],
    "limit": 5,
    "offset": 10,
    "formulas": [
      {
        "params": ["id"],
        "field": "id",
        "operator": "",
        "position": 1
      },
      {
        "params": ["ticket_no"],
        "field": "ticket_no",
        "operator": "",
        "position": 2
      }
    ]
  }' | jq '.'

echo ""
echo ""
echo "================================"
echo "All tests completed!"
echo "================================"
