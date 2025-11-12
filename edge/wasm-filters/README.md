# WASM Filters for APX Edge

WASM (WebAssembly) filters provide lightweight, sandboxed request/response transformations at the edge.

## Design Principles

1. **Micro-transforms only**: No heavy policy evaluation (that's router's job)
2. **Fast-path focused**: Tenant extraction, header injection, basic redaction
3. **Budget-aware**: Each filter adds <1ms p99 latency
4. **Fail-open**: If WASM crashes, request proceeds to router

## Available Filters

### 1. tenant_extractor
**Purpose**: Extract tenant_id from JWT claims or API key header
**Phase**: Pre-router
**Latency target**: <0.5ms p99

**Inputs:**
- JWT payload (from `x-jwt-payload` header)
- API key (from `X-API-Key` header)

**Outputs:**
- Sets `x-tenant-id` header
- Sets `x-tenant-tier` header (free/pro/enterprise)
- Sets OTEL baggage: `tenant_id`, `tenant_tier`

### 2. request_id_propagator
**Purpose**: Ensure request ID is present and propagated
**Phase**: Pre-router
**Latency target**: <0.2ms p99

**Inputs:**
- Existing `x-request-id` header (if any)

**Outputs:**
- Generates UUID if `x-request-id` missing
- Propagates to OTEL trace context

### 3. pii_redactor (optional, post-auth)
**Purpose**: Redact PII from request bodies for logging
**Phase**: Post-auth, pre-logging
**Latency target**: <2ms p99 (only runs on sampled requests)

**Inputs:**
- Request body (JSON)
- PII field list from policy

**Outputs:**
- Modified request body with `[REDACTED]` placeholders
- Only affects logged data, not actual backend request

## Development

### Prerequisites
- Rust 1.75+ (for building WASM modules)
- `cargo install cargo-wasi`
- `rustup target add wasm32-wasi`

### Build

```bash
cd wasm-filters
make build
```

This produces `.wasm` files in `wasm-filters/build/`.

### Test Locally

```bash
# Run tests
make test

# Integration test with Envoy
make integration-test
```

### Deploy

```bash
# Hash WASM module for integrity
sha256sum build/tenant_extractor.wasm

# Upload to GCS artifact store
gsutil cp build/tenant_extractor.wasm \
  gs://apx-artifacts/wasm/tenant_extractor@sha256:<hash>.wasm

# Update PolicyBundle transform reference
# transforms:
#   - wasm: tenant_extractor@sha256:<hash>
```

## Scaffold Structure

```
wasm-filters/
├── tenant_extractor/
│   ├── src/
│   │   └── lib.rs
│   ├── Cargo.toml
│   └── tests/
├── request_id_propagator/
│   ├── src/
│   │   └── lib.rs
│   └── Cargo.toml
├── pii_redactor/
│   ├── src/
│   │   └── lib.rs
│   └── Cargo.toml
├── Makefile
└── README.md
```

## Examples

### tenant_extractor (Rust pseudocode)

```rust
use proxy_wasm::traits::*;
use proxy_wasm::types::*;

#[no_mangle]
pub fn _start() {
    proxy_wasm::set_http_context(|_, _| -> Box<dyn HttpContext> {
        Box::new(TenantExtractor)
    });
}

struct TenantExtractor;

impl HttpContext for TenantExtractor {
    fn on_http_request_headers(&mut self, _num_headers: usize) -> Action {
        // Extract tenant from JWT payload
        if let Some(jwt_payload) = self.get_http_request_header("x-jwt-payload") {
            let tenant_id = extract_tenant_from_jwt(&jwt_payload);
            self.set_http_request_header("x-tenant-id", Some(&tenant_id));
        }

        // Or extract from API key header
        if let Some(api_key) = self.get_http_request_header("x-api-key") {
            let tenant_id = lookup_tenant_from_key(&api_key);
            self.set_http_request_header("x-tenant-id", Some(&tenant_id));
        }

        Action::Continue
    }
}
```

## Performance Budget

| Filter | p50 | p99 | p99.9 | Fail Mode |
|--------|-----|-----|-------|-----------|
| tenant_extractor | 0.2ms | 0.5ms | 1ms | Fail-open, log error |
| request_id_propagator | 0.1ms | 0.2ms | 0.5ms | Fail-open |
| pii_redactor | 0.5ms | 2ms | 5ms | Fail-open, skip redaction |

**Total budget**: <3ms added latency at edge (target: <20ms end-to-end)

## Monitoring

WASM filter metrics exposed to Envoy stats:
- `wasm.<filter_name>.invocations`
- `wasm.<filter_name>.failures`
- `wasm.<filter_name>.latency_ms`

## Roadmap

- [ ] M1: tenant_extractor, request_id_propagator
- [ ] M2: pii_redactor (basic)
- [ ] M3: Advanced transforms (header rewrite, path normalization)
- [ ] M4: Circuit breaker (in WASM for ultra-low latency)

## References

- [Envoy WASM documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter)
- [proxy-wasm Rust SDK](https://github.com/proxy-wasm/proxy-wasm-rust-sdk)
- [WASM performance best practices](https://github.com/proxy-wasm/spec/blob/master/docs/WebAssembly-Performance.md)
