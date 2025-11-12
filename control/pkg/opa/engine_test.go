package opa

import (
	"context"
	"testing"
)

func TestNewEngine(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		policy      string
		query       string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid policy and query",
			policy: `
				package example
				allow {
					input.method == "GET"
				}
			`,
			query:   "data.example.allow",
			wantErr: false,
		},
		{
			name:        "empty policy",
			policy:      "",
			query:       "data.example.allow",
			wantErr:     true,
			errContains: "policy cannot be empty",
		},
		{
			name: "empty query",
			policy: `
				package example
				allow = true
			`,
			query:       "",
			wantErr:     true,
			errContains: "query cannot be empty",
		},
		{
			name: "invalid policy syntax",
			policy: `
				package example
				allow {
					invalid syntax here
			`,
			query:       "data.example.allow",
			wantErr:     true,
			errContains: "failed to prepare policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(ctx, tt.policy, tt.query)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewEngine() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("NewEngine() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("NewEngine() unexpected error = %v", err)
				return
			}
			if engine == nil {
				t.Errorf("NewEngine() returned nil engine")
			}
		})
	}
}

func TestEngine_Eval(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		policy  string
		query   string
		input   interface{}
		want    bool
		wantErr bool
	}{
		{
			name: "allow GET request",
			policy: `
				package example
				allow {
					input.method == "GET"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"method": "GET",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "deny POST request",
			policy: `
				package example
				allow {
					input.method == "GET"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"method": "POST",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "allow with multiple conditions",
			policy: `
				package example
				allow {
					input.method == "GET"
					input.path == "/api/public"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"method": "GET",
				"path":   "/api/public",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "deny with one condition failing",
			policy: `
				package example
				allow {
					input.method == "GET"
					input.path == "/api/public"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"method": "GET",
				"path":   "/api/private",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "allow with user role check",
			policy: `
				package example
				allow {
					input.user.role == "admin"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"role": "admin",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "deny with wrong user role",
			policy: `
				package example
				allow {
					input.user.role == "admin"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"role": "user",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "complex policy with OR logic",
			policy: `
				package example
				allow {
					input.method == "GET"
				}
				allow {
					input.user.role == "admin"
				}
			`,
			query: "data.example.allow",
			input: map[string]interface{}{
				"method": "POST",
				"user": map[string]interface{}{
					"role": "admin",
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(ctx, tt.policy, tt.query)
			if err != nil {
				t.Fatalf("NewEngine() failed: %v", err)
			}

			got, err := engine.Eval(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Engine.Eval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Eval_NilEngine(t *testing.T) {
	ctx := context.Background()
	var engine *Engine

	_, err := engine.Eval(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Engine.Eval() with nil engine should return error")
	}
	if !contains(err.Error(), "engine is nil") {
		t.Errorf("Engine.Eval() error = %v, want error containing 'engine is nil'", err)
	}
}

func TestEngine_Policy(t *testing.T) {
	ctx := context.Background()
	policy := `
		package example
		allow = true
	`
	query := "data.example.allow"

	engine, err := NewEngine(ctx, policy, query)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine.Policy() != policy {
		t.Errorf("Engine.Policy() = %v, want %v", engine.Policy(), policy)
	}
}

func TestEngine_Query(t *testing.T) {
	ctx := context.Background()
	policy := `
		package example
		allow = true
	`
	query := "data.example.allow"

	engine, err := NewEngine(ctx, policy, query)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine.Query() != query {
		t.Errorf("Engine.Query() = %v, want %v", engine.Query(), query)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
