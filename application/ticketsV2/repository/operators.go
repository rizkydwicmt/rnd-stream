package repository

import (
	"stream/application/ticketsV2/domain"
	"stream/application/tickets"
)

// GetOperatorRegistry returns the operator registry
// This wraps the operators from the original tickets package for reuse
func GetOperatorRegistry() map[string]domain.OperatorFunc {
	// Get the original operator registry
	originalOps := tickets.GetOperatorRegistry()

	// Convert to domain.OperatorFunc type
	// Since the function signatures are identical, we can directly use them
	ops := make(map[string]domain.OperatorFunc, len(originalOps))
	for name, op := range originalOps {
		ops[name] = domain.OperatorFunc(op)
	}

	return ops
}
