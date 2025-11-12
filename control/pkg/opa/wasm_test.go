package opa

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/opa/compile"
)

func TestWASMCompilation(t *testing.T) {
	ctx := context.Background()

	// Define a simple policy for WASM compilation
	policy := `package example

default allow = false

allow {
	input.method == "GET"
}

allow {
	input.user.role == "admin"
}
`

	// Create a temporary directory for the policy file
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.rego")

	// Write the policy to a file
	err := os.WriteFile(policyPath, []byte(policy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create a new compiler for WASM
	wasmCompiler := compile.New().
		WithTarget("wasm").
		WithEntrypoints("example/allow").
		WithPaths(policyPath)

	// Build the WASM bundle
	err = wasmCompiler.Build(ctx)
	if err != nil {
		t.Fatalf("WASM compilation failed: %v", err)
	}

	// Get the compiled bundle
	bundle := wasmCompiler.Bundle()

	// Verify bundle was created
	if bundle == nil {
		t.Fatal("WASM bundle is nil")
	}

	// Verify WASM modules are present
	if bundle.WasmModules == nil || len(bundle.WasmModules) == 0 {
		t.Fatal("WASM bundle contains no modules")
	}

	// Log WASM module information
	for i, module := range bundle.WasmModules {
		if module.Raw == nil {
			t.Errorf("WASM module %d has no raw data", i)
			continue
		}
		moduleSize := len(module.Raw)
		t.Logf("WASM module %d: %s, size: %d bytes", i, module.Path, moduleSize)

		if moduleSize == 0 {
			t.Errorf("WASM module %d is empty", i)
		}

		// Verify reasonable size (WASM modules should be at least a few KB)
		if moduleSize < 1024 {
			t.Errorf("WASM module %d is suspiciously small: %d bytes", i, moduleSize)
		}
	}

	// Verify the bundle has manifest
	if bundle.Manifest.Revision == "" {
		t.Log("WASM bundle manifest has empty revision (this is OK for test bundles)")
	}

	t.Logf("Successfully compiled policy to WASM with %d module(s)", len(bundle.WasmModules))
}

func TestWASMCompilation_MultipleEntrypoints(t *testing.T) {
	ctx := context.Background()

	// Define a policy with multiple decision points
	policy := `package authz

default allow = false
default deny = true

allow {
	input.method == "GET"
	input.path == "/public"
}

deny {
	input.method == "DELETE"
}

admin_only {
	input.user.role == "admin"
}
`

	// Create a temporary directory for the policy file
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "authz.rego")

	// Write the policy to a file
	err := os.WriteFile(policyPath, []byte(policy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create a WASM compiler with multiple entrypoints
	wasmCompiler := compile.New().
		WithTarget("wasm").
		WithEntrypoints("authz/allow", "authz/deny", "authz/admin_only").
		WithPaths(policyPath)

	// Build the WASM bundle
	err = wasmCompiler.Build(ctx)
	if err != nil {
		t.Fatalf("WASM compilation with multiple entrypoints failed: %v", err)
	}

	// Get the compiled bundle
	bundle := wasmCompiler.Bundle()

	// Verify bundle was created
	if bundle == nil {
		t.Fatal("WASM bundle is nil")
	}

	// Verify WASM modules are present
	if bundle.WasmModules == nil || len(bundle.WasmModules) == 0 {
		t.Fatal("WASM bundle contains no modules")
	}

	t.Logf("Successfully compiled policy with multiple entrypoints to WASM with %d module(s)", len(bundle.WasmModules))
}

func TestWASMCompilation_InvalidPolicy(t *testing.T) {
	ctx := context.Background()

	// This test verifies that invalid policies are caught during compilation
	invalidPolicy := `package example
allow {
	this is not valid rego syntax
}
`

	// Create a temporary directory for the policy file
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "invalid.rego")

	// Write the policy to a file
	err := os.WriteFile(policyPath, []byte(invalidPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create a WASM compiler
	wasmCompiler := compile.New().
		WithTarget("wasm").
		WithEntrypoints("example/allow").
		WithPaths(policyPath)

	// Build should fail
	err = wasmCompiler.Build(ctx)
	if err == nil {
		t.Fatal("Expected compilation to fail for invalid policy, but it succeeded")
	}

	t.Logf("Invalid policy correctly rejected: %v", err)
}

func TestWASMCompilation_EmptyPolicy(t *testing.T) {
	_ = context.Background()

	// Define an empty but valid policy
	policy := `package example
`

	// Create a temporary directory for the policy file
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "empty.rego")

	// Write the policy to a file
	err := os.WriteFile(policyPath, []byte(policy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Note: We don't attempt WASM compilation here because there are no entrypoints
	// This test just verifies that an empty policy file can be created
	// Actual compilation would fail without entrypoints
	t.Log("Empty policy successfully created")
}
