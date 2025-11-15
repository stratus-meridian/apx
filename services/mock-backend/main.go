package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// MockBackend provides a simple echo/test service like mocktarget.apigee.net
type MockBackend struct {
	logger *zap.Logger
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	port := getEnv("PORT", "8080")
	service := getEnv("SERVICE_NAME", "apx-mock-backend")

	backend := &MockBackend{logger: logger}
	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", backend.Health).Methods("GET")
	r.HandleFunc("/healthz", backend.Health).Methods("GET")

	// Root greeting (like mocktarget.apigee.net)
	r.HandleFunc("/", backend.Root).Methods("GET")

	// Echo endpoint - returns request details
	r.HandleFunc("/echo", backend.Echo).Methods("GET", "POST", "PUT", "DELETE", "PATCH")

	// JSON endpoint - returns mock JSON data
	r.HandleFunc("/json", backend.JSON).Methods("GET")

	// XML endpoint - returns mock XML data
	r.HandleFunc("/xml", backend.XML).Methods("GET")

	// IP endpoint - returns caller's IP
	r.HandleFunc("/ip", backend.IP).Methods("GET")

	// Headers endpoint - returns all request headers
	r.HandleFunc("/headers", backend.Headers).Methods("GET")

	// User-Agent endpoint
	r.HandleFunc("/user-agent", backend.UserAgent).Methods("GET")

	// Delay endpoint - delays response by N seconds
	r.HandleFunc("/delay/{seconds}", backend.Delay).Methods("GET")

	// Status endpoint - returns specified status code
	r.HandleFunc("/status/{code}", backend.Status).Methods("GET", "POST")

	// UUID endpoint - generates a UUID
	r.HandleFunc("/uuid", backend.UUID).Methods("GET")

	// Base64 endpoint - base64 encode/decode
	r.HandleFunc("/base64/encode/{data}", backend.Base64Encode).Methods("GET")
	r.HandleFunc("/base64/decode/{data}", backend.Base64Decode).Methods("GET")

	// Stream endpoint - SSE stream
	r.HandleFunc("/stream/{count}", backend.Stream).Methods("GET")

	// Bytes endpoint - returns N random bytes
	r.HandleFunc("/bytes/{n}", backend.Bytes).Methods("GET")

	// APX-specific showcase endpoints
	r.HandleFunc("/showcase/rate-limit", backend.RateLimitTest).Methods("GET")
	r.HandleFunc("/showcase/quota", backend.QuotaTest).Methods("GET")
	r.HandleFunc("/showcase/auth", backend.AuthTest).Methods("GET")
	r.HandleFunc("/showcase/transforms", backend.TransformTest).Methods("POST")
	r.HandleFunc("/showcase/circuit-breaker", backend.CircuitBreakerTest).Methods("GET")
	r.HandleFunc("/showcase/retry", backend.RetryTest).Methods("GET")
	r.HandleFunc("/showcase/cache", backend.CacheTest).Methods("GET")
	r.HandleFunc("/showcase/cors", backend.CORSTest).Methods("OPTIONS", "GET")
	r.HandleFunc("/showcase/streaming", backend.StreamingTest).Methods("GET")
	r.HandleFunc("/showcase/large-response", backend.LargeResponseTest).Methods("GET")
	r.HandleFunc("/showcase/timeout", backend.TimeoutTest).Methods("GET")
	r.HandleFunc("/showcase/slow", backend.SlowTest).Methods("GET")

	// RESTful CRUD examples
	r.HandleFunc("/api/users", backend.ListUsers).Methods("GET")
	r.HandleFunc("/api/users", backend.CreateUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", backend.GetUser).Methods("GET")
	r.HandleFunc("/api/users/{id}", backend.UpdateUser).Methods("PUT", "PATCH")
	r.HandleFunc("/api/users/{id}", backend.DeleteUser).Methods("DELETE")

	// GraphQL-style endpoint
	r.HandleFunc("/graphql", backend.GraphQL).Methods("POST")

	// Webhook simulation
	r.HandleFunc("/webhooks/incoming", backend.WebhookReceiver).Methods("POST")

	// File upload simulation
	r.HandleFunc("/upload", backend.FileUpload).Methods("POST")

	// Catch-all echo endpoint
	r.PathPrefix("/").HandlerFunc(backend.Echo)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("mock backend starting",
		zap.String("port", port),
		zap.String("service", service))

	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func (mb *MockBackend) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "apx-mock-backend",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (mb *MockBackend) Root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		mb.Echo(w, r)
		return
	}
	
	// Return HTML for browsers, plain text for curl/API clients
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "text/html") {
		mb.RootHTML(w, r)
	} else {
		mb.RootPlain(w, r)
	}
}

func (mb *MockBackend) RootHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APX MockTarget - API Testing & Development Service</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
            background: #020617;
            color: #e2e8f0;
            line-height: 1.6;
            min-height: 100vh;
            overflow-x: hidden;
        }

        /* Animated grid background */
        body::before {
            content: '';
            position: fixed;
            inset: 0;
            background-image:
                linear-gradient(to right, rgba(6, 182, 212, 0.05) 1px, transparent 1px),
                linear-gradient(to bottom, rgba(6, 182, 212, 0.05) 1px, transparent 1px);
            background-size: 64px 64px;
            pointer-events: none;
            z-index: 0;
        }

        .container {
            position: relative;
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem 1.5rem;
            z-index: 1;
        }

        /* Header */
        header {
            text-align: center;
            padding: 3rem 2rem;
            margin-bottom: 3rem;
            position: relative;
            border-bottom: 1px solid rgba(6, 182, 212, 0.2);
        }

        .logo {
            display: inline-flex;
            align-items: center;
            gap: 0.75rem;
            margin-bottom: 1rem;
        }

        .logo-icon {
            width: 48px;
            height: 48px;
            background: linear-gradient(135deg, #06b6d4 0%, #10b981 100%);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-center;
            font-size: 24px;
        }

        header h1 {
            font-size: 3rem;
            font-weight: 700;
            background: linear-gradient(135deg, #06b6d4 0%, #10b981 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
            font-family: 'Courier New', monospace;
        }

        header p {
            font-size: 1.25rem;
            color: #94a3b8;
            margin-bottom: 1rem;
        }

        .badge {
            display: inline-block;
            padding: 0.5rem 1rem;
            background: rgba(6, 182, 212, 0.1);
            border: 1px solid rgba(6, 182, 212, 0.3);
            border-radius: 6px;
            font-size: 0.875rem;
            color: #06b6d4;
            font-family: 'Courier New', monospace;
        }

        .status-indicator {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            margin-top: 1rem;
            padding: 0.5rem 1rem;
            background: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.3);
            border-radius: 6px;
            font-size: 0.875rem;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background: #10b981;
            border-radius: 50%;
            animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        /* Section */
        .section {
            margin-bottom: 3rem;
            background: rgba(15, 23, 42, 0.6);
            border: 1px solid rgba(51, 65, 85, 0.5);
            border-radius: 12px;
            padding: 2rem;
            backdrop-filter: blur(10px);
        }

        .section h2 {
            font-size: 1.75rem;
            color: #06b6d4;
            margin-bottom: 1.5rem;
            font-family: 'Courier New', monospace;
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .section h2::before {
            content: '>';
            color: #10b981;
            font-weight: bold;
        }

        /* Endpoint Grid */
        .endpoint-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 1rem;
        }

        .endpoint {
            background: linear-gradient(135deg, rgba(15, 23, 42, 0.8) 0%, rgba(30, 41, 59, 0.4) 100%);
            border: 1px solid rgba(51, 65, 85, 0.5);
            border-left: 3px solid #06b6d4;
            border-radius: 8px;
            padding: 1.25rem;
            transition: all 0.3s ease;
            position: relative;
        }

        .endpoint:hover {
            border-color: rgba(6, 182, 212, 0.6);
            box-shadow: 0 0 24px rgba(6, 182, 212, 0.2);
            transform: translateY(-2px);
        }

        .endpoint:hover::before {
            content: '';
            position: absolute;
            inset: 0;
            background: linear-gradient(135deg, rgba(6, 182, 212, 0.05) 0%, rgba(16, 185, 129, 0.05) 100%);
            border-radius: 8px;
            pointer-events: none;
        }

        .method {
            display: inline-block;
            padding: 0.35rem 0.65rem;
            border-radius: 4px;
            font-size: 0.75rem;
            font-weight: 700;
            font-family: 'Courier New', monospace;
            margin-right: 0.75rem;
            letter-spacing: 0.5px;
        }

        .method-get { background: rgba(16, 185, 129, 0.2); color: #10b981; border: 1px solid rgba(16, 185, 129, 0.4); }
        .method-post { background: rgba(59, 130, 246, 0.2); color: #3b82f6; border: 1px solid rgba(59, 130, 246, 0.4); }
        .method-put { background: rgba(251, 191, 36, 0.2); color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.4); }
        .method-patch { background: rgba(6, 182, 212, 0.2); color: #06b6d4; border: 1px solid rgba(6, 182, 212, 0.4); }
        .method-delete { background: rgba(239, 68, 68, 0.2); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.4); }
        .method-any { background: rgba(148, 163, 184, 0.2); color: #94a3b8; border: 1px solid rgba(148, 163, 184, 0.4); }

        .endpoint-path {
            font-family: 'Courier New', monospace;
            font-weight: 600;
            color: #e2e8f0;
            font-size: 1rem;
            display: block;
            margin-bottom: 0.5rem;
        }

        .endpoint-desc {
            font-size: 0.875rem;
            color: #94a3b8;
            line-height: 1.5;
        }

        /* CTA Section */
        .cta-section {
            background: linear-gradient(135deg, rgba(6, 182, 212, 0.1) 0%, rgba(16, 185, 129, 0.1) 100%);
            border: 1px solid rgba(6, 182, 212, 0.3);
            border-radius: 12px;
            padding: 2.5rem 2rem;
            text-align: center;
            margin: 3rem 0;
        }

        .cta-section h3 {
            font-size: 1.75rem;
            color: #e2e8f0;
            margin-bottom: 1.5rem;
            font-family: 'Courier New', monospace;
        }

        .code-block {
            background: rgba(2, 6, 23, 0.8);
            border: 1px solid rgba(51, 65, 85, 0.5);
            padding: 1.25rem;
            border-radius: 8px;
            font-family: 'Courier New', monospace;
            font-size: 0.9rem;
            text-align: left;
            overflow-x: auto;
            margin: 1.5rem 0;
            color: #06b6d4;
            line-height: 1.6;
        }

        .btn {
            display: inline-block;
            padding: 0.875rem 1.75rem;
            background: linear-gradient(135deg, #06b6d4 0%, #10b981 100%);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: all 0.3s ease;
            margin: 0.5rem;
            border: none;
            font-size: 1rem;
        }

        .btn:hover {
            box-shadow: 0 0 24px rgba(6, 182, 212, 0.4);
            transform: translateY(-2px);
        }

        .btn-secondary {
            background: transparent;
            border: 1px solid rgba(6, 182, 212, 0.5);
            color: #06b6d4;
        }

        .btn-secondary:hover {
            background: rgba(6, 182, 212, 0.1);
            box-shadow: 0 0 16px rgba(6, 182, 212, 0.2);
        }

        /* Footer */
        footer {
            text-align: center;
            padding: 2.5rem 2rem;
            margin-top: 4rem;
            border-top: 1px solid rgba(51, 65, 85, 0.5);
            color: #64748b;
        }

        footer p {
            margin-bottom: 0.5rem;
        }

        footer a {
            color: #06b6d4;
            text-decoration: none;
            transition: color 0.2s;
        }

        footer a:hover {
            color: #10b981;
        }

        @media (max-width: 768px) {
            header h1 { font-size: 2rem; }
            .endpoint-grid { grid-template-columns: 1fr; }
            .section { padding: 1.5rem; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">
                <div class="logo-icon">🎯</div>
            </div>
            <h1>APX MockTarget</h1>
            <p>API Testing & Development Service</p>
            <div class="badge">Similar to mocktarget.apigee.net</div>
            <div class="status-indicator">
                <span class="status-dot"></span>
                <span>All Systems Operational</span>
            </div>
        </header>

        <div class="section">
            <h2>Basic Endpoints</h2>
                <div class="endpoint-grid">
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/</span>
                        <div class="endpoint-desc">This page</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/health</span>
                        <div class="endpoint-desc">Health check</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-any">ANY</span>
                        <span class="endpoint-path">/echo</span>
                        <div class="endpoint-desc">Echo request details</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/json</span>
                        <div class="endpoint-desc">Sample JSON response</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/xml</span>
                        <div class="endpoint-desc">Sample XML response</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/ip</span>
                        <div class="endpoint-desc">Client IP address</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/headers</span>
                        <div class="endpoint-desc">All request headers</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/user-agent</span>
                        <div class="endpoint-desc">User-Agent header</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/delay/:N</span>
                        <div class="endpoint-desc">Delay N seconds (max 10)</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-any">ANY</span>
                        <span class="endpoint-path">/status/:code</span>
                        <div class="endpoint-desc">Return HTTP status code</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/uuid</span>
                        <div class="endpoint-desc">Generate UUID</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/bytes/:N</span>
                        <div class="endpoint-desc">Return N bytes</div>
                    </div>
                </div>
            </div>

        </div>

        <div class="section">
            <h2>APX Gateway Showcase</h2>
                <div class="endpoint-grid">
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/rate-limit</span>
                        <div class="endpoint-desc">Rate limiting demo</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/quota</span>
                        <div class="endpoint-desc">Quota enforcement</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/auth</span>
                        <div class="endpoint-desc">Authentication testing</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-post">POST</span>
                        <span class="endpoint-path">/showcase/transforms</span>
                        <div class="endpoint-desc">Request transformation</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/circuit-breaker</span>
                        <div class="endpoint-desc">Circuit breaker demo</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/retry</span>
                        <div class="endpoint-desc">Retry policy demo</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/cache</span>
                        <div class="endpoint-desc">Response caching</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/streaming</span>
                        <div class="endpoint-desc">SSE streaming</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/large-response</span>
                        <div class="endpoint-desc">Large payload (~1MB)</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/showcase/timeout</span>
                        <div class="endpoint-desc">Timeout testing (35s)</div>
                    </div>
                </div>
            </div>

        </div>

        <div class="section">
            <h2>RESTful CRUD API</h2>
                <div class="endpoint-grid">
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/api/users</span>
                        <div class="endpoint-desc">List all users</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-post">POST</span>
                        <span class="endpoint-path">/api/users</span>
                        <div class="endpoint-desc">Create new user</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-get">GET</span>
                        <span class="endpoint-path">/api/users/:id</span>
                        <div class="endpoint-desc">Get user by ID</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-put">PUT</span>
                        <span class="endpoint-path">/api/users/:id</span>
                        <div class="endpoint-desc">Update user</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-delete">DELETE</span>
                        <span class="endpoint-path">/api/users/:id</span>
                        <div class="endpoint-desc">Delete user</div>
                    </div>
                </div>
            </div>

        </div>

        <div class="section">
            <h2>Advanced Endpoints</h2>
                <div class="endpoint-grid">
                    <div class="endpoint">
                        <span class="method method-post">POST</span>
                        <span class="endpoint-path">/graphql</span>
                        <div class="endpoint-desc">GraphQL endpoint</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-post">POST</span>
                        <span class="endpoint-path">/webhooks/incoming</span>
                        <div class="endpoint-desc">Webhook receiver</div>
                    </div>
                    <div class="endpoint">
                        <span class="method method-post">POST</span>
                        <span class="endpoint-path">/upload</span>
                        <div class="endpoint-desc">File upload simulation</div>
                    </div>
                </div>
            </div>

        </div>

        <div class="cta-section">
            <h3>&gt; Try it with APX Gateway</h3>
            <div class="code-block"># Test with APX API Gateway<br>curl https://mocktarget.apx.build/showcase/rate-limit \<br>  -H "X-APX-Key: YOUR_API_KEY" \<br>  -H "Content-Type: application/json"</div>
            <a href="https://portal.apx.build/console/playground" class="btn">Open Playground</a>
            <a href="https://portal.apx.build/docs/quickstart" class="btn btn-secondary">View Docs</a>
        </div>

        <footer>
            <p>Powered by <strong>APX</strong> - AI-Native API Gateway Platform</p>
            <p>Similar to mocktarget.apigee.net | <a href="https://github.com/apx">GitHub</a> | <a href="https://portal.apx.build">Portal</a></p>
        </footer>
    </div>
</body>
</html>`
	
	w.Write([]byte(html))
}

func (mb *MockBackend) RootPlain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, Guest!\n")
	fmt.Fprintf(w, "\n═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "  APX Mock Backend - API Gateway Testing Service\n")
	fmt.Fprintf(w, "  Similar to mocktarget.apigee.net\n")
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════\n\n")

	fmt.Fprintf(w, "📋 BASIC ENDPOINTS:\n")
	fmt.Fprintf(w, "  GET  /              - This page\n")
	fmt.Fprintf(w, "  GET  /health        - Health check\n")
	fmt.Fprintf(w, "  ANY  /echo          - Echo request details\n")
	fmt.Fprintf(w, "  GET  /json          - Return sample JSON\n")
	fmt.Fprintf(w, "  GET  /xml           - Return sample XML\n")
	fmt.Fprintf(w, "  GET  /ip            - Return client IP\n")
	fmt.Fprintf(w, "  GET  /headers       - Return all headers\n")
	fmt.Fprintf(w, "  GET  /user-agent    - Return User-Agent\n")
	fmt.Fprintf(w, "  GET  /delay/:N      - Delay response N seconds (max 10)\n")
	fmt.Fprintf(w, "  ANY  /status/:code  - Return specified status code\n")
	fmt.Fprintf(w, "  GET  /uuid          - Generate UUID\n")
	fmt.Fprintf(w, "  GET  /stream/:N     - SSE stream N events\n")
	fmt.Fprintf(w, "  GET  /bytes/:N      - Return N bytes\n\n")

	fmt.Fprintf(w, "🚀 APX GATEWAY SHOWCASE:\n")
	fmt.Fprintf(w, "  GET  /showcase/rate-limit      - Rate limiting demo\n")
	fmt.Fprintf(w, "  GET  /showcase/quota           - Quota enforcement demo\n")
	fmt.Fprintf(w, "  GET  /showcase/auth            - Authentication testing\n")
	fmt.Fprintf(w, "  POST /showcase/transforms      - Request transformation demo\n")
	fmt.Fprintf(w, "  GET  /showcase/circuit-breaker - Circuit breaker demo\n")
	fmt.Fprintf(w, "  GET  /showcase/retry           - Retry policy demo\n")
	fmt.Fprintf(w, "  GET  /showcase/cache           - Response caching demo\n")
	fmt.Fprintf(w, "  GET  /showcase/cors            - CORS handling demo\n")
	fmt.Fprintf(w, "  GET  /showcase/streaming       - SSE streaming demo\n")
	fmt.Fprintf(w, "  GET  /showcase/large-response  - Large payload (~1MB)\n")
	fmt.Fprintf(w, "  GET  /showcase/timeout         - Timeout testing (35s)\n")
	fmt.Fprintf(w, "  GET  /showcase/slow?delay=N    - Slow response (N seconds)\n\n")

	fmt.Fprintf(w, "👥 RESTful CRUD API (Users):\n")
	fmt.Fprintf(w, "  GET    /api/users     - List all users\n")
	fmt.Fprintf(w, "  POST   /api/users     - Create new user\n")
	fmt.Fprintf(w, "  GET    /api/users/:id - Get user by ID\n")
	fmt.Fprintf(w, "  PUT    /api/users/:id - Update user\n")
	fmt.Fprintf(w, "  PATCH  /api/users/:id - Partial update user\n")
	fmt.Fprintf(w, "  DELETE /api/users/:id - Delete user\n\n")

	fmt.Fprintf(w, "🔧 ADVANCED:\n")
	fmt.Fprintf(w, "  POST /graphql           - GraphQL endpoint\n")
	fmt.Fprintf(w, "  POST /webhooks/incoming - Webhook receiver\n")
	fmt.Fprintf(w, "  POST /upload            - File upload simulation\n\n")

	fmt.Fprintf(w, "💡 Try it with APX:\n")
	fmt.Fprintf(w, "  curl https://api.apx.build/mock/showcase/rate-limit \\\n")
	fmt.Fprintf(w, "    -H \"Authorization: Bearer YOUR_API_KEY\"\n\n")

	fmt.Fprintf(w, "📚 Playground: https://portal.apx.build/console/playground\n")
}

func (mb *MockBackend) Echo(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}

	response := map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   r.URL.Query(),
		"headers": headers,
		"body":    string(body),
		"host":    r.Host,
		"remote":  r.RemoteAddr,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (mb *MockBackend) JSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slideshow": map[string]interface{}{
			"author": "APX Team",
			"date":   "2025-11-15",
			"title":  "Sample Slide Show",
			"slides": []map[string]string{
				{"title": "Welcome to APX", "type": "all"},
				{"title": "API Gateway Features", "type": "all"},
			},
		},
	})
}

func (mb *MockBackend) XML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<slideshow>
  <author>APX Team</author>
  <date>2025-11-15</date>
  <title>Sample Slide Show</title>
  <slide>
    <title>Welcome to APX</title>
    <type>all</type>
  </slide>
  <slide>
    <title>API Gateway Features</title>
    <type>all</type>
  </slide>
</slideshow>`)
}

func (mb *MockBackend) IP(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"origin": ip})
}

func (mb *MockBackend) Headers(w http.ResponseWriter, r *http.Request) {
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"headers": headers})
}

func (mb *MockBackend) UserAgent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"user-agent": r.UserAgent()})
}

func (mb *MockBackend) Delay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seconds := vars["seconds"]
	
	var delay time.Duration
	fmt.Sscanf(seconds, "%d", &delay)
	if delay > 10 {
		delay = 10 // Max 10 seconds
	}
	
	time.Sleep(delay * time.Second)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delayed": fmt.Sprintf("%ds", delay),
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (mb *MockBackend) Status(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	
	var statusCode int
	fmt.Sscanf(code, "%d", &statusCode)
	if statusCode < 100 || statusCode > 599 {
		statusCode = 200
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    statusCode,
		"message": http.StatusText(statusCode),
	})
}

func (mb *MockBackend) UUID(w http.ResponseWriter, r *http.Request) {
	// Simple UUID v4 generation
	uuid := fmt.Sprintf("%08x-%04x-4%03x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		time.Now().UnixNano()>>48&0xFFF,
		0x8000|(time.Now().UnixNano()&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF,
	)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
}

func (mb *MockBackend) Base64Encode(w http.ResponseWriter, r *http.Request) {
	// Simplified - implement if needed
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "base64 encode"})
}

func (mb *MockBackend) Base64Decode(w http.ResponseWriter, r *http.Request) {
	// Simplified - implement if needed
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "base64 decode"})
}

func (mb *MockBackend) Stream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	count := 10
	fmt.Sscanf(vars["count"], "%d", &count)
	if count > 100 {
		count = 100
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for i := 0; i < count; i++ {
		fmt.Fprintf(w, "data: {\"event\": %d, \"time\": \"%s\"}\n\n", i, time.Now().Format(time.RFC3339))
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}
}

func (mb *MockBackend) Bytes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	n := 100
	fmt.Sscanf(vars["n"], "%d", &n)
	if n > 10000 {
		n = 10000
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i % 256)
	}
	w.Write(data)
}

// APX Showcase Endpoints - demonstrate gateway capabilities

func (mb *MockBackend) RateLimitTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", "100")
	w.Header().Set("X-RateLimit-Remaining", "99")
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Rate limit testing endpoint",
		"tip":     "APX will add rate limiting headers automatically",
		"headers": map[string]string{
			"X-RateLimit-Limit":     "100",
			"X-RateLimit-Remaining": "99",
			"X-RateLimit-Reset":     "timestamp",
		},
	})
}

func (mb *MockBackend) QuotaTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Quota enforcement testing",
		"quota_used":    250,
		"quota_limit":   1000,
		"quota_percent": 25.0,
		"resets_at":     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

func (mb *MockBackend) AuthTest(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	apiKey := r.Header.Get("X-API-Key")
	
	response := map[string]interface{}{
		"message":          "Authentication testing endpoint",
		"auth_header":      authHeader != "",
		"api_key_present":  apiKey != "",
		"authenticated_as": "test-user",
	}
	
	if authHeader == "" && apiKey == "" {
		w.WriteHeader(http.StatusUnauthorized)
		response["error"] = "No authentication provided"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (mb *MockBackend) TransformTest(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	var input map[string]interface{}
	json.Unmarshal(body, &input)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Request transformation demo",
		"original":   input,
		"transformed": map[string]interface{}{
			"uppercase_keys": true,
			"added_metadata": map[string]string{
				"processed_at": time.Now().Format(time.RFC3339),
				"gateway":      "APX",
			},
		},
		"tip": "APX can transform requests/responses using policies",
	})
}

func (mb *MockBackend) CircuitBreakerTest(w http.ResponseWriter, r *http.Request) {
	// Simulate occasional failures for circuit breaker testing
	if time.Now().Second()%5 == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "service_unavailable",
			"message": "Simulated failure for circuit breaker testing",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Circuit breaker test - healthy response",
		"status":  "healthy",
		"tip":     "APX circuit breaker will trip after consecutive failures",
	})
}

func (mb *MockBackend) RetryTest(w http.ResponseWriter, r *http.Request) {
	attemptHeader := r.Header.Get("X-Retry-Attempt")
	attempt := 1
	if attemptHeader != "" {
		fmt.Sscanf(attemptHeader, "%d", &attempt)
	}
	
	// Fail first 2 attempts, succeed on 3rd
	if attempt < 3 {
		w.WriteHeader(http.StatusBadGateway)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "temporary_failure",
			"attempt": attempt,
			"message": "Simulated failure - retry recommended",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Retry successful",
		"attempt":  attempt,
		"tip":      "APX automatically retries failed requests based on policy",
	})
}

func (mb *MockBackend) CacheTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, time.Now().Unix()/300))
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Caching demo endpoint",
		"data":       "This response can be cached for 5 minutes",
		"cached_at":  time.Now().Format(time.RFC3339),
		"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		"tip":        "APX can cache responses at edge for better performance",
	})
}

func (mb *MockBackend) CORSTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "CORS preflight handled",
		"origin":  r.Header.Get("Origin"),
		"tip":     "APX handles CORS automatically based on gateway config",
	})
}

func (mb *MockBackend) StreamingTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	
	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"event":     i,
			"message":   fmt.Sprintf("Streaming event %d", i),
			"timestamp": time.Now().Format(time.RFC3339),
			"progress":  (i + 1) * 10,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		time.Sleep(200 * time.Millisecond)
	}
	
	fmt.Fprintf(w, "data: {\"event\":\"complete\",\"message\":\"Streaming complete\"}\n\n")
}

func (mb *MockBackend) LargeResponseTest(w http.ResponseWriter, r *http.Request) {
	// Generate a large JSON response (~1MB)
	items := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = map[string]interface{}{
			"id":          i,
			"name":        fmt.Sprintf("Item %d", i),
			"description": strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 10),
			"created_at":  time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			"metadata": map[string]string{
				"category": "test",
				"status":   "active",
			},
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Large response test",
		"item_count":  1000,
		"items":       items,
		"tip":         "APX can compress large responses automatically",
	})
}

func (mb *MockBackend) TimeoutTest(w http.ResponseWriter, r *http.Request) {
	// Sleep for 35 seconds to trigger timeout (most gateways timeout at 30s)
	seconds := 35
	if s := r.URL.Query().Get("seconds"); s != "" {
		fmt.Sscanf(s, "%d", &seconds)
	}
	
	time.Sleep(time.Duration(seconds) * time.Second)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "If you see this, timeout did not occur",
		"slept":   seconds,
	})
}

func (mb *MockBackend) SlowTest(w http.ResponseWriter, r *http.Request) {
	// Configurable slow endpoint
	seconds := 3
	if s := r.URL.Query().Get("delay"); s != "" {
		fmt.Sscanf(s, "%d", &seconds)
		if seconds > 10 {
			seconds = 10
		}
	}
	
	time.Sleep(time.Duration(seconds) * time.Second)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Slow endpoint demo",
		"delay":   seconds,
		"tip":     "APX tracks latency metrics for all requests",
	})
}

// RESTful CRUD Examples

var mockUsers = map[string]map[string]interface{}{
	"1": {"id": "1", "name": "Alice Johnson", "email": "alice@example.com", "role": "admin"},
	"2": {"id": "2", "name": "Bob Smith", "email": "bob@example.com", "role": "user"},
	"3": {"id": "3", "name": "Carol Williams", "email": "carol@example.com", "role": "user"},
}

func (mb *MockBackend) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := make([]map[string]interface{}, 0, len(mockUsers))
	for _, user := range mockUsers {
		users = append(users, user)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"total": len(users),
		"page":  1,
		"limit": 10,
	})
}

func (mb *MockBackend) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	user, exists := mockUsers[id]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (mb *MockBackend) CreateUser(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	var user map[string]interface{}
	json.Unmarshal(body, &user)
	
	user["id"] = fmt.Sprintf("%d", len(mockUsers)+1)
	user["created_at"] = time.Now().Format(time.RFC3339)
	
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/users/%s", user["id"]))
	json.NewEncoder(w).Encode(user)
}

func (mb *MockBackend) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	user, exists := mockUsers[id]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}
	
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	var updates map[string]interface{}
	json.Unmarshal(body, &updates)
	
	for k, v := range updates {
		user[k] = v
	}
	user["updated_at"] = time.Now().Format(time.RFC3339)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (mb *MockBackend) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if _, exists := mockUsers[id]; !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func (mb *MockBackend) GraphQL(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	var query map[string]interface{}
	json.Unmarshal(body, &query)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"user": map[string]interface{}{
				"id":    "1",
				"name":  "Alice Johnson",
				"email": "alice@example.com",
				"posts": []map[string]string{
					{"id": "1", "title": "GraphQL with APX"},
					{"id": "2", "title": "API Gateway Patterns"},
				},
			},
		},
		"tip": "APX can route GraphQL queries like any other POST request",
	})
}

func (mb *MockBackend) WebhookReceiver(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	signature := r.Header.Get("X-Webhook-Signature")
	
	mb.logger.Info("webhook received",
		zap.String("signature", signature),
		zap.Int("size", len(body)))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "received",
		"webhook_id":  fmt.Sprintf("wh_%d", time.Now().Unix()),
		"received_at": time.Now().Format(time.RFC3339),
		"size":        len(body),
		"tip":         "APX can validate webhook signatures and route to backends",
	})
}

func (mb *MockBackend) FileUpload(w http.ResponseWriter, r *http.Request) {
	// Simulate file upload
	r.ParseMultipartForm(10 << 20) // 10 MB max
	
	files := []string{}
	if r.MultipartForm != nil {
		for _, fileHeaders := range r.MultipartForm.File {
			for _, fileHeader := range fileHeaders {
				files = append(files, fileHeader.Filename)
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "uploaded",
		"files":        files,
		"file_count":   len(files),
		"upload_id":    fmt.Sprintf("up_%d", time.Now().Unix()),
		"tip":          "APX can handle multipart file uploads",
	})
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

