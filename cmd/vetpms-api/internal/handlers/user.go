package handlers

import (
	"context"
	"net/http"

	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/web"
	"github.com/os-foundry/vetpms/internal/user"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// User represents the User API method handler set.
type User struct {
	// db            *sqlx.DB
	st            user.Storage
	authenticator *auth.Authenticator

	// ADD OTHER STATE LIKE THE LOGGER AND CONFIG HERE.
}

// List returns all the existing users in the system.
func (u *User) List(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.List")
	defer span.End()

	usrs, err := u.st.List(ctx)
	if err != nil {
		return err
	}

	return web.Respond(ctx, w, usrs, http.StatusOK)
}

// Retrieve returns the specified user from the system.
func (u *User) Retrieve(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.Retrieve")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return errors.New("claims missing from context")
	}

	// Set the id to the provided id in params or to the current user
	id := params["id"]
	if id == "" {
		id = claims.Subject
	}

	usr, err := u.st.Retrieve(ctx, claims, id)
	if err != nil {
		switch err {
		case user.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case user.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case user.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "Id: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, usr, http.StatusOK)
}

// Create inserts a new user into the system.
func (u *User) Create(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.Create")
	defer span.End()

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var nu user.NewUser
	if err := web.Decode(r, &nu); err != nil {
		return errors.Wrap(err, "")
	}

	usr, err := u.st.Create(ctx, nu, v.Now)
	if err != nil {
		return errors.Wrapf(err, "User: %+v", &usr)
	}

	return web.Respond(ctx, w, usr, http.StatusCreated)
}

// Update updates the specified user in the system.
func (u *User) Update(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.Update")
	defer span.End()

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return errors.New("claims missing from context")
	}

	var upd user.UpdateUser
	if err := web.Decode(r, &upd); err != nil {
		return errors.Wrap(err, "")
	}

	err := u.st.Update(ctx, claims, params["id"], upd, v.Now)
	if err != nil {
		switch err {
		case user.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case user.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case user.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "ID: %s  User: %+v", params["id"], &upd)
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}

// Delete removes the specified user from the system.
func (u *User) Delete(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.Delete")
	defer span.End()

	err := u.st.Delete(ctx, params["id"])
	if err != nil {
		switch err {
		case user.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case user.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case user.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "Id: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}

// Token handles a request to authenticate a user. It expects a request using
// Basic Auth with a user's email and password. It responds with a JWT.
func (u *User) Token(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.User.Token")
	defer span.End()

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	email, pass, ok := r.BasicAuth()
	if !ok {
		err := errors.New("must provide email and password in Basic auth")
		return web.NewRequestError(err, http.StatusUnauthorized)
	}

	claims, err := u.st.Authenticate(ctx, v.Now, email, pass)
	if err != nil {
		switch err {
		case user.ErrAuthenticationFailure:
			return web.NewRequestError(err, http.StatusUnauthorized)
		default:
			return errors.Wrap(err, "authenticating")
		}
	}

	var tkn struct {
		Token string `json:"token"`
	}
	tkn.Token, err = u.authenticator.GenerateToken(claims)
	if err != nil {
		return errors.Wrap(err, "generating token")
	}

	return web.Respond(ctx, w, tkn, http.StatusOK)
}
