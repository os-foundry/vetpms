package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/os-foundry/vetpms/internal/mid"
	"github.com/os-foundry/vetpms/internal/platform/auth" // Import is removed in final PR
	"github.com/os-foundry/vetpms/internal/platform/database"
	"github.com/os-foundry/vetpms/internal/platform/web"
	"github.com/os-foundry/vetpms/internal/product"
	"github.com/os-foundry/vetpms/internal/user"
)

// API constructs an http.Handler with all application routes defined.
func API(shutdown chan os.Signal, log *log.Logger, u user.Storage, p product.Storage, authenticator *auth.Authenticator) http.Handler {

	// Construct the web.App which holds all routes as well as common Middleware.
	app := web.NewApp(shutdown, log, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics(log))

	// Register health check endpoint. This route is not authenticated.
	check := Check{
		checks: []database.StatusChecker{u},
	}
	app.Handle("GET", "/v1/health", check.Health)

	// Register user management and authentication endpoints.
	uh := User{
		st:            u,
		authenticator: authenticator,
	}

	app.Handle("GET", "/v1/users", uh.List, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))
	app.Handle("POST", "/v1/users", uh.Create, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))
	app.Handle("GET", "/v1/user", uh.Retrieve, mid.Authenticate(authenticator))
	app.Handle("GET", "/v1/users/:id", uh.Retrieve, mid.Authenticate(authenticator))
	app.Handle("PUT", "/v1/users/:id", uh.Update, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))
	app.Handle("DELETE", "/v1/users/:id", uh.Delete, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))

	// This route is not authenticated
	app.Handle("GET", "/v1/users/token", uh.Token)

	// Register product and sale endpoints.
	ph := Product{
		st: p,
	}
	app.Handle("GET", "/v1/products", ph.List, mid.Authenticate(authenticator))
	app.Handle("POST", "/v1/products", ph.Create, mid.Authenticate(authenticator))
	app.Handle("GET", "/v1/products/:id", ph.Retrieve, mid.Authenticate(authenticator))
	app.Handle("PUT", "/v1/products/:id", ph.Update, mid.Authenticate(authenticator))
	app.Handle("DELETE", "/v1/products/:id", ph.Delete, mid.Authenticate(authenticator))

	return app
}
