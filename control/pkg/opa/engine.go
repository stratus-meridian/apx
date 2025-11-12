package opa

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
)

// Engine wraps the OPA Rego engine for policy evaluation.
// It provides a simplified interface for loading and evaluating policies.
type Engine struct {
	query        rego.PreparedEvalQuery
	policyString string
	queryString  string
}

// NewEngine creates a new OPA engine instance with the given policy and query.
// The policy parameter should contain the Rego policy source code.
// The query parameter should contain the query to evaluate (e.g., "data.example.allow").
//
// Example:
//
//	policy := `
//	  package example
//	  allow {
//	    input.method == "GET"
//	  }
//	`
//	engine, err := NewEngine(ctx, policy, "data.example.allow")
//
// Returns an error if the policy fails to compile or the query is invalid.
func NewEngine(ctx context.Context, policy string, query string) (*Engine, error) {
	if policy == "" {
		return nil, fmt.Errorf("policy cannot be empty")
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Create a new Rego object with the policy and query
	r := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", policy),
	)

	// Prepare the query for evaluation
	preparedQuery, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare policy for evaluation: %w", err)
	}

	return &Engine{
		query:        preparedQuery,
		policyString: policy,
		queryString:  query,
	}, nil
}

// Eval evaluates the policy against the provided input data.
// The input parameter should be a map or struct containing the data to evaluate.
//
// Returns true if the policy evaluation succeeds and returns a truthy value,
// false if it returns a falsy value, and an error if evaluation fails.
//
// Example:
//
//	input := map[string]interface{}{
//	  "method": "GET",
//	  "path": "/api/users",
//	}
//	allowed, err := engine.Eval(ctx, input)
func Eval(ctx context.Context, input interface{}) (bool, error) {
	return false, fmt.Errorf("not implemented: use Engine.Eval instead")
}

// Eval evaluates the policy against the provided input data.
// The input parameter should be a map or struct containing the data to evaluate.
//
// Returns true if the policy evaluation succeeds and returns a truthy value,
// false if it returns a falsy value or undefined, and an error if evaluation fails.
func (e *Engine) Eval(ctx context.Context, input interface{}) (bool, error) {
	if e == nil {
		return false, fmt.Errorf("engine is nil")
	}

	// Evaluate the query with the provided input
	results, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Check if we got any results
	// In OPA, if a rule doesn't match, it returns undefined (no results)
	// This should be treated as false/denied
	if len(results) == 0 {
		return false, nil
	}

	// Check if the result has any expressions
	if len(results[0].Expressions) == 0 {
		return false, nil
	}

	// Get the first expression value
	value := results[0].Expressions[0].Value

	// Convert to boolean
	// Handle both boolean and undefined values
	allowed, ok := value.(bool)
	if !ok {
		// If the value is not a boolean, treat it as false
		// This can happen with undefined or other non-boolean values
		return false, nil
	}

	return allowed, nil
}

// Policy returns the policy string used by this engine.
func (e *Engine) Policy() string {
	if e == nil {
		return ""
	}
	return e.policyString
}

// Query returns the query string used by this engine.
func (e *Engine) Query() string {
	if e == nil {
		return ""
	}
	return e.queryString
}
