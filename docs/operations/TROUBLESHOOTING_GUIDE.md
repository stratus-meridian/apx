# Portal Troubleshooting Guide (PI-T4-017)

**Version:** 1.0.0
**Last Updated:** 2025-11-12
**For:** Operations Team, Developers, Support

## Quick Reference

| Issue | Common Cause | Quick Fix |
|-------|-------------|-----------|
| Portal not loading | Service down | Check `kubectl get pods` |
| 500 errors | Backend unreachable | Verify BACKEND_API_URL |
| Auth failing | Session expired | Clear cookies, re-login |
| Slow performance | High traffic | Scale up replicas |
| WebSocket disconnecting | Load balancer timeout | Increase timeout config |

---

## Table of Contents

1. [Common Issues](#common-issues)
2. [Error Messages](#error-messages)
3. [Debugging Steps](#debugging-steps)
4. [Log Analysis](#log-analysis)
5. [Performance Issues](#performance-issues)
6. [Network & Connectivity](#network--connectivity)
7. [Database Issues](#database-issues)
8. [Deployment Issues](#deployment-issues)
9. [Emergency Procedures](#emergency-procedures)
10. [Contact Information](#contact-information)

---

## Common Issues

### 1. Portal Not Loading

**Symptoms:**
- Blank white screen
- "Cannot connect" error
- Timeout errors

**Possible Causes:**
- Service is down
- DNS not resolving
- Load balancer issue
- SSL certificate expired

**Debugging Steps:**

```bash
# Check if portal is running
kubectl get pods -n apx-portal
kubectl logs -n apx-portal deployment/apx-portal --tail=100

# Check service health
curl https://portal.apx.example.com/api/health

# Check DNS resolution
nslookup portal.apx.example.com

# Check SSL certificate
echo | openssl s_client -servername portal.apx.example.com -connect portal.apx.example.com:443 2>/dev/null | openssl x509 -noout -dates
```

**Solutions:**
- Restart pods: `kubectl rollout restart deployment/apx-portal -n apx-portal`
- Check load balancer: `kubectl get ingress -n apx-portal`
- Renew SSL certificate: `certbot renew`

---

### 2. Authentication Failures

**Symptoms:**
- "Unauthorized" errors
- Redirected to login repeatedly
- "Invalid session" messages

**Possible Causes:**
- Session expired
- NextAuth misconfiguration
- Firebase auth issues
- Cookie domain mismatch

**Debugging Steps:**

```bash
# Check NextAuth logs
kubectl logs -n apx-portal deployment/apx-portal | grep "next-auth"

# Verify environment variables
kubectl get configmap apx-portal-config -n apx-portal -o yaml

# Check Firebase connectivity
curl -I https://firestore.googleapis.com
```

**Solutions:**
- Clear browser cookies
- Verify `NEXTAUTH_URL` matches actual URL
- Check `NEXTAUTH_SECRET` is set
- Ensure Firebase credentials are valid

**Browser Console:**
```javascript
// Check session
console.log(document.cookie);

// Clear Next-Auth session
document.cookie.split(";").forEach(c => {
  document.cookie = c.trim().split("=")[0] + "=;expires=Thu, 01 Jan 1970 00:00:00 UTC";
});
```

---

### 3. Slow Performance

**Symptoms:**
- Pages load slowly
- API calls timeout
- Dashboard charts lag

**Possible Causes:**
- High traffic
- Database queries slow
- Memory/CPU constraints
- Network latency

**Debugging Steps:**

```bash
# Check resource usage
kubectl top pods -n apx-portal

# Check HPA status
kubectl get hpa -n apx-portal

# Check for throttling
kubectl get events -n apx-portal --sort-by='.lastTimestamp'

# Check backend response times
kubectl logs -n apx-portal deployment/apx-portal | grep "response_time"
```

**Solutions:**
- Scale up: `kubectl scale deployment apx-portal --replicas=5 -n apx-portal`
- Increase resources: Update deployment YAML with higher limits
- Add caching: Enable Redis cache
- Optimize queries: Check slow query logs

---

### 4. WebSocket Connection Issues

**Symptoms:**
- "WebSocket disconnected" errors
- Real-time updates not working
- Frequent reconnections

**Possible Causes:**
- Load balancer doesn't support WebSocket
- Connection timeout too short
- Network proxy blocking WS
- Too many concurrent connections

**Debugging Steps:**

```bash
# Test WebSocket connection
wscat -c wss://portal.apx.example.com/ws

# Check WebSocket logs
kubectl logs -n apx-portal deployment/websocket-server --tail=100

# Check connection count
kubectl exec -n apx-portal deployment/websocket-server -- ps aux | grep ws
```

**Solutions:**
- Update ingress for WebSocket support:
```yaml
annotations:
  nginx.ingress.kubernetes.io/websocket-services: "websocket-server"
  nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
```
- Scale WebSocket server
- Use sticky sessions

---

### 5. API Key Not Working

**Symptoms:**
- "Invalid API key" errors
- 401 Unauthorized responses
- API key not syncing to router

**Possible Causes:**
- Key not synced to router
- Key revoked
- Key format invalid
- Rate limit exceeded

**Debugging Steps:**

```bash
# Check API key in Firestore
gcloud firestore collections documents get apiKeys --document=<key-id>

# Check router sync status
curl -H "Authorization: Bearer <admin-token>" \
  https://api.apx.example.com/v1/admin/keys/<key-id>

# Test API key
curl -H "X-API-Key: sk_xxxxx" \
  https://api.apx.example.com/v1/models
```

**Solutions:**
- Manually sync key: POST `/api/keys/{id}/sync`
- Regenerate key if corrupted
- Check sync logs: `kubectl logs -n apx-portal deployment/apx-portal | grep "key-sync"`

---

## Error Messages

### Common Error Codes

#### 500 Internal Server Error
**Meaning:** Server-side error
**Check:**
- Application logs
- Database connectivity
- Memory/CPU usage
- Recent deployments

#### 502 Bad Gateway
**Meaning:** Load balancer can't reach backend
**Check:**
- Service is running
- Health checks passing
- Network connectivity
- Firewall rules

#### 503 Service Unavailable
**Meaning:** Service temporarily down
**Check:**
- Pod status
- Resource limits
- Health check failing
- Too many requests

#### 504 Gateway Timeout
**Meaning:** Request took too long
**Check:**
- Backend response time
- Database queries
- Load balancer timeout
- Network latency

---

## Debugging Steps

### Step-by-Step Debugging

#### 1. Check Service Status

```bash
# Pod status
kubectl get pods -n apx-portal

# Service status
kubectl get svc -n apx-portal

# Ingress status
kubectl get ingress -n apx-portal

# Describe pod for details
kubectl describe pod <pod-name> -n apx-portal
```

#### 2. Check Application Logs

```bash
# Recent logs
kubectl logs -n apx-portal deployment/apx-portal --tail=100

# Follow logs
kubectl logs -n apx-portal deployment/apx-portal -f

# Logs from specific container
kubectl logs -n apx-portal <pod-name> -c apx-portal

# Previous logs (if restarted)
kubectl logs -n apx-portal <pod-name> --previous
```

#### 3. Check Backend Connectivity

```bash
# Test from within pod
kubectl exec -it -n apx-portal <pod-name> -- curl http://backend-api/health

# Test DNS resolution
kubectl exec -it -n apx-portal <pod-name> -- nslookup backend-api

# Test network connectivity
kubectl exec -it -n apx-portal <pod-name> -- ping -c 3 backend-api
```

#### 4. Check Database Connectivity

```bash
# Test Firestore
gcloud firestore collections list

# Check Cloud Storage
gsutil ls gs://apx-policies/

# Check BigQuery
bq query "SELECT 1"

# Check Pub/Sub
gcloud pubsub topics list
```

---

## Log Analysis

### Important Log Patterns

#### Error Patterns
```bash
# Find errors
kubectl logs -n apx-portal deployment/apx-portal | grep "ERROR"

# Find critical errors
kubectl logs -n apx-portal deployment/apx-portal | grep "CRITICAL"

# Find failed requests
kubectl logs -n apx-portal deployment/apx-portal | grep "status.*[45][0-9][0-9]"

# Find slow requests (> 1s)
kubectl logs -n apx-portal deployment/apx-portal | grep "response_time.*[0-9][0-9][0-9][0-9]"
```

#### Success Patterns
```bash
# Find successful requests
kubectl logs -n apx-portal deployment/apx-portal | grep "status.*2[0-9][0-9]"

# Find successful deployments
kubectl logs -n apx-portal deployment/apx-portal | grep "deployment.*success"
```

### Log Aggregation

Use Cloud Logging:
```bash
# Query logs
gcloud logging read "resource.type=k8s_container AND resource.labels.namespace_name=apx-portal" --limit 100

# Query errors
gcloud logging read "resource.type=k8s_container AND resource.labels.namespace_name=apx-portal AND severity>=ERROR" --limit 50
```

---

## Performance Issues

### Memory Leaks

**Symptoms:**
- Memory usage increasing over time
- OOMKilled pods
- Slow performance

**Debugging:**
```bash
# Check memory usage
kubectl top pods -n apx-portal

# Check for OOMKilled
kubectl get pods -n apx-portal -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].lastState.terminated.reason}{"\n"}{end}'

# Get memory profile
kubectl exec -n apx-portal <pod-name> -- curl http://localhost:3000/api/debug/memory
```

**Solutions:**
- Increase memory limits
- Fix memory leaks in code
- Restart pods regularly
- Enable memory profiling

### CPU Throttling

**Symptoms:**
- Slow performance
- High CPU usage
- Requests timing out

**Debugging:**
```bash
# Check CPU usage
kubectl top pods -n apx-portal

# Check for throttling
kubectl describe pod <pod-name> -n apx-portal | grep -A 5 "CPU"

# Get CPU profile
kubectl exec -n apx-portal <pod-name> -- curl http://localhost:3000/api/debug/cpu
```

**Solutions:**
- Increase CPU limits
- Scale horizontally
- Optimize code
- Use caching

---

## Emergency Procedures

### Rollback Deployment

```bash
# Check deployment history
kubectl rollout history deployment/apx-portal -n apx-portal

# Rollback to previous version
kubectl rollout undo deployment/apx-portal -n apx-portal

# Rollback to specific revision
kubectl rollout undo deployment/apx-portal -n apx-portal --to-revision=3
```

### Scale Down Traffic

```bash
# Update ingress to maintenance page
kubectl edit ingress apx-portal-ingress -n apx-portal

# Or redirect traffic
kubectl patch ingress apx-portal-ingress -n apx-portal -p '{"spec":{"rules":[{"host":"portal.apx.example.com","http":{"paths":[{"path":"/","pathType":"Prefix","backend":{"service":{"name":"maintenance-page","port":{"number":80}}}}]}}]}}'
```

### Emergency Restart

```bash
# Restart all pods
kubectl rollout restart deployment/apx-portal -n apx-portal

# Delete specific pod (will recreate)
kubectl delete pod <pod-name> -n apx-portal

# Scale to zero and back
kubectl scale deployment apx-portal --replicas=0 -n apx-portal
kubectl scale deployment apx-portal --replicas=3 -n apx-portal
```

---

## Contact Information

### Escalation Path

**Level 1: Self-Service**
- Check this troubleshooting guide
- Review logs and metrics
- Try common fixes

**Level 2: Team Support**
- Slack: #apx-portal-support
- Email: portal-support@example.com
- Response Time: 30 minutes (business hours)

**Level 3: On-Call Engineer**
- PagerDuty: portal-oncall
- Phone: +1-555-PORTAL
- Response Time: 15 minutes (24/7)

**Level 4: Emergency**
- CTO: cto@example.com
- VP Engineering: vp-eng@example.com
- CEO: ceo@example.com

### Useful Links

- **Status Page:** https://status.apx.example.com
- **Monitoring:** https://monitoring.apx.example.com
- **Logs:** https://logs.apx.example.com
- **Wiki:** https://wiki.apx.example.com/portal
- **Runbooks:** https://runbooks.apx.example.com

---

**Last Updated:** 2025-11-12
**Next Review:** 2025-12-12
**Document Owner:** Operations Team
