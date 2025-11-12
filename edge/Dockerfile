# Dockerfile for APX Edge Gateway - Cloud Run Deployment
# Based on official Envoy image with Cloud Run optimizations

FROM envoyproxy/envoy:v1.29-latest

# Install envsubst for environment variable substitution
USER root
RUN apt-get update && \
    apt-get install -y gettext-base && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy Envoy configuration (cloud version)
COPY edge/envoy/envoy-cloud.yaml /etc/envoy/envoy-cloud.yaml

# Copy entrypoint script
COPY edge/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Health check (Cloud Run uses /healthz)
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

# Expose port 8080 (Cloud Run requirement)
EXPOSE 8080

# Switch to non-root user
USER envoy

# Set environment defaults
ENV ROUTER_HOST="router-placeholder.example.com"
ENV PORT="8080"

# Run via entrypoint script
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD []
