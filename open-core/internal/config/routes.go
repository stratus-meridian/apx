package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// RouteConfig represents a single route configuration
type RouteConfig struct {
	Path      string   `yaml:"path"`
	Backend   string   `yaml:"backend"`
	Mode      string   `yaml:"mode"` // "sync" or "async"
	Methods   []string `yaml:"methods"`
	PathStrip string   `yaml:"path_strip"` // Prefix to strip before proxying
}

// RoutesConfig represents all route configurations
type RoutesConfig struct {
	Routes []RouteConfig `yaml:"routes"`
}

// LoadRoutes loads route configurations from file
func LoadRoutes(filename string) ([]RouteConfig, error) {
	// If no file specified, return empty routes (use defaults)
	if filename == "" {
		return []RouteConfig{}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read routes file: %w", err)
	}

	var cfg RoutesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse routes YAML: %w", err)
	}

	// Validate and set defaults
	for i := range cfg.Routes {
		route := &cfg.Routes[i]

		// Default mode to async
		if route.Mode == "" {
			route.Mode = "async"
		}

		// Validate mode
		if route.Mode != "sync" && route.Mode != "async" {
			return nil, fmt.Errorf("invalid mode '%s' for route %s (must be 'sync' or 'async')", route.Mode, route.Path)
		}

		// Default methods to all if not specified
		if len(route.Methods) == 0 {
			route.Methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
		}
	}

	return cfg.Routes, nil
}

// LoadRoutesFromEnv loads routes from environment variable ROUTES_CONFIG
// Format: PATH1=BACKEND1:MODE1,PATH2=BACKEND2:MODE2
// Example: /mock/**=https://mocktarget.apigee.net:sync,/api/**=https://api.example.com:async
func LoadRoutesFromEnv() []RouteConfig {
	routesEnv := os.Getenv("ROUTES_CONFIG")
	if routesEnv == "" {
		return []RouteConfig{}
	}

	var routes []RouteConfig
	parts := strings.Split(routesEnv, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse PATH=BACKEND:MODE
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		path := strings.TrimSpace(kv[0])

		// Parse BACKEND:MODE
		// Format: https://backend.com:sync
		// Need to parse carefully to avoid splitting URL (https://)
		backendModeStr := strings.TrimSpace(kv[1])
		lastColon := strings.LastIndex(backendModeStr, ":")

		backend := backendModeStr
		mode := "async" // default

		// Check if the last : is for mode (not part of URL like https://)
		// Mode should be "sync" or "async", so check if what follows is a valid mode
		if lastColon > 0 && lastColon < len(backendModeStr)-1 {
			potentialMode := strings.TrimSpace(backendModeStr[lastColon+1:])
			if potentialMode == "sync" || potentialMode == "async" {
				backend = strings.TrimSpace(backendModeStr[:lastColon])
				mode = potentialMode
			}
		}

		// Auto-calculate PathStrip from wildcard patterns
		// Example: /mock/** -> strip /mock, /api/v1/** -> strip /api/v1
		pathStrip := ""
		if strings.HasSuffix(path, "/**") {
			pathStrip = strings.TrimSuffix(path, "/**")
		}

		routes = append(routes, RouteConfig{
			Path:      path,
			Backend:   backend,
			Mode:      mode,
			Methods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			PathStrip: pathStrip,
		})
	}

	return routes
}
