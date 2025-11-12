# APX Platform - GKE Deployment Guide

**Version:** 1.0
**Last Updated:** 2025-11-12
**Status:** Production-Ready Alternative to Cloud Run

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Architecture](#architecture)
4. [Pre-Deployment Checklist](#pre-deployment-checklist)
5. [Step 1: Provision GKE Cluster](#step-1-provision-gke-cluster)
6. [Step 2: Build and Push Container Images](#step-2-build-and-push-container-images)
7. [Step 3: Deploy Core Services](#step-3-deploy-core-services)
8. [Step 4: Deploy APX Services](#step-4-deploy-apx-services)
9. [Step 5: Configure Ingress](#step-5-configure-ingress)
10. [Step 6: Enable Observability](#step-6-enable-observability)
11. [Step 7: (Optional) Enable Istio Service Mesh](#step-7-optional-enable-istio-service-mesh)
12. [Validation & Testing](#validation--testing)
13. [Monitoring & Observability](#monitoring--observability)
14. [Troubleshooting](#troubleshooting)
15. [Production Recommendations](#production-recommendations)
16. [Cost Optimization](#cost-optimization)
17. [Rollback Procedures](#rollback-procedures)

---

## Overview

This guide walks through deploying the APX Platform on Google Kubernetes Engine (GKE) Autopilot. This deployment option provides:

- **Full Kubernetes control** - Custom networking, policies, and resource management
- **Istio service mesh** - mTLS, traffic management, advanced observability
- **GPU support** - Specialized hardware for AI workloads
- **Enterprise features** - Sidecar injection, custom operators, multi-tenancy
- **Self-hosted option** - Can run on-premises or in other clouds

### When to Choose GKE vs Cloud Run

| Scenario | Recommended Platform |
|----------|---------------------|
| Standard HTTP API workloads | **Cloud Run** (simpler, cheaper) |
| Need Istio service mesh | **GKE** |
| GPU/TPU requirements | **GKE** |
| Complex network policies | **GKE** |
| Custom Kubernetes operators | **GKE** |
| Self-hosted deployment | **GKE** |
| Zero-to-N scaling | **Cloud Run** |

---

## Prerequisites

### Required Tools

```bash
# GCloud CLI
gcloud version  # Required: >= 450.0.0

# Terraform
terraform version  # Required: >= 1.5.0

# kubectl
kubectl version --client  # Required: >= 1.28

# Docker
docker --version  # Required: >= 24.0
```

### GCP Project Requirements

1. **GCP Project with billing enabled**
   - Project ID: `apx-build-478003` (or your project)
   - Billing account linked

2. **Enable Required APIs**
   ```bash
   gcloud services enable \
     container.googleapis.com \
     compute.googleapis.com \
     pubsub.googleapis.com \
     firestore.googleapis.com \
     cloudtrace.googleapis.com \
     monitoring.googleapis.com \
     logging.googleapis.com \
     artifactregistry.googleapis.com
   ```

3. **IAM Permissions**
   - `roles/container.admin` - Create/manage GKE clusters
   - `roles/iam.serviceAccountAdmin` - Manage service accounts
   - `roles/compute.networkAdmin` - Configure networking
   - `roles/resourcemanager.projectIamAdmin` - Manage IAM bindings

4. **Resource Quotas** (verify sufficient quotas)
   ```bash
   gcloud compute project-info describe --project=apx-build-478003 \
     --format="value(quotas.filter(metric:CPUS))"
   ```
   - CPUs: >= 24
   - In-use addresses: >= 10
   - Persistent disks: >= 500 GB

### Local Environment Setup

```bash
# Set project
gcloud config set project apx-build-478003

# Set region
gcloud config set compute/region us-central1

# Authenticate
gcloud auth login
gcloud auth application-default login
```

---

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Google Cloud Load Balancer              │
│                    (Ingress / Gateway API)                  │
└───────────────────────┬─────────────────────────────────────┘
                        │
         ┌──────────────┴──────────────┐
         │      GKE Autopilot Cluster   │
         │                              │
         │  ┌────────────────────────┐  │
         │  │   APX Router           │  │
         │  │   (2-10 replicas)      │◄─┼─── Workload Identity
         │  │   - HTTP API           │  │
         │  │   - Rate limiting      │  │
         │  │   - Policy evaluation  │  │
         │  └──────┬─────────────────┘  │
         │         │                     │
         │         │ Pub/Sub             │
         │         │                     │
         │  ┌──────▼─────────────────┐  │
         │  │   APX Worker Pool      │  │
         │  │   (3-20 replicas)      │◄─┼─── Workload Identity
         │  │   - Async processing   │  │
         │  │   - Status updates     │  │
         │  └────────────────────────┘  │
         │                              │
         │  ┌────────────────────────┐  │
         │  │   Redis StatefulSet    │  │
         │  │   - Rate limit store   │  │
         │  │   - Status cache       │  │
         │  └────────────────────────┘  │
         │                              │
         │  ┌────────────────────────┐  │
         │  │   OTEL Collector       │  │
         │  │   (DaemonSet)          │◄─┼─── Cloud Trace/Monitoring
         │  │   - Traces             │  │
         │  │   - Metrics            │  │
         │  │   - Logs               │  │
         │  └────────────────────────┘  │
         │                              │
         └──────────────────────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   Cloud Services     │
         │  - Pub/Sub           │
         │  - Firestore         │
         │  - Cloud Trace       │
         │  - Cloud Monitoring  │
         └──────────────────────┘
```

### Namespace Layout

- **apx** - Main application namespace
  - Router pods
  - Worker pods
  - Redis StatefulSet
  - OTEL Collector
  - ConfigMaps & Secrets

---

## Pre-Deployment Checklist

- [ ] GCP project created and billing enabled
- [ ] All required APIs enabled
- [ ] IAM permissions verified
- [ ] Resource quotas sufficient
- [ ] Docker images built and tested locally
- [ ] Environment variables configured
- [ ] Pub/Sub topics and subscriptions created (or will be managed by Terraform)
- [ ] Firestore database created
- [ ] Backup plan established

---

## Step 1: Provision GKE Cluster

### 1.1 Navigate to Infrastructure Directory

```bash
cd .private/infra/gke
```

### 1.2 Review Configuration

Edit `gke_cluster.tf` to customize:

```hcl
variable "project_id" {
  default = "apx-build-478003"  # Your project
}

variable "region" {
  default = "us-central1"  # Your preferred region
}

variable "cluster_name" {
  default = "apx-cluster"  # Your cluster name
}
```

### 1.3 Initialize Terraform

```bash
terraform init
```

Expected output:
```
Initializing the backend...
Initializing provider plugins...
- Finding hashicorp/google versions matching "~> 5.0"...
- Finding hashicorp/kubernetes versions matching "~> 2.23"...
✓ Terraform has been successfully initialized!
```

### 1.4 Plan Cluster Creation

```bash
terraform plan -out=tfplan
```

Review the plan carefully. Expected resources:
- `google_container_cluster.apx_cluster`
- `google_service_account.apx_router`
- `google_service_account.apx_worker`
- IAM bindings (10+)
- `kubernetes_namespace.apx`
- `kubernetes_service_account.apx_router`
- `kubernetes_service_account.apx_worker`

### 1.5 Apply Configuration

```bash
terraform apply tfplan
```

⏱️ **Expected Duration:** 10-15 minutes

### 1.6 Verify Cluster Creation

```bash
# Get cluster credentials
gcloud container clusters get-credentials apx-cluster \
  --region us-central1 \
  --project apx-build-478003

# Verify connectivity
kubectl cluster-info

# Check nodes (Autopilot creates nodes on-demand)
kubectl get nodes

# Verify namespace
kubectl get namespace apx
```

---

## Step 2: Build and Push Container Images

### 2.1 Configure Docker for GCR

```bash
gcloud auth configure-docker gcr.io
```

### 2.2 Build Router Image

```bash
cd /Users/agentsy/APILEE  # Project root

# Build
docker build -t gcr.io/apx-build-478003/apx-router:latest \
  -f router/Dockerfile .

# Test locally (optional)
docker run --rm -p 8081:8081 \
  -e GCP_PROJECT_ID=apx-build-478003 \
  -e REDIS_ADDR=localhost:6379 \
  gcr.io/apx-build-478003/apx-router:latest

# Push to GCR
docker push gcr.io/apx-build-478003/apx-router:latest
```

### 2.3 Build Worker Image

```bash
# Build
docker build -t gcr.io/apx-build-478003/apx-worker-cpu:latest \
  -f workers/cpu-pool/Dockerfile \
  workers/cpu-pool/

# Test locally (optional)
docker run --rm -p 8080:8080 \
  -e GCP_PROJECT_ID=apx-build-478003 \
  -e REDIS_ADDR=localhost:6379 \
  -e PUBSUB_SUBSCRIPTION=apx-workers-us \
  gcr.io/apx-build-478003/apx-worker-cpu:latest

# Push to GCR
docker push gcr.io/apx-build-478003/apx-worker-cpu:latest
```

### 2.4 Verify Images

```bash
gcloud container images list --repository=gcr.io/apx-build-478003

# Should see:
# gcr.io/apx-build-478003/apx-router
# gcr.io/apx-build-478003/apx-worker-cpu
```

---

## Step 3: Deploy Core Services

### 3.1 Deploy Redis

```bash
kubectl apply -f .private/infra/gke/gke_redis_deploy.yaml

# Wait for StatefulSet to be ready
kubectl wait --for=condition=ready pod -l app=redis -n apx --timeout=300s

# Verify
kubectl get statefulset -n apx
kubectl get pods -n apx -l app=redis
```

### 3.2 Test Redis Connection

```bash
# Port-forward
kubectl port-forward -n apx svc/apx-redis 6379:6379 &

# Test (requires redis-cli)
redis-cli -h localhost -p 6379 ping
# Should return: PONG

# Stop port-forward
pkill -f "kubectl port-forward.*apx-redis"
```

### 3.3 Deploy OTEL Collector

```bash
kubectl apply -f .private/infra/gke/gke_otel_collector.yaml

# Wait for DaemonSet
kubectl wait --for=condition=ready pod -l app=otel-collector -n apx --timeout=300s

# Verify
kubectl get daemonset -n apx
kubectl get pods -n apx -l app=otel-collector
```

---

## Step 4: Deploy APX Services

### 4.1 Deploy Router

```bash
kubectl apply -f .private/infra/gke/gke_router_deploy.yaml

# Wait for deployment
kubectl wait --for=condition=available deployment/apx-router -n apx --timeout=300s

# Verify
kubectl get deployment -n apx apx-router
kubectl get pods -n apx -l app=apx-router
kubectl get hpa -n apx apx-router
```

### 4.2 Verify Router Health

```bash
# Port-forward
kubectl port-forward -n apx svc/apx-router 8081:8081

# In another terminal:
curl http://localhost:8081/health
# Expected: {"status":"ok","service":"apx-router"}

curl http://localhost:8081/ready
# Expected: {"status":"ready"}

curl http://localhost:8081/metrics | grep apx_
# Should see Prometheus metrics
```

### 4.3 Deploy Workers

```bash
kubectl apply -f .private/infra/gke/gke_worker_deploy.yaml

# Wait for deployment
kubectl wait --for=condition=available deployment/apx-worker -n apx --timeout=300s

# Verify
kubectl get deployment -n apx apx-worker
kubectl get pods -n apx -l app=apx-worker
kubectl get hpa -n apx apx-worker
```

### 4.4 Verify Worker Health

```bash
# Port-forward
kubectl port-forward -n apx svc/apx-worker 8080:8080

# In another terminal:
curl http://localhost:8080/health
# Expected: {"status":"ok","service":"apx-worker-cpu"}

curl http://localhost:8080/ready
# Expected: {"status":"ready"}
```

### 4.5 Check All Pods

```bash
kubectl get pods -n apx

# Expected output:
# NAME                          READY   STATUS    RESTARTS   AGE
# apx-redis-0                   1/1     Running   0          5m
# apx-router-xxx                1/1     Running   0          3m
# apx-router-yyy                1/1     Running   0          3m
# apx-worker-xxx                1/1     Running   0          2m
# apx-worker-yyy                1/1     Running   0          2m
# apx-worker-zzz                1/1     Running   0          2m
# otel-collector-xxx            1/1     Running   0          4m
```

---

## Step 5: Configure Ingress

### 5.1 Deploy Ingress

```bash
kubectl apply -f .private/infra/gke/gke_ingress.yaml

# Wait for ingress (can take 5-10 minutes)
kubectl get ingress -n apx apx-ingress --watch
```

### 5.2 Get Ingress IP

```bash
INGRESS_IP=$(kubectl get ingress -n apx apx-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Ingress IP: $INGRESS_IP"

# Test via ingress
curl http://$INGRESS_IP/health
```

### 5.3 (Optional) Configure DNS

```bash
# Create DNS A record pointing to ingress IP
# Example: api.apx.dev → $INGRESS_IP

# Update gke_ingress.yaml with your domain:
# spec:
#   tls:
#   - secretName: apx-tls-secret
#     hosts:
#     - api.apx.dev
```

### 5.4 (Optional) Enable HTTPS

```bash
# Option 1: Google-managed certificate
kubectl apply -f - <<EOF
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: apx-managed-cert
  namespace: apx
spec:
  domains:
  - api.apx.dev
EOF

# Option 2: Let's Encrypt with cert-manager
# Install cert-manager first
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

---

## Step 6: Enable Observability

### 6.1 Verify OTEL Collector Exporting

```bash
# Check OTEL collector logs
kubectl logs -n apx -l app=otel-collector --tail=50

# Should see successful exports to Cloud Trace/Monitoring
```

### 6.2 Generate Test Traffic

```bash
# Send test requests
for i in {1..10}; do
  curl -X POST http://$INGRESS_IP/api/test \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: test-tenant" \
    -d '{"test": "data"}'
  sleep 1
done
```

### 6.3 View Traces in Cloud Trace

```bash
# Open Cloud Trace
echo "https://console.cloud.google.com/traces/list?project=apx-build-478003"

# Filter by service: apx-router
```

### 6.4 View Metrics in Cloud Monitoring

```bash
# Open Cloud Monitoring
echo "https://console.cloud.google.com/monitoring?project=apx-build-478003"

# Create dashboard for APX metrics
```

### 6.5 View Logs in Cloud Logging

```bash
# View router logs
gcloud logging read \
  "resource.type=k8s_container AND resource.labels.container_name=router" \
  --limit 50 \
  --format json

# View worker logs
gcloud logging read \
  "resource.type=k8s_container AND resource.labels.container_name=worker" \
  --limit 50 \
  --format json
```

---

## Step 7: (Optional) Enable Istio Service Mesh

### 7.1 Install Istio

```bash
# Option 1: Anthos Service Mesh (recommended for GKE)
gcloud container fleet mesh enable --project apx-build-478003

gcloud container fleet memberships register apx-cluster \
  --gke-cluster us-central1/apx-cluster \
  --enable-workload-identity

# Option 2: Open Source Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH
istioctl install --set profile=default -y
```

### 7.2 Enable Sidecar Injection

```bash
kubectl label namespace apx istio-injection=enabled --overwrite

# Restart pods to inject sidecars
kubectl rollout restart deployment -n apx
```

### 7.3 Apply Istio Configuration

```bash
kubectl apply -f .private/infra/gke/istio.yaml

# Verify
kubectl get gateway -n apx
kubectl get virtualservice -n apx
kubectl get destinationrule -n apx
kubectl get peerauthentication -n apx
```

### 7.4 Verify mTLS

```bash
# Check mTLS status
istioctl authn tls-check apx-router-xxx.apx -n apx

# Should show STRICT mode enabled
```

---

## Validation & Testing

### Functional Tests

```bash
cd /Users/agentsy/APILEE

# 1. Health checks
curl http://$INGRESS_IP/health
curl http://$INGRESS_IP/ready

# 2. Router API test
REQUEST_ID=$(uuidgen)
curl -X POST http://$INGRESS_IP/api/test \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $REQUEST_ID" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{"data": "test"}'

# Should return 202 Accepted with status URL

# 3. Check status
curl http://$INGRESS_IP/status/$REQUEST_ID

# 4. Verify Pub/Sub message consumed
kubectl logs -n apx -l app=apx-worker --tail=20 | grep $REQUEST_ID

# 5. Check Redis status storage
kubectl exec -n apx apx-redis-0 -- redis-cli GET "status:$REQUEST_ID"
```

### Performance Tests

```bash
# Install k6 if not already installed
# brew install k6  # macOS
# or download from https://k6.io/

# Run load test
k6 run - <<EOF
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
};

export default function () {
  let res = http.post('http://$INGRESS_IP/api/test',
    JSON.stringify({ data: 'test' }),
    { headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'test' } }
  );
  check(res, { 'status is 202': (r) => r.status === 202 });
  sleep(1);
}
EOF
```

### Observability Tests

```bash
# 1. Verify traces exported
# Check Cloud Trace console

# 2. Verify metrics exported
# Check Cloud Monitoring console

# 3. Verify logs exported
gcloud logging read \
  "resource.type=k8s_container AND resource.labels.namespace_name=apx" \
  --limit 10

# 4. Check HPA scaling
kubectl get hpa -n apx --watch
# Generate load and observe scaling
```

---

## Monitoring & Observability

### Key Metrics to Monitor

1. **Pod Health**
   ```bash
   kubectl get pods -n apx --watch
   ```

2. **HPA Status**
   ```bash
   kubectl get hpa -n apx
   ```

3. **Resource Usage**
   ```bash
   kubectl top pods -n apx
   kubectl top nodes
   ```

4. **Request Rate**
   - Cloud Monitoring: `kubernetes.io/container/cpu/request_utilization`
   - Custom metrics: `apx_requests_total`

5. **Error Rate**
   - Cloud Logging: Filter by severity ERROR
   - Custom metrics: `apx_requests_failed_total`

6. **Latency (p50, p95, p99)**
   - Cloud Trace: Latency distributions
   - Custom metrics: `apx_request_duration_seconds`

### Recommended Alerts

```yaml
# Example alert policies (apply via Cloud Monitoring)

# High error rate
condition: apx_requests_failed_total / apx_requests_total > 0.05
duration: 5 minutes
notification: PagerDuty / Email

# High latency
condition: apx_request_duration_seconds{quantile="0.95"} > 1.0
duration: 5 minutes

# Pod crashes
condition: kubernetes.io/pod/status/restart_count > 5
duration: 10 minutes

# HPA at max capacity
condition: current_replicas >= max_replicas
duration: 15 minutes
```

---

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n apx

# Common issues:
# 1. ImagePullBackOff - Check image name and GCR permissions
# 2. CrashLoopBackOff - Check logs
# 3. Pending - Check resource quotas

# View logs
kubectl logs <pod-name> -n apx
kubectl logs <pod-name> -n apx --previous  # Previous crash
```

### Service Not Accessible

```bash
# Check service
kubectl get svc -n apx
kubectl describe svc apx-router -n apx

# Check endpoints
kubectl get endpoints -n apx

# Test service connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n apx -- \
  curl http://apx-router.apx.svc.cluster.local:8081/health
```

### Ingress Not Working

```bash
# Check ingress status
kubectl describe ingress apx-ingress -n apx

# Check backend health
kubectl get backendconfig -n apx

# View GCE load balancer status
gcloud compute url-maps list
gcloud compute backend-services list
```

### Workload Identity Issues

```bash
# Verify service account annotation
kubectl get sa apx-router -n apx -o yaml

# Should have:
# metadata:
#   annotations:
#     iam.gke.io/gcp-service-account: apx-router-gke@apx-build-478003.iam.gserviceaccount.com

# Test from pod
kubectl exec -it <router-pod> -n apx -- sh
# Inside pod:
gcloud auth list
# Should show: apx-router-gke@apx-build-478003.iam.gserviceaccount.com
```

### High Memory/CPU Usage

```bash
# Check resource usage
kubectl top pods -n apx

# Adjust resources in deployment YAML:
# resources:
#   requests:
#     cpu: "1000m"
#     memory: "2Gi"
#   limits:
#     cpu: "4000m"
#     memory: "8Gi"

# Apply changes
kubectl apply -f gke_router_deploy.yaml
```

---

## Production Recommendations

### Security

1. **Enable Binary Authorization**
   ```bash
   # Ensure only signed images can run
   gcloud container binauthz policy export > policy.yaml
   # Edit policy.yaml to enforce signatures
   gcloud container binauthz policy import policy.yaml
   ```

2. **Enable Pod Security Standards**
   ```bash
   kubectl label namespace apx \
     pod-security.kubernetes.io/enforce=restricted \
     pod-security.kubernetes.io/audit=restricted \
     pod-security.kubernetes.io/warn=restricted
   ```

3. **Use Memorystore Redis (managed)**
   - Replace in-cluster Redis with Cloud Memorystore
   - Better performance, reliability, and backups

4. **Enable Cloud Armor**
   - DDoS protection
   - WAF rules
   - Rate limiting at edge

5. **Rotate Secrets Regularly**
   ```bash
   # Use Secret Manager
   gcloud secrets create apx-redis-password --data-file=-
   # Reference in deployment
   ```

### Reliability

1. **Multi-Region Deployment**
   - Deploy clusters in multiple regions
   - Use Global Load Balancer
   - Configure failover

2. **Backup Strategy**
   ```bash
   # Backup Redis data
   kubectl exec -n apx apx-redis-0 -- redis-cli BGSAVE

   # Backup cluster configuration
   kubectl get all -n apx -o yaml > apx-backup.yaml
   ```

3. **Disaster Recovery Plan**
   - Document recovery procedures
   - Test recovery regularly
   - Automate with Terraform

### Performance

1. **Use Node Pools with Specific Machine Types**
   - For Autopilot, GKE manages this
   - For Standard GKE, configure node pools:
     ```hcl
     node_pool {
       name = "high-cpu-pool"
       machine_type = "n2-highcpu-8"
       ...
     }
     ```

2. **Enable CDN for Static Content**
   - Configure in BackendConfig

3. **Optimize Container Images**
   ```dockerfile
   # Use multi-stage builds
   # Minimize layers
   # Use .dockerignore
   ```

---

## Cost Optimization

### Right-Size Resources

```bash
# Monitor actual usage
kubectl top pods -n apx --containers

# Adjust requests/limits based on actual usage
# Don't over-provision
```

### Use Preemptible Nodes (Standard GKE only)

```hcl
# For non-critical workloads
node_pool {
  name = "preemptible-pool"
  node_config {
    preemptible = true
    machine_type = "n2-standard-4"
  }
}
```

### Enable Cluster Autoscaling

- Autopilot handles this automatically
- For Standard GKE, configure autoscaling per node pool

### Use Committed Use Discounts

```bash
# Purchase 1-year or 3-year commitments for predictable workloads
# Up to 57% discount
```

### Monitor Costs

```bash
# View GKE costs
gcloud billing projects describe apx-build-478003 \
  --format="value(billingAccountName)"

# Enable cost allocation tracking
kubectl label namespace apx cost-center=apx-platform
```

---

## Rollback Procedures

### Rollback Deployment

```bash
# View rollout history
kubectl rollout history deployment/apx-router -n apx

# Rollback to previous version
kubectl rollout undo deployment/apx-router -n apx

# Rollback to specific revision
kubectl rollout undo deployment/apx-router -n apx --to-revision=2

# Verify
kubectl rollout status deployment/apx-router -n apx
```

### Rollback Cluster Changes

```bash
# Revert Terraform changes
cd .private/infra/gke
terraform plan  # Review changes
terraform apply  # Apply previous state

# Or use specific state
terraform state pull > backup.tfstate
terraform apply -state=backup.tfstate
```

### Emergency Shutdown

```bash
# Scale down all services
kubectl scale deployment --all --replicas=0 -n apx

# Or delete deployments
kubectl delete deployment --all -n apx

# Cluster still runs but no application pods
```

---

## Summary

You have successfully deployed the APX Platform on GKE! 🎉

**Deployed Components:**
- ✅ GKE Autopilot Cluster
- ✅ APX Router (2+ replicas)
- ✅ APX Worker Pool (3+ replicas)
- ✅ Redis StatefulSet
- ✅ OTEL Collector (DaemonSet)
- ✅ Ingress / Load Balancer
- ✅ Observability (Cloud Trace, Monitoring, Logging)
- ✅ (Optional) Istio Service Mesh

**Next Steps:**
1. Configure production domain and HTTPS
2. Set up monitoring dashboards
3. Configure alerting policies
4. Perform load testing
5. Document runbooks
6. Train team on operations

**Support:**
- Documentation: `docs/`
- Terraform: `.private/infra/gke/`
- Kubernetes: `.private/infra/gke/gke_*.yaml`

---

**Version History:**
- v1.0 (2025-11-12) - Initial GKE deployment guide
