# Production Deployment Checklist (PI-T4-018)

**Version:** 1.0.0
**Last Updated:** 2025-11-12
**For:** DevOps, Release Engineers, QA

## Overview

Comprehensive checklist for deploying the APX Portal to production. Follow this checklist for every production deployment to ensure consistency and reliability.

---

## Pre-Deployment Checklist

### Code Quality

- [ ] All tests passing (unit, integration, E2E)
- [ ] Code coverage > 80%
- [ ] No critical/high security vulnerabilities
- [ ] Code reviewed and approved
- [ ] Linting passes without errors
- [ ] TypeScript compilation succeeds
- [ ] No console.log statements in production code
- [ ] Environment-specific code properly gated

### Documentation

- [ ] CHANGELOG.md updated with release notes
- [ ] API documentation current
- [ ] Deployment runbook reviewed
- [ ] Configuration changes documented
- [ ] Migration guide created (if needed)
- [ ] Known issues documented

### Dependencies

- [ ] All dependencies up-to-date
- [ ] npm audit shows no vulnerabilities
- [ ] Third-party services available
- [ ] API keys and credentials valid
- [ ] SSL certificates valid (> 30 days remaining)

### Infrastructure

- [ ] Production environment provisioned
- [ ] Database backups verified
- [ ] Disaster recovery plan reviewed
- [ ] Monitoring alerts configured
- [ ] Log aggregation working
- [ ] CDN configured and tested
- [ ] DNS records updated
- [ ] Firewall rules configured

### Configuration

- [ ] Environment variables set correctly
- [ ] Secrets stored in Secret Manager
- [ ] Feature flags configured
- [ ] Rate limits configured
- [ ] CORS origins whitelisted
- [ ] API endpoints verified
- [ ] Webhook URLs configured

### Testing

- [ ] Staging deployment successful
- [ ] Smoke tests passing
- [ ] Integration tests passing
- [ ] Performance benchmarks met
- [ ] Load testing completed
- [ ] Security audit passed
- [ ] Accessibility audit passed

### Communication

- [ ] Deployment scheduled and announced
- [ ] Stakeholders notified
- [ ] Change advisory board approved
- [ ] Customer communication prepared
- [ ] Support team briefed
- [ ] On-call engineer assigned

---

## Deployment Checklist

### Pre-Deployment

- [ ] Create git tag for release
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

- [ ] Backup production database
```bash
./scripts/backup-production.sh
```

- [ ] Enable maintenance mode (if needed)
```bash
kubectl apply -f k8s/maintenance-mode.yaml
```

- [ ] Notify users of upcoming deployment
- [ ] Take snapshot of current metrics

### Deployment

- [ ] Build Docker image
```bash
docker build -t gcr.io/project/apx-portal:v1.0.0 .
```

- [ ] Push to container registry
```bash
docker push gcr.io/project/apx-portal:v1.0.0
```

- [ ] Deploy to production
```bash
kubectl set image deployment/apx-portal apx-portal=gcr.io/project/apx-portal:v1.0.0 -n apx-portal
```

- [ ] Watch deployment progress
```bash
kubectl rollout status deployment/apx-portal -n apx-portal
```

- [ ] Verify pods are running
```bash
kubectl get pods -n apx-portal
```

### Verification

- [ ] Health check endpoint responding
```bash
curl https://portal.apx.example.com/api/health
```

- [ ] Authentication working
- [ ] API endpoints responding
- [ ] WebSocket connections established
- [ ] Database queries working
- [ ] Real-time updates working
- [ ] Static assets loading
- [ ] SSL certificate valid

### Smoke Tests

- [ ] User can log in
- [ ] Dashboard loads correctly
- [ ] API keys can be created
- [ ] Policies can be deployed
- [ ] Analytics data displays
- [ ] Search functionality works
- [ ] Forms submit successfully
- [ ] Notifications display

### Performance

- [ ] Page load time < 3s
- [ ] API response time < 500ms
- [ ] WebSocket latency < 100ms
- [ ] No memory leaks detected
- [ ] CPU usage normal
- [ ] Database query performance normal

### Monitoring

- [ ] Metrics flowing to monitoring system
- [ ] Alerts configured and testing
- [ ] Error rate within acceptable range
- [ ] Request rate as expected
- [ ] Latency within SLA
- [ ] No spike in errors

---

## Post-Deployment Checklist

### Immediate (0-15 minutes)

- [ ] Disable maintenance mode
```bash
kubectl delete -f k8s/maintenance-mode.yaml
```

- [ ] Monitor error logs
```bash
kubectl logs -f deployment/apx-portal -n apx-portal
```

- [ ] Check error rate in monitoring dashboard
- [ ] Verify user traffic resuming
- [ ] Test critical user flows
- [ ] Announce deployment complete

### Short-term (15-60 minutes)

- [ ] Monitor system metrics
- [ ] Check for elevated error rates
- [ ] Verify all services healthy
- [ ] Test edge cases
- [ ] Check third-party integrations
- [ ] Review user feedback
- [ ] Update status page

### Medium-term (1-24 hours)

- [ ] Analyze performance metrics
- [ ] Review error logs
- [ ] Check database performance
- [ ] Monitor resource usage
- [ ] Gather user feedback
- [ ] Document any issues encountered
- [ ] Update post-mortem if needed

### Long-term (1-7 days)

- [ ] Analyze week-over-week metrics
- [ ] Review incident reports
- [ ] Collect team feedback
- [ ] Update documentation
- [ ] Plan improvements
- [ ] Schedule retrospective

---

## Rollback Checklist

### When to Rollback

Rollback immediately if:
- Critical functionality broken
- Data corruption detected
- Security vulnerability exposed
- Error rate > 5%
- User complaints surge
- Performance degradation > 50%

### Rollback Steps

- [ ] Announce rollback decision
- [ ] Enable maintenance mode
- [ ] Identify last good version
```bash
kubectl rollout history deployment/apx-portal -n apx-portal
```

- [ ] Execute rollback
```bash
kubectl rollout undo deployment/apx-portal -n apx-portal
```

- [ ] Verify rollback successful
```bash
kubectl rollout status deployment/apx-portal -n apx-portal
```

- [ ] Run smoke tests
- [ ] Disable maintenance mode
- [ ] Monitor for 30 minutes
- [ ] Notify stakeholders
- [ ] Create incident report
- [ ] Schedule post-mortem

---

## Monitoring Checklist

### Metrics to Watch

**Application Metrics:**
- [ ] Request rate
- [ ] Error rate
- [ ] Response time (p50, p95, p99)
- [ ] Active users
- [ ] API key validations/sec

**Infrastructure Metrics:**
- [ ] CPU usage
- [ ] Memory usage
- [ ] Network throughput
- [ ] Disk I/O
- [ ] Pod count

**Business Metrics:**
- [ ] User signups
- [ ] API key creations
- [ ] Policy deployments
- [ ] Revenue (if applicable)

### Alerts to Configure

- [ ] High error rate (> 1%)
- [ ] High latency (p99 > 2s)
- [ ] Low request rate (traffic drop)
- [ ] Service down
- [ ] Database connection failures
- [ ] Memory usage > 80%
- [ ] CPU usage > 80%
- [ ] Disk usage > 80%
- [ ] SSL certificate expiring (< 30 days)

---

## Security Checklist

### Pre-Deployment Security

- [ ] Security audit completed
- [ ] Penetration testing passed
- [ ] OWASP Top 10 checked
- [ ] Dependencies scanned for vulnerabilities
- [ ] Secrets rotation completed
- [ ] Access control reviewed
- [ ] Encryption verified (in-transit and at-rest)

### Post-Deployment Security

- [ ] Monitor security logs
- [ ] Check for unusual access patterns
- [ ] Verify authentication working
- [ ] Test authorization rules
- [ ] Review firewall logs
- [ ] Check for failed login attempts
- [ ] Verify API rate limiting working

---

## Compliance Checklist

### Data Protection

- [ ] GDPR compliance verified
- [ ] Data retention policies enforced
- [ ] User consent mechanisms working
- [ ] Data export functionality tested
- [ ] Data deletion functionality tested
- [ ] Privacy policy updated

### Audit Trail

- [ ] Audit logs enabled
- [ ] User actions logged
- [ ] System changes logged
- [ ] Security events logged
- [ ] Log retention configured
- [ ] Log access restricted

---

## Emergency Contacts

### On-Call Rotation
- **Primary:** John Doe (john@example.com, +1-555-0001)
- **Secondary:** Jane Smith (jane@example.com, +1-555-0002)
- **Escalation:** CTO (cto@example.com, +1-555-0003)

### Support Channels
- **Slack:** #production-deployments
- **PagerDuty:** portal-oncall
- **Status Page:** https://status.apx.example.com

---

## Sign-Off

### Deployment Approval

- [ ] **QA Lead:** __________________ Date: __________
- [ ] **Tech Lead:** __________________ Date: __________
- [ ] **DevOps:** __________________ Date: __________
- [ ] **Product Manager:** __________________ Date: __________

### Post-Deployment Confirmation

- [ ] **QA Verification:** __________________ Date: __________
- [ ] **Monitoring Confirmed:** __________________ Date: __________
- [ ] **No Critical Issues:** __________________ Date: __________
- [ ] **Sign-off Complete:** __________________ Date: __________

---

## Notes

Add any deployment-specific notes here:

```
Deployment Date: ___________________
Version: ___________________
Deployed By: ___________________
Issues Encountered: ___________________
Resolution: ___________________
```

---

**Last Updated:** 2025-11-12
**Next Review:** 2025-12-12
**Document Owner:** DevOps Team
