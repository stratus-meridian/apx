package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client handles HTTP proxying to backend services
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// Config for proxy client
type Config struct {
	// Connection settings
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration

	// Timeouts
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration

	// TLS settings
	InsecureSkipVerify bool
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		DialTimeout:           10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		InsecureSkipVerify:    false,
	}
}

// NewClient creates a new proxy client with connection pooling
func NewClient(cfg *Config, logger *zap.Logger) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			// Don't follow redirects - let client handle
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger: logger,
	}
}

// ProxyRequestWithPathStrip proxies an HTTP request to a backend, optionally stripping a path prefix
func (c *Client) ProxyRequestWithPathStrip(ctx context.Context, req *http.Request, backendURL string, pathStrip string) (*http.Response, error) {
	// Parse backend URL
	backend, err := url.Parse(backendURL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}

	// Clone the request
	proxyReq := req.Clone(ctx)

	// Clear RequestURI - it's set by the server and must not be set in client requests
	proxyReq.RequestURI = ""

	// Update the URL to point to backend
	proxyReq.URL.Scheme = backend.Scheme
	proxyReq.URL.Host = backend.Host

	// Strip path prefix if configured
	targetPath := proxyReq.URL.Path
	if pathStrip != "" && strings.HasPrefix(targetPath, pathStrip) {
		targetPath = strings.TrimPrefix(targetPath, pathStrip)
		// Ensure path starts with /
		if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}
	}

	// If backend has a path, prepend it
	if backend.Path != "" && backend.Path != "/" {
		proxyReq.URL.Path = backend.Path + targetPath
	} else {
		proxyReq.URL.Path = targetPath
	}

	// Update Host header to match backend
	proxyReq.Host = backend.Host
	proxyReq.Header.Set("Host", backend.Host)

	// Add X-Forwarded headers
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if prior := proxyReq.Header.Get("X-Forwarded-For"); prior != "" {
			clientIP = prior + ", " + clientIP
		}
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}

	proxyReq.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
	if req.URL.Scheme == "" {
		proxyReq.Header.Set("X-Forwarded-Proto", "http")
	}
	proxyReq.Header.Set("X-Forwarded-Host", req.Host)

	// Add X-Real-IP
	if req.RemoteAddr != "" {
		if ip, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			proxyReq.Header.Set("X-Real-IP", ip)
		}
	}

	// Log the proxy request
	c.logger.Debug("proxying request",
		zap.String("method", proxyReq.Method),
		zap.String("url", proxyReq.URL.String()),
		zap.String("backend", backendURL),
	)

	// Execute the request
	resp, err := c.httpClient.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("backend request failed: %w", err)
	}

	return resp, nil
}

// ProxyRequest proxies an HTTP request to a backend (without path stripping)
func (c *Client) ProxyRequest(ctx context.Context, req *http.Request, backendURL string) (*http.Response, error) {
	return c.ProxyRequestWithPathStrip(ctx, req, backendURL, "")
}

// Close closes the client and cleans up connections
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// CopyResponse copies response from backend to client
func CopyResponse(dst http.ResponseWriter, src *http.Response) error {
	// Copy status code
	dst.WriteHeader(src.StatusCode)

	// Copy headers
	for key, values := range src.Header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}

	// Copy body
	_, err := io.Copy(dst, src.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %w", err)
	}

	return src.Body.Close()
}
