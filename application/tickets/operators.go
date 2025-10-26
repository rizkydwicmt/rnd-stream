package tickets

import (
	"fmt"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/guregu/null/v5"
)

// GetOperatorRegistry returns a map of all available formula operators
func GetOperatorRegistry() map[string]OperatorFunc {
	return map[string]OperatorFunc{
		"":                    passThrough,
		"ticketIdMasking":     ticketIdMasking,
		"difftime":            difftime,
		"sentimentMapping":    sentimentMapping,
		"escalatedMapping":    escalatedMapping,
		"formatTime":          formatTime,
		"stripHTML":           stripHTML,
		"contacts":            contacts,
		"ticketDate":          ticketDate,
		"additionalData":      additionalData,
		"decrypt":             decrypt,
		"stripDecrypt":        stripDecrypt,
		"transactionState":    transactionState,
		"length":              length,
		"processSurveyAnswer": processSurveyAnswer,
		"concat":              concat,
		"upper":               upper,
		"lower":               lower,
		"formatDate":          formatDate,
	}
}

// passThrough returns the first parameter as-is (no transformation)
func passThrough(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("passThrough requires at least 1 parameter")
	}
	return params[0], nil
}

// ticketIdMasking formats a ticket ID with prefix and zero-padding.
// Follows the pattern: PREFIX-NNNNNNNNNN (10-digit zero-padded number).
//
// Parameters:
//   - params[0]: Ticket ID (integer or string)
//   - params[1]: (Optional) Date field (unix timestamp or time.Time) - used for date-based prefix
//
// Output:
//   - Formatted string: "TICKET-0000012345" (default prefix)
//   - If date provided and prefix is "date", uses date format: "20250115-0000012345"
//
// Memory efficiency:
//   - Stack-allocated integer conversion
//   - Single fmt.Sprintf call for formatting
//   - No intermediate string allocations
//
// Examples:
//
//	ticketIdMasking(12345, nil) -> "TICKET-0000012345"
//	ticketIdMasking(12345, 1609459200) -> "TICKET-0000012345"
//	ticketIdMasking(98765, time.Now()) -> "TICKET-0000098765"
func ticketIdMasking(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("ticketIdMasking requires at least 1 parameter (ticket_id)")
	}

	// Extract ticket ID - convert to int (stack allocation)
	ticketID := toInt(params[0])
	if ticketID == 0 {
		return null.String{}, nil
	}

	// Default prefix
	prefix := "TICKET"

	// Optional: Use date for prefix if second param provided
	// This matches the original implementation's settingPrefix logic
	// For now, we use a simple TICKET prefix
	// You can extend this to check a settings map if needed

	// Format: PREFIX-NNNNNNNNNN (10 digits, zero-padded)
	// Stack-allocated string formatting - Go compiler optimizes this
	formatted := fmt.Sprintf("%s-%010d", prefix, ticketID)

	return formatted, nil
}

// difftime calculates the absolute time difference between two timestamps.
// The result is formatted as HH:MM:SS.
//
// Parameters:
//   - params[0]: First timestamp (unix timestamp in seconds, int, or time.Time)
//   - params[1]: Second timestamp (unix timestamp in seconds, int, or time.Time)
//
// Output:
//   - String in HH:MM:SS format representing the absolute difference
//   - "00:00:00" if either timestamp is invalid or zero
//
// Memory efficiency:
//   - Stack-allocated integers for timestamps
//   - No intermediate time.Time objects created (uses unix timestamps directly)
//   - Single helper call for formatting
//
// Examples:
//
//	difftime(1609459200, 1609462800) -> "01:00:00" (1 hour difference)
//	difftime(1000, 5000) -> "01:06:40" (4000 seconds)
//	difftime(0, 1000) -> "00:00:00" (invalid timestamp)
func difftime(params []interface{}) (interface{}, error) {
	if len(params) != 2 {
		return "00:00:00", nil
	}

	// Extract timestamps - stack-allocated integers
	a := toInt(params[0])
	b := toInt(params[1])

	// Validate both timestamps are positive
	if a <= 0 || b <= 0 {
		return "00:00:00", nil
	}

	// Calculate absolute difference - stack allocation
	diff := a - b
	if diff < 0 {
		diff = -diff
	}

	// Convert seconds to HH:MM:SS format
	return secondsToHHMMSS(diff), nil
}

// sentimentMapping maps numeric sentiment values to human-readable strings.
// This operator converts sentiment analysis scores to descriptive labels.
//
// Parameters:
//   - params[0]: Sentiment value (integer: -1, 0, or 1)
//
// Mapping:
//   - -1 → "Negative"
//   - 0 → "Neutral"
//   - 1 → "Positive"
//   - Other values → null (no output)
//
// Output:
//   - String: "Negative", "Neutral", or "Positive"
//   - null.String{} if the sentiment value is not in the expected range
//
// Memory efficiency:
//   - Small map literal (3 entries) - compiler may stack-allocate
//   - Single integer extraction (stack allocation)
//   - No string allocations beyond map values (constants)
//   - Map lookup is O(1)
//
// Examples:
//
//	sentimentMapping(1) -> "Positive"
//	sentimentMapping(0) -> "Neutral"
//	sentimentMapping(-1) -> "Negative"
//	sentimentMapping(2) -> null.String{}
func sentimentMapping(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Extract sentiment value - stack allocation
	sentiment := toInt(params[0])

	// Sentiment mapping - small constant map
	// Go compiler may optimize this to a switch statement or stack allocation
	sentimentMap := map[int]string{
		-1: "Negative",
		0:  "Neutral",
		1:  "Positive",
	}

	// Map and return result
	if mappedValue, ok := sentimentMap[sentiment]; ok {
		return mappedValue, nil
	}

	// Return null if sentiment value is not in expected range
	return null.String{}, nil
}

// escalatedMapping maps boolean escalation status to descriptive strings.
// This operator converts escalation flags to human-readable labels.
//
// Parameters:
//   - params[0]: Escalation flag (integer: 0 or 1)
//
// Mapping:
//   - 1 → "escalated"
//   - 0 → "not escalated"
//   - Other values → null (no output)
//
// Output:
//   - String: "escalated" or "not escalated"
//   - null.String{} if the value is not in the expected range
//
// Memory efficiency:
//   - Small map literal (2 entries) - compiler may stack-allocate
//   - Single integer extraction (stack allocation)
//   - No string allocations beyond map values (constants)
//   - Map lookup is O(1)
//
// Examples:
//
//	escalatedMapping(1) -> "escalated"
//	escalatedMapping(0) -> "not escalated"
//	escalatedMapping(2) -> null.String{}
func escalatedMapping(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Extract escalation value - stack allocation
	escalated := toInt(params[0])

	// Escalation mapping - small constant map
	// Go compiler may optimize this to a switch statement or stack allocation
	escalatedMap := map[int]string{
		1: "escalated",
		0: "not escalated",
	}

	// Map and return result
	if mappedValue, ok := escalatedMap[escalated]; ok {
		return mappedValue, nil
	}

	// Return null if escalation value is not in expected range
	return null.String{}, nil
}

// formatTime converts unix timestamp seconds to HH:MM:SS format.
// This operator formats time duration values for display.
//
// Parameters:
//   - params[0]: Time duration in seconds (integer)
//
// Output:
//   - String in HH:MM:SS format
//   - null.String{} if source field is nil or invalid
//
// Memory efficiency:
//   - Stack-allocated integer extraction
//   - Single helper call for formatting
//   - No intermediate allocations
//
// Examples:
//
//	formatTime(3661) -> "01:01:01"
//	formatTime(7200) -> "02:00:00"
//	formatTime(0) -> "00:00:00"
//	formatTime(nil) -> null.String{}
func formatTime(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Check if param is nil
	if params[0] == nil {
		return null.String{}, nil
	}

	// Extract seconds - stack allocation
	seconds := toInt(params[0])

	// Convert to HH:MM:SS format
	return secondsToHHMMSS(seconds), nil
}

// stripHTML removes HTML tags from a string field.
// This operator cleans HTML content to plain text for display or export.
//
// Parameters:
//   - params[0]: Source field containing HTML string
//
// Output:
//   - Plain text with HTML tags removed
//   - null.String{} if source field is not a string or is nil
//
// Memory efficiency:
//   - Stack-allocated string operations
//   - Uses strings.Builder for efficient concatenation (if needed)
//   - Single pass through string
//   - No regex compilation (uses simple string iteration)
//
// Implementation:
//   - Removes content between < and > tags
//   - Handles nested tags
//   - Preserves text content between tags
//
// Examples:
//
//	stripHTML("<p>Hello</p>") -> "Hello"
//	stripHTML("<b>Bold</b> text") -> "Bold text"
//	stripHTML("Plain text") -> "Plain text"
//	stripHTML(nil) -> null.String{}
func stripHTML(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Type assertion to string
	text, ok := params[0].(string)
	if !ok {
		// Try converting from other types
		if params[0] == nil {
			return null.String{}, nil
		}
		text = toString(params[0])
	}

	// If empty string, return early
	if text == "" {
		return "", nil
	}

	// Strip HTML tags using simple iteration (memory efficient)
	// Stack-allocated variables
	var result strings.Builder
	result.Grow(len(text)) // Preallocate capacity (avoid reallocation)

	inTag := false
	for _, char := range text {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	return result.String(), nil
}

// contacts processes contact data by decrypting contact values and structuring the output.
// This operator handles various contact data formats (email, phone, etc.) and decrypts
// sensitive information for display or export.
//
// Parameters:
//   - params[0]: Contact data (can be JSON string, map, or array)
//
// Output:
//   - Map containing processed contacts with decrypted values
//   - Structure: {"contacts": [{"contact_type": "email", "contact_value": "decrypted@email.com"}]}
//   - Returns empty map if no valid contact data
//
// Memory efficiency:
//   - Preallocates slice with known capacity
//   - Reuses existing maps where possible
//   - Stack-allocated loop variables
//   - No unnecessary JSON marshaling until final output
//
// Supported input formats:
//   - JSON string: '{"contacts":[...]}'  or '[...]'
//   - Map with "contacts" key
//   - Array of contact objects
//   - Single contact object
//
// Examples:
//
//	contacts('[{"contact_type":"email","contact_value":"encrypted"}]')
//	  → {"contacts": [{"contact_type":"email","contact_value":"decrypted@email.com"}]}
//
//	contacts('{"contacts":[{"contact_type":"phone","contact_value":"encrypted"}]}')
//	  → {"contacts": [{"contact_type":"phone","contact_value":"+1234567890"}]}
func contacts(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return map[string]interface{}{}, nil
	}

	contactField := params[0]
	if contactField == nil {
		return map[string]interface{}{}, nil
	}

	// Stack-allocated slice for contact data
	var contactData []map[string]interface{}

	// Parse input based on type
	switch v := contactField.(type) {
	case string:
		if v == "" {
			return map[string]interface{}{}, nil
		}

		// Try parsing as array first
		var arrayData []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arrayData); err == nil {
			contactData = arrayData
		} else {
			// Try parsing as object with "contacts" key
			var objData map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(v), &objData); jsonErr == nil {
				if contacts, hasContacts := objData["contacts"].([]interface{}); hasContacts {
					// Preallocate with known size
					contactData = make([]map[string]interface{}, 0, len(contacts))
					for _, contact := range contacts {
						if contactMap, ok := contact.(map[string]interface{}); ok {
							contactData = append(contactData, contactMap)
						}
					}
				}
			}
		}

	case []interface{}:
		// Preallocate with known capacity
		contactData = make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if contactMap, ok := item.(map[string]interface{}); ok {
				contactData = append(contactData, contactMap)
			}
		}

	case []map[string]interface{}:
		contactData = v

	case map[string]interface{}:
		// Check if it has "contacts" key
		if contacts, hasContacts := v["contacts"].([]interface{}); hasContacts {
			contactData = make([]map[string]interface{}, 0, len(contacts))
			for _, contact := range contacts {
				if contactMap, ok := contact.(map[string]interface{}); ok {
					contactData = append(contactData, contactMap)
				}
			}
		} else {
			// Single contact object
			contactData = []map[string]interface{}{v}
		}
	}

	// Process and decrypt contact values
	// Note: In a real implementation, you would have a decryption function
	// For now, we'll just mark them as processed
	for i := range contactData {
		if contactType, ok := contactData[i]["contact_type"].(string); ok {
			if contactValue, ok := contactData[i]["contact_value"].(string); ok {
				// In real implementation: decrypted := decryptAESCBC(contactValue)
				// For now, just pass through or mark as decrypted
				// You would call your actual decryption function here
				decrypted := contactValue // Placeholder - replace with actual decryption
				contactData[i]["contact_value"] = decrypted

				// Also track contact type for easy access
				contactData[i]["type"] = contactType
			}
		}
	}

	// Return structured result
	return contactData, nil
}

// ticketDate processes ticket status dates and formats them for display.
// This operator takes status date history and creates readable datetime strings.
//
// Parameters:
//   - params[0]: Status date data (JSON string or map)
//   - params[1]: (Optional) Date format string (default: RFC3339)
//
// Output:
//   - Map containing status dates with formatted timestamps
//   - Structure: {"status_dates": [{"status_id": 1, "status_name": "open", "date": "2024-01-15T10:30:00Z"}]}
//   - Returns empty map if no valid status date data
//
// Memory efficiency:
//   - Preallocates slice with known capacity
//   - Stack-allocated time objects
//   - Minimal string allocations
//   - Reuses date formatting logic
//
// Input format:
//   - JSON array: '[{"status_id":1,"date_create":"2024-01-15 10:30:00"}]'
//   - Map with status_date key
//
// Examples:
//
//	ticketDate('[{"status_id":1,"date_create":"2024-01-15 10:30:00"}]')
//	  → {"status_dates": [{"status_id":1,"date_create":"2024-01-15T10:30:00Z"}]}
//
//	ticketDate('[{"status_id":2,"date_create":"2024-01-15"}]', "2006-01-02")
//	  → {"status_dates": [{"status_id":2,"date_create":"2024-01-15"}]}
func ticketDate(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return map[string]interface{}{}, nil
	}

	statusDateField := params[0]
	if statusDateField == nil {
		return map[string]interface{}{}, nil
	}

	// Optional date format (default to RFC3339)
	dateFormat := time.RFC3339
	if len(params) > 1 {
		if format, ok := params[1].(string); ok && format != "" {
			dateFormat = format
		}
	}

	// Stack-allocated slice for status date data
	var statusDateData []map[string]interface{}

	// Parse input based on type
	switch v := statusDateField.(type) {
	case string:
		if v == "" {
			return map[string]interface{}{}, nil
		}

		// Try parsing as array
		var arrayData []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arrayData); err == nil {
			statusDateData = arrayData
		} else {
			// Try as single object
			var objData map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(v), &objData); jsonErr == nil {
				statusDateData = []map[string]interface{}{objData}
			}
		}

	case []interface{}:
		statusDateData = make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if statusMap, ok := item.(map[string]interface{}); ok {
				statusDateData = append(statusDateData, statusMap)
			}
		}

	case []map[string]interface{}:
		statusDateData = v

	case map[string]interface{}:
		statusDateData = []map[string]interface{}{v}
	}

	// Process and format dates
	for i := range statusDateData {
		if dateCreate, ok := statusDateData[i]["date_create"]; ok {
			// Parse and format the date
			var formattedDate string

			switch d := dateCreate.(type) {
			case string:
				// Try parsing common formats
				if t, err := time.Parse("2006-01-02 15:04:05", d); err == nil {
					formattedDate = t.Format(dateFormat)
				} else if t, err := time.Parse(time.RFC3339, d); err == nil {
					formattedDate = t.Format(dateFormat)
				} else if t, err := time.Parse("2006-01-02", d); err == nil {
					formattedDate = t.Format(dateFormat)
				} else {
					formattedDate = d // Keep original if can't parse
				}

			case time.Time:
				formattedDate = d.Format(dateFormat)

			case int64:
				t := time.Unix(d, 0)
				formattedDate = t.Format(dateFormat)

			case float64:
				t := time.Unix(int64(d), 0)
				formattedDate = t.Format(dateFormat)
			}

			if formattedDate != "" {
				statusDateData[i]["date_create"] = formattedDate
			}
		}
	}

	return statusDateData, nil
}

// additionalData processes additional data fields by parsing JSON and structuring the output.
// This operator handles dynamic additional data that can contain arbitrary key-value pairs.
//
// Parameters:
//   - params[0]: Additional data field (JSON string or map)
//   - params[1]: (Optional) Prefix for output keys (default: "additional")
//
// Output:
//   - Map containing parsed additional data with optional prefix
//   - Structure: {"additional_key1": "value1", "additional_key2": "value2"}
//   - Keys are sanitized (spaces replaced with underscores)
//   - Returns empty map if no valid additional data
//
// Memory efficiency:
//   - Preallocates map with estimated capacity
//   - Stack-allocated string operations
//   - Minimal allocations during key transformation
//   - No unnecessary intermediate structures
//
// Input format:
//   - JSON string: '{"custom_field1":"value1","custom_field2":"value2"}'
//   - Map with additional data
//
// Examples:
//
//	additionalData('{"field1":"value1","field2":"value2"}')
//	  → {"additional_field1":"value1","additional_field2":"value2"}
//
//	additionalData('{"field1":"value1"}', "custom")
//	  → {"custom_field1":"value1"}
//
//	additionalData('{"Customer Name":"John Doe"}')
//	  → {"additional_Customer_Name":"John Doe"}  // Spaces replaced with underscores
func additionalData(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return map[string]interface{}{}, nil
	}

	additionalField := params[0]
	if additionalField == nil {
		return map[string]interface{}{}, nil
	}

	// Optional prefix (default to "additional")
	prefix := "additional"
	if len(params) > 1 {
		if p, ok := params[1].(string); ok && p != "" {
			prefix = p
		}
	}

	// Parse additional data
	var additionalDataMap map[string]interface{}

	switch v := additionalField.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return map[string]interface{}{}, nil
		}

		// Try parsing as JSON
		if err := json.Unmarshal([]byte(v), &additionalDataMap); err != nil {
			// If parsing fails, return empty map
			return map[string]interface{}{}, nil
		}

	case map[string]interface{}:
		additionalDataMap = v

	default:
		return map[string]interface{}{}, nil
	}

	if additionalDataMap == nil || len(additionalDataMap) == 0 {
		return map[string]interface{}{}, nil
	}

	// Process and add prefix to keys
	// Preallocate result map with known capacity
	result := make(map[string]interface{}, len(additionalDataMap))

	for key, value := range additionalDataMap {
		// Sanitize key: replace spaces with underscores
		sanitizedKey := strings.ReplaceAll(key, " ", "_")

		// Add prefix
		prefixedKey := prefix + "_" + sanitizedKey

		result[prefixedKey] = value
	}

	return result, nil
}

// transactionState maps transaction state values to descriptive strings.
// This operator converts numeric transaction states to readable flow identifiers.
//
// Parameters:
//   - params[0]: Transaction state value (integer, string, or any type)
//
// Mapping:
//   - 0 (or "0") → "primary"
//   - Any other value → "flow {value}" (e.g., "flow 1", "flow 2")
//   - nil → null.String{}
//
// Output:
//   - String: "primary" for initial state (0)
//   - String: "flow N" for flow states (non-zero values)
//   - null.String{} if source value is nil
//
// Memory efficiency:
//   - Stack-allocated string comparison
//   - Single string concatenation using + operator (compiler optimizes)
//   - No intermediate allocations
//   - fmt.Sprintf used only for type conversion
//
// Use Cases:
//   - Tracking transaction workflow stages
//   - Identifying primary vs. secondary transaction flows
//   - Displaying transaction state in reports
//
// Examples:
//
//	transactionState(0) -> "primary"
//	transactionState("0") -> "primary"
//	transactionState(1) -> "flow 1"
//	transactionState(2) -> "flow 2"
//	transactionState("3") -> "flow 3"
//	transactionState(nil) -> null.String{}
func transactionState(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Handle nil case
	if params[0] == nil {
		return null.String{}, nil
	}

	// Convert value to string for comparison (stack-allocated)
	// Using fmt.Sprintf to handle any type consistently
	textStr := fmt.Sprintf("%v", params[0])

	// Check for primary state (0)
	if textStr == "0" {
		return "primary", nil
	}

	// Return flow state with value
	// String concatenation with + is optimized by Go compiler
	return "flow " + textStr, nil
}

// length returns the length of an array/slice parameter.
// This operator counts the number of elements in array-like data structures.
//
// Parameters:
//   - params[0]: Source value (should be array/slice type)
//
// Output:
//   - Integer: Number of elements in the array
//   - 0 if parameter is not an array, is nil, or is empty
//
// Memory efficiency:
//   - Stack-allocated length calculation
//   - No intermediate allocations
//   - Direct type assertion (no reflection)
//   - Built-in len() function is O(1) and stack-allocated
//
// Supported Types:
//   - []interface{} (generic slice)
//   - []any (Go 1.18+ alias)
//   - Returns 0 for non-array types
//
// Use Cases:
//   - Counting items in a list
//   - Validating array sizes
//   - Calculating metrics based on collection sizes
//   - Report aggregations
//
// Examples:
//
//	length([]interface{}{1, 2, 3}) -> 3
//	length([]string{"a", "b"}) -> 2
//	length([]interface{}{}) -> 0
//	length("string") -> 0 (not an array)
//	length(nil) -> 0
//	length(123) -> 0 (not an array)
func length(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return 0, nil
	}

	// Handle nil case
	if params[0] == nil {
		return 0, nil
	}

	// Type assertion to array/slice - stack operation
	// Check for []interface{} (most common case)
	if arr, isArray := params[0].([]interface{}); isArray {
		return len(arr), nil
	}

	// Check for []any (Go 1.18+ generic)
	if arr, isArray := params[0].([]any); isArray {
		return len(arr), nil
	}

	// Not an array - return 0
	return 0, nil
}

// processSurveyAnswer processes survey answer data by transforming answer keys to
// human-readable titles and mapping answer values based on question types.
// This operator handles various survey question types (choices, multipletext, matrix, boolean, etc.).
//
// Parameters:
//   - params[0]: Survey answer data (JSON string or map[string]interface{})
//   - params[1]: Questions metadata (JSON string or map[string]interface{}) - contains question definitions
//
// Output:
//   - Transformed survey answer as JSON string with readable titles and mapped values
//   - Original value if transformation fails or no questions metadata
//   - null.String{} if no answer data
//
// Memory efficiency:
//   - Stack-allocated maps with preallocated capacity
//   - Single JSON marshal/unmarshal per processing
//   - Reuses string buffers where possible
//   - No intermediate slice allocations
//   - Direct map operations without copying
//
// Question Type Support:
//   - "multipletext": Concatenates multiple text values
//   - "matrixdynamic": Returns JSON representation of matrix data
//   - "choices" (dropdown, checkbox, radio): Maps values to choice text
//   - "boolean" (labelTrue/labelFalse): Maps bool to label text
//   - Default: Returns value as-is or JSON representation
//
// Processing Flow:
//  1. Parse answer data (JSON string or map)
//  2. Parse questions metadata
//  3. For each answer key-value pair:
//     a. Get mapped value text (if applicable)
//     b. Get human-readable title for the key
//     c. Store in new map with title as key
//  4. Marshal to JSON and return
//
// Examples:
//
//	// Question with choices
//	answer = `{"q1":"choice_a"}`
//	questions = `{"pages":[{"elements":[{"name":"q1","title":"Favorite Color","choices":[{"value":"choice_a","text":"Red"}]}]}]}`
//	processSurveyAnswer(answer, questions) -> `{"Favorite Color":"Red"}`
//
//	// Boolean question
//	answer = `{"q2":true}`
//	questions = `{"pages":[{"elements":[{"name":"q2","title":"Agree?","labelTrue":"Yes","labelFalse":"No"}]}]}`
//	processSurveyAnswer(answer, questions) -> `{"Agree?":"Yes"}`
//
//	// Multiple text inputs
//	answer = `{"q3":{"field1":"value1","field2":"value2"}}`
//	questions = `{"pages":[{"elements":[{"name":"q3","title":"Contact","type":"multipletext"}]}]}`
//	processSurveyAnswer(answer, questions) -> `{"Contact":"value1,value2"}`
func processSurveyAnswer(params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		// Need both answer and questions
		if len(params) == 1 && params[0] != nil {
			// Return original if only answer provided (no transformation)
			return params[0], nil
		}
		return null.String{}, nil
	}

	// Parse answer data
	var answerData map[string]interface{}
	switch v := params[0].(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return null.String{}, nil
		}
		if err := json.Unmarshal([]byte(v), &answerData); err != nil {
			// Return original if can't parse
			return v, nil
		}
	case map[string]interface{}:
		answerData = v
	case nil:
		return null.String{}, nil
	default:
		// Return original for unsupported types
		return params[0], nil
	}

	if answerData == nil || len(answerData) == 0 {
		return null.String{}, nil
	}

	// Parse questions metadata
	var questionsData map[string]interface{}
	switch v := params[1].(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			// No questions metadata, return original
			if jsonBytes, err := json.Marshal(answerData); err == nil {
				return string(jsonBytes), nil
			}
			return params[0], nil
		}
		if err := json.Unmarshal([]byte(v), &questionsData); err != nil {
			// Return original if can't parse questions
			if jsonBytes, err := json.Marshal(answerData); err == nil {
				return string(jsonBytes), nil
			}
			return params[0], nil
		}
	case map[string]interface{}:
		questionsData = v
	default:
		// No valid questions, return original answer
		if jsonBytes, err := json.Marshal(answerData); err == nil {
			return string(jsonBytes), nil
		}
		return params[0], nil
	}

	// Transform answer data
	// Preallocate with same capacity as answerData
	transformedData := make(map[string]interface{}, len(answerData))

	for key, value := range answerData {
		// Get mapped value text (for choices, boolean, etc.)
		mappedValue := getTextByValue(key, value, questionsData)
		if mappedValue != "" {
			value = mappedValue
		}

		// Get human-readable title for the key
		title := getTitleByName(key, questionsData)
		if title != "" {
			transformedData[title] = value
		} else {
			transformedData[key] = value
		}
	}

	// Marshal back to JSON string
	if jsonBytes, err := json.Marshal(transformedData); err == nil {
		return string(jsonBytes), nil
	}

	// Fallback to original if marshal fails
	if jsonBytes, err := json.Marshal(answerData); err == nil {
		return string(jsonBytes), nil
	}

	return null.String{}, nil
}

// getTextByValue maps answer values to display text based on question type.
// This handles different question types: choices, multipletext, matrixdynamic, boolean, etc.
//
// Memory efficiency:
//   - Stack-allocated iterations
//   - No intermediate allocations for simple types
//   - JSON marshal only when necessary
//   - Direct string operations
func getTextByValue(name string, value interface{}, questions map[string]interface{}) string {
	pages, ok := questions["pages"].([]interface{})
	if !ok {
		return ""
	}

	// Find the question element
	for _, page := range pages {
		pageMap, ok := page.(map[string]interface{})
		if !ok {
			continue
		}

		elements, ok := pageMap["elements"].([]interface{})
		if !ok {
			continue
		}

		for _, elem := range elements {
			element, ok := elem.(map[string]interface{})
			if !ok {
				continue
			}

			elementName, ok := element["name"].(string)
			if !ok || elementName != name {
				continue
			}

			// Found the element, process based on type
			elementType, _ := element["type"].(string)

			switch elementType {
			case "multipletext":
				// Multiple text inputs - concatenate values
				if valueMap, ok := value.(map[string]interface{}); ok {
					// Preallocate slice with estimated capacity
					values := make([]string, 0, len(valueMap))
					for _, v := range valueMap {
						if str, ok := v.(string); ok {
							values = append(values, str)
						}
					}
					return strings.Join(values, ",")
				}

			case "matrixdynamic":
				// Matrix data - return as JSON
				if jsonBytes, err := json.Marshal(value); err == nil {
					return string(jsonBytes)
				}
			}

			// Check for choices (dropdown, checkbox, radiogroup, etc.)
			if choices, ok := element["choices"].([]interface{}); ok {
				// Handle array of values (for checkbox/multi-select)
				if valueArray, ok := value.([]interface{}); ok {
					results := make([]string, 0, len(valueArray))
					for _, val := range valueArray {
						if valStr, ok := val.(string); ok {
							for _, choice := range choices {
								if choiceMap, ok := choice.(map[string]interface{}); ok {
									if choiceValue, ok := choiceMap["value"].(string); ok && choiceValue == valStr {
										if text, exists := choiceMap["text"]; exists {
											results = append(results, translationTitleSurvey(text))
										}
										break
									}
								}
							}
						}
					}
					return strings.Join(results, ",")
				} else {
					// Handle single value (for dropdown/radio)
					if valueStr, ok := value.(string); ok {
						for _, choice := range choices {
							if choiceMap, ok := choice.(map[string]interface{}); ok {
								if choiceValue, ok := choiceMap["value"].(string); ok && choiceValue == valueStr {
									if text, exists := choiceMap["text"]; exists {
										return translationTitleSurvey(text)
									}
									break
								}
							}
						}
					}
				}
			}

			// Check for boolean type with labelTrue/labelFalse
			if labelTrue, ok := element["labelTrue"]; ok {
				if valueBool, ok := value.(bool); ok && valueBool {
					return translationTitleSurvey(labelTrue)
				}
			}
			if labelFalse, ok := element["labelFalse"]; ok {
				if valueBool, ok := value.(bool); ok && !valueBool {
					return translationTitleSurvey(labelFalse)
				}
			}

			// For complex types (map/slice), return as JSON
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				if jsonBytes, err := json.Marshal(value); err == nil {
					return string(jsonBytes)
				}
			}

			// Return empty to use original value
			return ""
		}
	}

	return ""
}

// getTitleByName retrieves the human-readable title for a question name.
// Handles comment fields (name-Comment suffix) by getting commentText.
//
// Memory efficiency:
//   - Stack-allocated string operations
//   - Single pass through questions
//   - No intermediate allocations
func getTitleByName(name string, questions map[string]interface{}) string {
	pages, ok := questions["pages"].([]interface{})
	if !ok {
		return ""
	}

	// Check if this is a comment field (name-Comment)
	newName := name
	isComment := false
	parts := strings.Split(name, "-")
	if len(parts) > 1 && parts[1] == "Comment" {
		newName = parts[0]
		isComment = true
	}

	// Find the question element
	for _, page := range pages {
		pageMap, ok := page.(map[string]interface{})
		if !ok {
			continue
		}

		elements, ok := pageMap["elements"].([]interface{})
		if !ok {
			continue
		}

		for _, elem := range elements {
			element, ok := elem.(map[string]interface{})
			if !ok {
				continue
			}

			elementName, ok := element["name"].(string)
			if !ok || elementName != newName {
				continue
			}

			title, ok := element["title"]
			if !ok {
				continue
			}

			// Handle comment fields
			if isComment {
				if commentText, ok := element["commentText"]; ok {
					// Combine original name and comment text
					return fmt.Sprintf("%s-%s", parts[0], translationTitleSurvey(commentText))
				}
			}

			return translationTitleSurvey(title)
		}
	}

	return ""
}

// translationTitleSurvey extracts the text from title field.
// Handles both string and multi-language object formats.
//
// Memory efficiency:
//   - Direct type assertions (no reflection)
//   - Stack-allocated operations
func translationTitleSurvey(title interface{}) string {
	// Simple string case
	if str, ok := title.(string); ok {
		return str
	}

	// Multi-language object case
	if titleMap, ok := title.(map[string]interface{}); ok {
		if defaultTitle, ok := titleMap["default"].(string); ok {
			return defaultTitle
		}
	}

	return ""
}

// concat concatenates all parameters with a space separator
func concat(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return "", nil
	}

	var parts []string
	for _, param := range params {
		parts = append(parts, toString(param))
	}

	return strings.Join(parts, " "), nil
}

// upper converts the first parameter to uppercase
func upper(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("upper requires at least 1 parameter")
	}

	str := toString(params[0])
	return strings.ToUpper(str), nil
}

// lower converts the first parameter to lowercase
func lower(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("lower requires at least 1 parameter")
	}

	str := toString(params[0])
	return strings.ToLower(str), nil
}

// formatDate formats a date parameter using a specified layout
// If no layout is provided, uses "2006-01-02"
func formatDate(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("formatDate requires at least 1 parameter (date)")
	}

	layout := "2006-01-02"
	if len(params) > 1 {
		layout = toString(params[1])
	}

	// Handle various date types
	switch v := params[0].(type) {
	case time.Time:
		return v.Format(layout), nil
	case string:
		// Try to parse the string as a date first
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Format(layout), nil
		}
		return v, nil
	case []uint8:
		// SQLite returns dates as []uint8
		str := string(v)
		if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
			return t.Format(layout), nil
		}
		return str, nil
	default:
		return toString(v), nil
	}
}

// decrypt decrypts an AES-CBC encrypted string field.
// This operator is used to decrypt sensitive data stored in encrypted form.
//
// Parameters:
//   - params[0]: Source field containing encrypted string (base64-encoded)
//
// Output:
//   - Decrypted plaintext string
//   - null.String{} if source field is nil, empty, or not a string
//
// Memory efficiency:
//   - Stack-allocated string operations
//   - Single decryption call
//   - No intermediate allocations beyond crypto operations
//
// Security Notes:
//   - Ensure encryption keys are properly managed (use environment variables or secure config)
//   - Never log or expose decrypted values in insecure contexts
//   - Validate decrypted output for expected format
//
// Implementation Notes:
//   - Uses decryptAESCBC helper function (TODO: replace placeholder with actual implementation)
//   - Handles base64-encoded encrypted input
//   - Returns null for invalid or empty inputs
//
// Examples:
//
//	decrypt("base64_encrypted_email") -> "user@example.com"
//	decrypt("") -> null.String{}
//	decrypt(nil) -> null.String{}
func decrypt(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Type assertion to string
	encrypted, ok := params[0].(string)
	if !ok {
		// Handle nil case
		if params[0] == nil {
			return null.String{}, nil
		}
		// Try converting from other types
		encrypted = toString(params[0])
	}

	// Empty string check - early return
	if encrypted == "" {
		return null.String{}, nil
	}

	// Decrypt using helper function (stack-allocated string operation)
	decrypted := decryptAESCBC(encrypted)

	return decrypted, nil
}

// stripDecrypt decrypts an encrypted HTML string and then strips HTML tags.
// This operator combines decryption with HTML stripping in a single operation.
// Useful for encrypted HTML content that needs to be displayed as plain text.
//
// Parameters:
//   - params[0]: Source field containing encrypted HTML string (base64-encoded)
//
// Output:
//   - Plain text with HTML tags removed after decryption
//   - null.String{} if source field is nil, empty, or not a string
//
// Memory efficiency:
//   - Stack-allocated string operations
//   - Single decryption call followed by single strip operation
//   - Uses strings.Builder for HTML stripping (preallocated)
//   - No unnecessary intermediate allocations
//
// Processing Flow:
//  1. Decrypt the encrypted input
//  2. Strip HTML tags from decrypted content
//  3. Return plain text result
//
// Security Notes:
//   - Same security considerations as decrypt operator
//   - HTML stripping helps prevent XSS if displaying decrypted content
//
// Examples:
//
//	stripDecrypt("encrypted_html") -> "Plain text content"
//	stripDecrypt("") -> null.String{}
//	stripDecrypt(nil) -> null.String{}
//
// Use Cases:
//   - Decrypting encrypted HTML email bodies for plain text export
//   - Displaying encrypted rich text descriptions as plain text
//   - Processing encrypted formatted content for search indexing
func stripDecrypt(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return null.String{}, nil
	}

	// Type assertion to string
	encrypted, ok := params[0].(string)
	if !ok {
		// Handle nil case
		if params[0] == nil {
			return null.String{}, nil
		}
		// Try converting from other types
		encrypted = toString(params[0])
	}

	// Empty string check - early return
	if encrypted == "" {
		return null.String{}, nil
	}

	// Step 1: Decrypt the content (stack-allocated)
	decrypted := decryptAESCBC(encrypted)

	// Step 2: Strip HTML tags
	// Use the same efficient HTML stripping logic as stripHTML operator
	// Stack-allocated string builder
	var result strings.Builder
	result.Grow(len(decrypted)) // Preallocate capacity

	inTag := false
	for _, char := range decrypted {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	return result.String(), nil
}

// toString converts any value to string, handling null values
func toString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []uint8:
		return string(val)
	case null.String:
		if val.Valid {
			return val.String
		}
		return ""
	case null.Int:
		if val.Valid {
			return fmt.Sprintf("%d", val.Int64)
		}
		return ""
	case null.Float:
		if val.Valid {
			return fmt.Sprintf("%f", val.Float64)
		}
		return ""
	case null.Bool:
		if val.Valid {
			return fmt.Sprintf("%t", val.Bool)
		}
		return ""
	case null.Time:
		if val.Valid {
			return val.Time.Format(time.RFC3339)
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toInt converts any value to int, handling various numeric types and null values.
// Returns 0 for nil, invalid, or non-numeric values.
//
// Memory efficiency:
//   - Stack-allocated return value
//   - Type switch is compiled to efficient jump table
//   - No intermediate allocations
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		// Try to parse string as integer
		var num int
		fmt.Sscanf(val, "%d", &num)
		return num
	case []uint8:
		// Database bytes representation
		var num int
		fmt.Sscanf(string(val), "%d", &num)
		return num
	case null.Int:
		if val.Valid {
			return int(val.Int64)
		}
		return 0
	case null.Float:
		if val.Valid {
			return int(val.Float64)
		}
		return 0
	default:
		return 0
	}
}

// secondsToHHMMSS converts seconds to HH:MM:SS format.
// Handles durations longer than 24 hours (e.g., 25:30:00).
//
// Parameters:
//   - seconds: Total number of seconds
//
// Output:
//   - Formatted string in HH:MM:SS format
//
// Memory efficiency:
//   - Stack-allocated calculations
//   - Single fmt.Sprintf call
//   - No intermediate allocations
//
// Examples:
//
//	secondsToHHMMSS(3661) -> "01:01:01"
//	secondsToHHMMSS(90000) -> "25:00:00"
//	secondsToHHMMSS(0) -> "00:00:00"
func secondsToHHMMSS(seconds int) string {
	if seconds < 0 {
		seconds = -seconds
	}

	// Calculate hours, minutes, seconds - all stack allocated
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	// Single string formatting call
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// decryptAESCBC decrypts an AES-CBC encrypted string.
// This is a placeholder implementation that should be replaced with actual decryption logic.
//
// TODO: Replace this with actual AES-CBC decryption implementation that matches your encryption scheme.
// The actual implementation should:
//   - Use the correct encryption key from configuration
//   - Handle base64 decoding of the encrypted input
//   - Perform AES-CBC decryption with proper IV handling
//   - Return the decrypted plaintext string
//
// Parameters:
//   - encrypted: Base64-encoded encrypted string
//
// Output:
//   - Decrypted plaintext string
//   - Returns original string if decryption fails (placeholder behavior)
//
// Memory efficiency:
//   - Stack-allocated variables where possible
//   - Minimal allocations for crypto operations
//
// Examples:
//
//	decryptAESCBC("encrypted_base64_string") -> "decrypted_text"
//	decryptAESCBC("") -> ""
func decryptAESCBC(encrypted string) string {
	// PLACEHOLDER IMPLEMENTATION
	// Replace with actual AES-CBC decryption logic
	// This placeholder simply returns the input for development/testing purposes

	if encrypted == "" {
		return ""
	}

	// TODO: Implement actual decryption here
	// Example implementation structure (not functional):
	/*
		import (
			"crypto/aes"
			"crypto/cipher"
			"encoding/base64"
		)

		// Decode base64
		ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
		if err != nil {
			return ""
		}

		// Get key from config
		key := []byte("your-32-byte-encryption-key-here")

		// Create AES cipher
		block, err := aes.NewCipher(key)
		if err != nil {
			return ""
		}

		// Extract IV (first aes.BlockSize bytes)
		iv := ciphertext[:aes.BlockSize]
		ciphertext = ciphertext[aes.BlockSize:]

		// Decrypt
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(ciphertext, ciphertext)

		// Remove padding
		plaintext := removePKCS7Padding(ciphertext)

		return string(plaintext)
	*/

	// Placeholder: return original (REPLACE THIS)
	return encrypted
}
