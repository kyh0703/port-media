package health

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

//go:generate go tool counterfeiter -generate

const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusFailed   = "failed"
)

type DependencyCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Result struct {
	Status string                     `json:"status"`
	Checks map[string]DependencyCheck `json:"checks"`
}

//counterfeiter:generate . Checker
type Checker interface {
	Check(ctx context.Context) Result
}

type checker struct {
	db    *bun.DB
	redis *redis.Client
}

func NewChecker(db *bun.DB, redis *redis.Client) Checker {
	return &checker{
		db:    db,
		redis: redis,
	}
}

func (c *checker) Check(ctx context.Context) Result {
	checks := map[string]DependencyCheck{
		"postgres": c.checkPostgres(ctx),
		"redis":    c.checkRedis(ctx),
	}

	status := StatusOK
	for _, check := range checks {
		if check.Status != StatusOK {
			status = StatusDegraded
			break
		}
	}

	return Result{
		Status: status,
		Checks: checks,
	}
}

func (c *checker) checkPostgres(ctx context.Context) DependencyCheck {
	if _, err := c.db.ExecContext(ctx, "SELECT 1"); err != nil {
		return DependencyCheck{Status: StatusFailed, Error: err.Error()}
	}
	return DependencyCheck{Status: StatusOK}
}

func (c *checker) checkRedis(ctx context.Context) DependencyCheck {
	if err := c.redis.Ping(ctx).Err(); err != nil {
		return DependencyCheck{Status: StatusFailed, Error: err.Error()}
	}
	return DependencyCheck{Status: StatusOK}
}
