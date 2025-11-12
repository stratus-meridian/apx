# Multi-stage Dockerfile for APX Edge Gateway
# Based on official Envoy image with custom WASM filters

FROM envoyproxy/envoy:v1.29-latest as base

# Add custom WASM filters
FROM base as wasm-builder
WORKDIR /build
# WASM filters will be built separately and copied in
# For now, create placeholder directory
RUN mkdir -p /etc/envoy/wasm

# Final image
FROM base

# Copy Envoy configuration
COPY edge/envoy/envoy.yaml /etc/envoy/envoy.yaml

# Copy WASM filters (when available)
COPY --from=wasm-builder /etc/envoy/wasm /etc/envoy/wasm

# Health check
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:9901/ready || exit 1

# Expose ports
# 8080: Main HTTP port
# 9901: Admin port
EXPOSE 8080 9901

# Run Envoy
ENTRYPOINT ["/usr/local/bin/envoy"]
CMD ["-c", "/etc/envoy/envoy.yaml", "--service-cluster", "apx-edge"]
