package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// DynamicLoader manages dynamic route configuration reloading
type DynamicLoader struct {
	controlAPIURL string
	tenantID      string
	client        *http.Client
	currentConfig *RoutesConfig
	mu            sync.RWMutex
	logger        *zap.Logger
	onChange      func([]RouteConfig) error
}

// DynamicLoaderConfig holds configuration for the dynamic loader
type DynamicLoaderConfig struct {
	ControlAPIURL string
	TenantID      string
	ReloadInterval time.Duration
	OnChange      func([]RouteConfig) error
	Logger        *zap.Logger
}

// NewDynamicLoader creates a new dynamic configuration loader
func NewDynamicLoader(cfg DynamicLoaderConfig) *DynamicLoader {
	if cfg.ReloadInterval == 0 {
		cfg.ReloadInterval = 60 * time.Second // Default to 60 seconds
	}

	return &DynamicLoader{
		controlAPIURL: cfg.ControlAPIURL,
		tenantID:      cfg.TenantID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		currentConfig: &RoutesConfig{Routes: []RouteConfig{}},
		logger:        cfg.Logger,
		onChange:      cfg.OnChange,
	}
}

// Start begins the dynamic reloading loop
func (d *DynamicLoader) Start(ctx context.Context) error {
	// Do initial load
	if err := d.reload(ctx); err != nil {
		d.logger.Warn("initial config load failed, continuing with empty config",
			zap.Error(err))
	}

	// Start periodic reload
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	d.logger.Info("dynamic config loader started",
		zap.String("control_api_url", d.controlAPIURL),
		zap.Duration("reload_interval", 60*time.Second))

	for {
		select {
		case <-ticker.C:
			if err := d.reload(ctx); err != nil {
				d.logger.Error("failed to reload config",
					zap.Error(err))
			}
		case <-ctx.Done():
			d.logger.Info("dynamic config loader stopped")
			return ctx.Err()
		}
	}
}

// reload fetches the latest configuration from control-API
func (d *DynamicLoader) reload(ctx context.Context) error {
	// If no control API URL or tenant ID, skip reloading
	if d.controlAPIURL == "" || d.tenantID == "" {
		d.logger.Debug("skipping config reload: no control API URL or tenant ID configured")
		return nil
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/gateways/routes?tenant_id=%s", d.controlAPIURL, d.tenantID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Accept header for YAML response
	req.Header.Set("Accept", "application/x-yaml")

	// Make request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML
	var newConfig RoutesConfig
	if err := yaml.Unmarshal(body, &newConfig); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate config
	for i := range newConfig.Routes {
		route := &newConfig.Routes[i]

		// Default mode to async
		if route.Mode == "" {
			route.Mode = "async"
		}

		// Validate mode
		if route.Mode != "sync" && route.Mode != "async" {
			return fmt.Errorf("invalid mode '%s' for route %s", route.Mode, route.Path)
		}

		// Default methods if not specified
		if len(route.Methods) == 0 {
			route.Methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
		}
	}

	// Check if config has changed
	d.mu.RLock()
	hasChanged := !d.configEquals(d.currentConfig.Routes, newConfig.Routes)
	d.mu.RUnlock()

	if !hasChanged {
		d.logger.Debug("config unchanged, skipping reload")
		return nil
	}

	// Config has changed, update it
	d.mu.Lock()
	d.currentConfig = &newConfig
	d.mu.Unlock()

	d.logger.Info("config reloaded",
		zap.Int("route_count", len(newConfig.Routes)),
		zap.Bool("changed", hasChanged))

	// Notify onChange callback if provided
	if d.onChange != nil {
		if err := d.onChange(newConfig.Routes); err != nil {
			d.logger.Error("onChange callback failed",
				zap.Error(err))
			return fmt.Errorf("onChange callback failed: %w", err)
		}
	}

	// Log individual routes
	for _, route := range newConfig.Routes {
		d.logger.Debug("route loaded",
			zap.String("path", route.Path),
			zap.String("backend", route.Backend),
			zap.String("mode", route.Mode),
			zap.String("path_strip", route.PathStrip))
	}

	return nil
}

// GetConfig returns the current configuration (thread-safe)
func (d *DynamicLoader) GetConfig() []RouteConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy to prevent modifications
	routes := make([]RouteConfig, len(d.currentConfig.Routes))
	copy(routes, d.currentConfig.Routes)
	return routes
}

// configEquals checks if two route configs are equal
func (d *DynamicLoader) configEquals(a, b []RouteConfig) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison (order-independent)
	aMap := make(map[string]RouteConfig)
	for _, route := range a {
		key := fmt.Sprintf("%s:%s:%s", route.Path, route.Backend, route.Mode)
		aMap[key] = route
	}

	bMap := make(map[string]RouteConfig)
	for _, route := range b {
		key := fmt.Sprintf("%s:%s:%s", route.Path, route.Backend, route.Mode)
		bMap[key] = route
	}

	// Compare maps
	if len(aMap) != len(bMap) {
		return false
	}

	for key := range aMap {
		if _, exists := bMap[key]; !exists {
			return false
		}
	}

	return true
}
