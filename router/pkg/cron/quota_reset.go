package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"go.uber.org/zap"
)

// QuotaResetJob handles monthly quota resets for all tenants
type QuotaResetJob struct {
	enforcer *ratelimit.QuotaEnforcer
	logger   *zap.Logger
}

// NewQuotaResetJob creates a new quota reset job
func NewQuotaResetJob(enforcer *ratelimit.QuotaEnforcer, logger *zap.Logger) *QuotaResetJob {
	return &QuotaResetJob{
		enforcer: enforcer,
		logger:   logger,
	}
}

// Run executes the quota reset job
// This should be called on the first day of each month at midnight
func (j *QuotaResetJob) Run(ctx context.Context) error {
	j.logger.Info("starting monthly quota reset job")
	startTime := time.Now()

	// Reset all tenant quotas
	resetCount, err := j.enforcer.ResetAllQuotas(ctx)
	if err != nil {
		j.logger.Error("quota reset job failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))
		return fmt.Errorf("failed to reset quotas: %w", err)
	}

	duration := time.Since(startTime)
	j.logger.Info("monthly quota reset completed",
		zap.Int("tenants_reset", resetCount),
		zap.Duration("duration", duration))

	return nil
}

// Schedule returns the cron schedule expression for this job
// Runs on the 1st day of every month at 00:00 UTC
func (j *QuotaResetJob) Schedule() string {
	return "0 0 1 * *"
}

// Name returns a human-readable name for this job
func (j *QuotaResetJob) Name() string {
	return "quota-reset"
}

// Description returns a description of what this job does
func (j *QuotaResetJob) Description() string {
	return "Resets monthly quotas for all tenants on the first day of each month"
}
