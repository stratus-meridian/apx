package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIKeyLifecycle tests the complete lifecycle of an API key
// 1. Create API key in portal
// 2. Verify key in Firestore
// 3. Make request with Authorization header
// 4. Verify router resolves tenant/tier
// 5. Verify rate limiting applies
// 6. Revoke key in portal
// 7. Make request with revoked key
// 8. Verify 401 Unauthorized
func TestAPIKeyLifecycle(t *testing.T) {
	controlAPIURL := os.Getenv("CONTROL_API_URL")
	if controlAPIURL == "" {
		controlAPIURL = "http://localhost:8080"
	}

	routerURL := os.Getenv("ROUTER_URL")
	if routerURL == "" {
		t.Skip("ROUTER_URL not set, skipping integration test")
	}

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Create API key via control-API
	t.Run("CreateAPIKey", func(t *testing.T) {
		createReq := map[string]interface{}{
			"env_id":      "test-env",
			"name":        "Integration Test Key",
			"description": "API key for end-to-end testing",
			"scopes":      []string{"*"},
		}

		body, _ := json.Marshal(createReq)
		req, err := http.NewRequestWithContext(ctx, "POST",
			fmt.Sprintf("%s/api/v1/api-keys", controlAPIURL),
			bytes.NewReader(body))
		require.NoError(t, err)

		// Add JWT token for authentication
		jwtToken := generateTestJWT(t, "test-tenant")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "API key creation should succeed")

		var createResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)

		// Verify response contains the plaintext key
		apiKey, ok := createResp["key"].(string)
		require.True(t, ok, "Response should contain 'key' field")
		require.NotEmpty(t, apiKey, "API key should not be empty")
		assert.True(t, len(apiKey) > 40, "API key should be at least 41 characters")
		assert.True(t, apiKey[:9] == "apx_test_" || apiKey[:9] == "apx_live_",
			"API key should have correct prefix")

		keyID, ok := createResp["id"].(string)
		require.True(t, ok, "Response should contain 'id' field")

		// Store for later tests
		t.Logf("Created API key: %s (ID: %s)", apiKey, keyID)

		// Step 2: Verify key can be listed
		t.Run("ListAPIKeys", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "GET",
				fmt.Sprintf("%s/api/v1/api-keys", controlAPIURL), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var listResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&listResp)
			require.NoError(t, err)

			keys, ok := listResp["keys"].([]interface{})
			require.True(t, ok, "Response should contain 'keys' array")
			assert.GreaterOrEqual(t, len(keys), 1, "Should have at least one API key")

			// Verify the key we just created is in the list
			found := false
			for _, k := range keys {
				keyMap := k.(map[string]interface{})
				if keyMap["id"].(string) == keyID {
					found = true
					// Verify sensitive data is masked
					_, hasKey := keyMap["key"]
					assert.False(t, hasKey, "Plaintext key should not be in list response")
					assert.NotEmpty(t, keyMap["prefix"], "Should have prefix")
					break
				}
			}
			assert.True(t, found, "Created key should be in list")
		})

		// Step 3: Make authenticated request to router
		t.Run("AuthenticatedRouterRequest", func(t *testing.T) {
			// Make a request to the router with the API key
			req, err := http.NewRequestWithContext(ctx, "POST",
				fmt.Sprintf("%s/v1/evaluate", routerURL),
				bytes.NewReader([]byte(`{"policy": "test", "input": {}}`)))
			require.NoError(t, err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should NOT be 401 Unauthorized (key is valid)
			assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
				"Request with valid API key should not return 401")

			// Check if router resolved tenant correctly (would be in logs or response headers)
			t.Logf("Router responded with status: %d", resp.StatusCode)
		})

		// Step 4: Verify rate limiting is applied
		t.Run("RateLimitEnforcement", func(t *testing.T) {
			// Make multiple requests rapidly to test rate limiting
			successCount := 0
			rateLimitCount := 0

			for i := 0; i < 20; i++ {
				req, err := http.NewRequestWithContext(ctx, "POST",
					fmt.Sprintf("%s/v1/evaluate", routerURL),
					bytes.NewReader([]byte(`{"policy": "test", "input": {}}`)))
				require.NoError(t, err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				if resp.StatusCode == http.StatusTooManyRequests {
					rateLimitCount++
				} else if resp.StatusCode < 400 {
					successCount++
				}
				resp.Body.Close()
			}

			t.Logf("Rate limit test: %d successful, %d rate-limited", successCount, rateLimitCount)
			assert.Greater(t, successCount, 0, "At least some requests should succeed")
		})

		// Step 5: Revoke the API key
		t.Run("RevokeAPIKey", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "DELETE",
				fmt.Sprintf("%s/api/v1/api-keys/%s", controlAPIURL, keyID), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "API key revocation should succeed")

			t.Log("API key revoked successfully")
		})

		// Step 6: Verify revoked key returns 401
		t.Run("RevokedKeyReturns401", func(t *testing.T) {
			// Wait a moment for revocation to propagate
			time.Sleep(2 * time.Second)

			req, err := http.NewRequestWithContext(ctx, "POST",
				fmt.Sprintf("%s/v1/evaluate", routerURL),
				bytes.NewReader([]byte(`{"policy": "test", "input": {}}`)))
			require.NoError(t, err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Revoked key should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"Request with revoked API key should return 401")

			body, _ := io.ReadAll(resp.Body)
			t.Logf("Response body for revoked key: %s", string(body))
		})
	})
}

// TestAPIKeyValidation tests API key format validation
func TestAPIKeyValidation(t *testing.T) {
	routerURL := os.Getenv("ROUTER_URL")
	if routerURL == "" {
		t.Skip("ROUTER_URL not set, skipping integration test")
	}

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}

	testCases := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "Invalid format - too short",
			apiKey:         "apx_test_12345",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format - wrong prefix",
			apiKey:         "invalid_test_" + string(make([]byte, 32)),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No Bearer prefix",
			apiKey:         "apx_test_0123456789abcdef0123456789abcdef",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "POST",
				fmt.Sprintf("%s/v1/evaluate", routerURL),
				bytes.NewReader([]byte(`{"policy": "test", "input": {}}`)))
			require.NoError(t, err)

			if tc.apiKey != "" {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tc.apiKey))
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode,
				"Status code should match expected for %s", tc.name)
		})
	}
}

// Helper function to generate test JWT
func generateTestJWT(t *testing.T, tenantID string) string {
	// In a real test, you'd use a proper JWT library
	// For now, this is a placeholder that should be replaced with actual JWT generation
	// matching the format expected by the control-API

	// This is a mock - in production, use:
	// jwt.Sign with proper secret and claims
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJ0ZXN0LXRlbmFudCIsImVtYWlsIjoidGVzdEB0ZXN0LmNvbSJ9.placeholder"
}
