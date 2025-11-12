# APX Edge Gateway

The Edge Gateway is an Envoy-based reverse proxy that handles incoming API requests and routes them to the APX Router service.

## Deployment Configuration

This directory contains template deployment files. To deploy:

1. Copy `cloudbuild.yaml.example` to `cloudbuild.yaml`
2. Update the placeholders with your actual values:
   - `YOUR-PROJECT-ID`: Your GCP project ID
   - `YOUR-REPO`: Your Artifact Registry repository name
   - `REGION`: Your GCP region (e.g., `us-central1`)

3. (Optional) Create `.gcloudignore` to customize what gets uploaded during Cloud Build

**Note:** Actual deployment files (`cloudbuild.yaml`, `.gcloudignore`) are gitignored to prevent committing sensitive project information.

## Files

- `Dockerfile` - Container image definition
- `docker-entrypoint.sh` - Container startup script
- `envoy/envoy.yaml` - Envoy configuration for local development
- `envoy/envoy-cloud.yaml` - Envoy configuration for cloud deployment
- `cloudbuild.yaml.example` - Template for Google Cloud Build configuration
- `wasm-filters/` - WebAssembly filters for request processing
- `docker/` - Additional Docker configurations

## Building Locally

```bash
docker build -t apx-edge:local -f edge/Dockerfile .
```

## Running Locally

```bash
docker run -p 8080:8080 -p 8443:8443 apx-edge:local
```
