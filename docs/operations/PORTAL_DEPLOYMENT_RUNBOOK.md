# APX Portal Deployment Runbook

## Overview
This runbook provides step-by-step instructions for deploying, managing, and troubleshooting the APX Portal in production environments.

## Table of Contents
1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Deployment Procedures](#deployment-procedures)
3. [Rollback Procedures](#rollback-procedures)
4. [Monitoring and Health Checks](#monitoring-and-health-checks)
5. [Troubleshooting](#troubleshooting)
6. [Common Issues](#common-issues)
7. [On-Call Procedures](#on-call-procedures)

---

## Pre-Deployment Checklist

### Infrastructure Readiness
- [ ] GCP project configured with billing enabled
- [ ] Required APIs enabled (Cloud Run, GKE, Container Registry, etc.)
- [ ] Service accounts created with appropriate IAM roles
- [ ] VPC network and connectors configured
- [ ] Firestore database provisioned
- [ ] Pub/Sub topics created
- [ ] Secrets configured in Secret Manager

### Application Readiness
- [ ] Docker image built and pushed to Artifact Registry
- [ ] Environment variables configured (ConfigMap/Secrets)
- [ ] Health check endpoint `/api/health` functional
- [ ] Database migrations completed (if applicable)
- [ ] SSL certificates provisioned

### Testing
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] E2E tests passing in staging
- [ ] Load testing completed
- [ ] Security scan completed

---

## Deployment Procedures

### Cloud Run Deployment

#### Staging Deployment
```bash
# 1. Navigate to infrastructure directory
cd .private/infra/cloudrun

# 2. Set environment variables
export PROJECT_ID=your-project-id
export REGION=us-central1
export ENVIRONMENT=staging
export DOMAIN=staging.yourdomain.com

# 3. Deploy to staging
./deploy-cloudrun-complete.sh \
  --project-id $PROJECT_ID \
  --region $REGION \
  --environment staging \
  --domain $DOMAIN

# 4. Verify deployment
curl https://staging.yourdomain.com/api/health
```

#### Production Deployment
```bash
# 1. Set production environment variables
export ENVIRONMENT=production
export DOMAIN=yourdomain.com

# 2. Build and push image
docker build \
  --platform linux/amd64 \
  -t $REGION-docker.pkg.dev/$PROJECT_ID/apx-containers/portal:latest \
  -f .private/portal/Dockerfile \
  .private/portal/

docker push $REGION-docker.pkg.dev/$PROJECT_ID/apx-containers/portal:latest

# 3. Deploy service
envsubst < .private/infra/cloudrun/portal-service.yaml | \
  gcloud run services replace - --region=$REGION

# 4. Verify deployment
gcloud run services describe apx-portal --region=$REGION
curl https://yourdomain.com/api/health
```

### GKE Deployment

#### Staging Deployment
```bash
# 1. Navigate to GKE infrastructure
cd .private/infra/gke

# 2. Deploy to staging cluster
./deploy-gke-complete.sh \
  --project-id $PROJECT_ID \
  --region $REGION \
  --cluster-name apx-cluster-staging

# 3. Verify pods
kubectl get pods -n apx -l app=apx-portal
kubectl logs -n apx -l app=apx-portal --tail=50
```

#### Production Deployment
```bash
# 1. Deploy to production cluster
./deploy-gke-complete.sh \
  --project-id $PROJECT_ID \
  --cluster-name apx-cluster-prod \
  --with-loadbalancer \
  --domain yourdomain.com

# 2. Monitor rollout
kubectl rollout status deployment/apx-portal -n apx

# 3. Verify service
kubectl port-forward -n apx svc/apx-portal 3000:80
curl http://localhost:3000/api/health
```

---

## Rollback Procedures

### Cloud Run Rollback
```bash
# 1. List recent revisions
gcloud run revisions list \
  --service=apx-portal \
  --region=$REGION \
  --limit=5

# 2. Rollback to specific revision
gcloud run services update-traffic apx-portal \
  --region=$REGION \
  --to-revisions=apx-portal-REVISION_NAME=100

# 3. Verify rollback
gcloud run services describe apx-portal --region=$REGION
```

### GKE Rollback
```bash
# 1. View rollout history
kubectl rollout history deployment/apx-portal -n apx

# 2. Rollback to previous revision
kubectl rollout undo deployment/apx-portal -n apx

# 3. Rollback to specific revision
kubectl rollout undo deployment/apx-portal -n apx --to-revision=2

# 4. Verify rollback
kubectl rollout status deployment/apx-portal -n apx
kubectl get pods -n apx -l app=apx-portal
```

### Emergency Rollback
```bash
# If portal is completely broken, scale down immediately
kubectl scale deployment/apx-portal -n apx --replicas=0

# Investigate issue
kubectl logs -n apx -l app=apx-portal --previous

# Scale back up when ready
kubectl scale deployment/apx-portal -n apx --replicas=3
```

---

## Monitoring and Health Checks

### Health Check Endpoints
- **Application Health**: `GET /api/health`
  - Returns: `{"status": "ok", "timestamp": "..."}`
  - Expected: 200 OK

- **Readiness**: Kubernetes readiness probe checks `/api/health`
- **Liveness**: Kubernetes liveness probe checks `/api/health`

### Key Metrics to Monitor

#### Application Metrics
- Request latency (p50, p95, p99)
- Error rate (4xx, 5xx responses)
- Request throughput (requests/second)
- Active connections/sessions

#### Infrastructure Metrics
- CPU utilization (target: <70%)
- Memory utilization (target: <80%)
- Pod/instance count
- Network I/O

#### Business Metrics
- User logins per minute
- API key creations
- Policy deployments
- Active sessions

### Cloud Monitoring Queries
```sql
-- High error rate
fetch cloud_run_revision
| metric 'run.googleapis.com/request_count'
| filter resource.service_name == 'apx-portal'
| filter metric.response_code_class != '2xx'
| group_by 1m, [value_request_count_aggregate: aggregate(value.request_count)]

-- High latency
fetch cloud_run_revision
| metric 'run.googleapis.com/request_latencies'
| filter resource.service_name == 'apx-portal'
| group_by 1m, [value_latencies_percentile: percentile(value.latencies, 99)]
```

---

## Troubleshooting

### Portal Not Starting

**Symptoms**: Pods in CrashLoopBackOff, Cloud Run service not ready

**Steps**:
1. Check logs:
   ```bash
   # GKE
   kubectl logs -n apx -l app=apx-portal --tail=100

   # Cloud Run
   gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=apx-portal" --limit=50
   ```

2. Verify environment variables:
   ```bash
   # GKE
   kubectl get configmap portal-config -n apx -o yaml
   kubectl get secret portal-secrets -n apx

   # Cloud Run
   gcloud run services describe apx-portal --region=$REGION --format=yaml
   ```

3. Check health endpoint locally:
   ```bash
   kubectl port-forward -n apx svc/apx-portal 3000:80
   curl http://localhost:3000/api/health
   ```

### High Error Rate

**Symptoms**: Spike in 500 errors, alerts firing

**Steps**:
1. Check application logs for exceptions
2. Verify backend API connectivity
3. Check database/Firestore connection
4. Review recent deployments or changes

### High Latency

**Symptoms**: Slow response times, timeouts

**Steps**:
1. Check CPU/memory usage
2. Scale up if needed:
   ```bash
   kubectl scale deployment/apx-portal -n apx --replicas=10
   ```
3. Review slow queries/operations
4. Check backend API latency

### Authentication Issues

**Symptoms**: Users cannot log in, session errors

**Steps**:
1. Verify NEXTAUTH_SECRET is configured
2. Check NEXTAUTH_URL matches domain
3. Verify Firestore connectivity
4. Review session logs

---

## Common Issues

### Issue: Build Failures

**Problem**: Docker build fails during deployment

**Solutions**:
- Check Dockerfile syntax
- Ensure all dependencies in package.json
- Verify Node.js version compatibility
- Clear Docker cache: `docker builder prune`

### Issue: Environment Variable Mismatch

**Problem**: Features not working in production

**Solutions**:
- Compare staging vs production env vars
- Check Secret Manager for missing secrets
- Verify ConfigMap/Secret mounted correctly
- Restart pods after env var changes

### Issue: SSL Certificate Issues

**Problem**: HTTPS not working, certificate errors

**Solutions**:
- Wait 10-30 minutes for certificate provisioning
- Verify domain DNS points to correct IP
- Check ManagedCertificate status (GKE):
  ```bash
  kubectl describe managedcertificate apx-ssl-cert -n apx
  ```

### Issue: Database Connection Failures

**Problem**: Cannot connect to Firestore/database

**Solutions**:
- Verify service account has correct IAM roles
- Check VPC connector configuration (Cloud Run)
- Verify Firestore database exists
- Check network policies (GKE)

---

## On-Call Procedures

### Severity Levels

**P0 - Critical (Service Down)**
- Portal completely unavailable
- Data loss occurring
- Security breach detected
- **Response time**: 15 minutes
- **Action**: Page on-call engineer immediately

**P1 - High (Degraded Service)**
- High error rate (>5%)
- Significant latency increase (>2x normal)
- Authentication failures
- **Response time**: 30 minutes
- **Action**: Notify on-call engineer

**P2 - Medium (Minor Issues)**
- Intermittent errors
- Non-critical feature broken
- Performance degradation (<2x normal)
- **Response time**: 2 hours
- **Action**: Create ticket

**P3 - Low (Informational)**
- Minor UI issues
- Documentation requests
- Feature requests
- **Response time**: Next business day
- **Action**: Create backlog item

### Alert Response Playbook

#### Alert: Portal Service Down
1. Check Cloud Run/GKE service status
2. Review recent deployments
3. Check application logs
4. Initiate rollback if needed
5. Notify stakeholders

#### Alert: High Error Rate
1. Identify error types from logs
2. Check backend API health
3. Scale up if resource constrained
4. Review recent code changes
5. Rollback if caused by recent deployment

#### Alert: High Latency
1. Check pod/instance CPU and memory
2. Scale up resources:
   ```bash
   kubectl scale deployment/apx-portal -n apx --replicas=10
   ```
3. Identify slow endpoints
4. Check database query performance
5. Enable caching if applicable

### Escalation Path
1. **On-Call Engineer** → Investigate and resolve
2. **Team Lead** → If issue persists >1 hour
3. **Engineering Manager** → If critical and unresolved >2 hours
4. **CTO** → If business impact significant

### Communication During Incidents
- Update status page every 30 minutes
- Notify #incidents Slack channel
- Email stakeholders for P0/P1 issues
- Post-mortem within 48 hours of resolution

---

## Post-Deployment Verification

### Verification Checklist
- [ ] Health endpoint returns 200 OK
- [ ] User can log in successfully
- [ ] Dashboard loads correctly
- [ ] API calls work (create API key, etc.)
- [ ] WebSocket connections established
- [ ] Monitoring dashboards show normal metrics
- [ ] No error spikes in logs
- [ ] SSL certificate valid
- [ ] Auto-scaling works correctly

### Smoke Tests
```bash
# Health check
curl https://yourdomain.com/api/health

# Login page loads
curl -I https://yourdomain.com/ | grep "200 OK"

# API endpoint accessible
curl https://yourdomain.com/api/auth/session
```

---

## Contacts

### On-Call Schedule
- Check PagerDuty for current on-call engineer
- Slack: #apx-oncall
- Email: oncall@company.com

### Support Channels
- **Urgent (P0/P1)**: Page via PagerDuty
- **Non-urgent**: Slack #apx-support
- **Documentation**: https://docs.apx.com

---

## Appendix

### Useful Commands

```bash
# View portal logs (last 1 hour)
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=apx-portal" --limit=100 --format=json | jq -r '.[] | "\(.timestamp) \(.textPayload)"'

# Check pod resources
kubectl top pods -n apx -l app=apx-portal

# Port forward for debugging
kubectl port-forward -n apx deployment/apx-portal 3000:3000

# Execute command in pod
kubectl exec -it -n apx deployment/apx-portal -- /bin/sh

# View recent events
kubectl get events -n apx --sort-by='.lastTimestamp' | grep portal
```

### Links
- [Cloud Run Console](https://console.cloud.google.com/run)
- [GKE Console](https://console.cloud.google.com/kubernetes)
- [Cloud Monitoring](https://console.cloud.google.com/monitoring)
- [Error Reporting](https://console.cloud.google.com/errors)
- [Cloud Logging](https://console.cloud.google.com/logs)

---

**Last Updated**: 2025-11-13
**Maintained By**: Infrastructure Team
**Review Frequency**: Monthly
