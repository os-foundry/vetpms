package handlers

import (
	"context"
	"net/http"

	"github.com/os-foundry/vetpms/internal/platform/database"
	"github.com/os-foundry/vetpms/internal/platform/web"
	"go.opencensus.io/trace"
)

// Check provides support for orchestration health checks.
type Check struct {
	checks []database.StatusChecker

	// ADD OTHER STATE LIKE THE LOGGER IF NEEDED.
}

// Health validates the service is healthy and ready to accept requests.
func (c *Check) Health(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Check.Health")
	defer span.End()

	var health struct {
		Status string `json:"status"`
	}

	// Perform the checks
	for _, check := range c.checks {
		if err := check.StatusCheck(ctx); err != nil {
			// If the database is not ready we will tell the client and use a 500
			// status. Do not respond by just returning an error because further up in
			// the call stack will interpret that as an unhandled error.
			health.Status = "db not ready"
			return web.Respond(ctx, w, health, http.StatusInternalServerError)
		}
	}

	health.Status = "ok"
	return web.Respond(ctx, w, health, http.StatusOK)
}
