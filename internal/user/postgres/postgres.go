package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/user"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/bcrypt"
)

const usersCollection = "users"

// Postgres implements the Storage interface for
// the postgres database
type Postgres struct {
	DB *sqlx.DB
}

// List retrieves a list of existing users from the database.
func (st Postgres) List(ctx context.Context) ([]user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.List")
	defer span.End()

	users := []user.User{}
	const q = `SELECT * FROM users`

	if err := st.DB.SelectContext(ctx, &users, q); err != nil {
		return nil, errors.Wrap(err, "selecting users")
	}

	return users, nil
}

// Retrieve gets the specified user from the database.
func (st Postgres) Retrieve(ctx context.Context, claims auth.Claims, id string) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, user.ErrInvalidID
	}

	// If you are not an admin and looking to retrieve someone else then you are rejected.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != id {
		return nil, user.ErrForbidden
	}

	var u user.User
	const q = `SELECT * FROM users WHERE user_id = $1`
	if err := st.DB.GetContext(ctx, &u, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrNotFound
		}

		return nil, errors.Wrapf(err, "selecting user %q", id)
	}

	return &u, nil
}

// Create inserts a new user into the database.
func (st Postgres) Create(ctx context.Context, n user.NewUser, now time.Time) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.Create")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(n.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, "generating password hash")
	}

	u := user.User{
		ID:           uuid.New().String(),
		Name:         n.Name,
		Email:        n.Email,
		PasswordHash: hash,
		Roles:        n.Roles,
		DateCreated:  now.UTC(),
		DateUpdated:  now.UTC(),
	}

	const q = `INSERT INTO users
		(user_id, name, email, password_hash, roles, date_created, date_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = st.DB.ExecContext(
		ctx, q,
		u.ID, u.Name, u.Email,
		u.PasswordHash, u.Roles,
		u.DateCreated, u.DateUpdated,
	)
	if err != nil {
		return nil, errors.Wrap(err, "inserting user")
	}

	return &u, nil
}

// Update replaces a user document in the database.
func (st Postgres) Update(ctx context.Context, claims auth.Claims, id string, upd user.UpdateUser, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.Update")
	defer span.End()

	u, err := st.Retrieve(ctx, claims, id)
	if err != nil {
		return err
	}

	if upd.Name != nil {
		u.Name = *upd.Name
	}
	if upd.Email != nil {
		u.Email = *upd.Email
	}
	if upd.Roles != nil {
		u.Roles = upd.Roles
	}
	if upd.Password != nil {
		pw, err := bcrypt.GenerateFromPassword([]byte(*upd.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "generating password hash")
		}
		u.PasswordHash = pw
	}

	u.DateUpdated = now

	const q = `UPDATE users SET
		"name" = $2,
		"email" = $3,
		"roles" = $4,
		"password_hash" = $5,
		"date_updated" = $6
		WHERE user_id = $1`
	_, err = st.DB.ExecContext(ctx, q, id,
		u.Name, u.Email, u.Roles,
		u.PasswordHash, u.DateUpdated,
	)
	if err != nil {
		return errors.Wrap(err, "updating user")
	}

	return nil
}

// Delete removes a user from the database.
func (st Postgres) Delete(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return user.ErrInvalidID
	}

	const q = `DELETE FROM users WHERE user_id = $1`

	if _, err := st.DB.ExecContext(ctx, q, id); err != nil {
		return errors.Wrapf(err, "deleting user %s", id)
	}

	return nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims value representing this user. The claims can be
// used to generate a token for future authentication.
func (st Postgres) Authenticate(ctx context.Context, now time.Time, email, password string) (auth.Claims, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.Authenticate")
	defer span.End()

	const q = `SELECT * FROM users WHERE email = $1`

	var u user.User
	if err := st.DB.GetContext(ctx, &u, q, email); err != nil {

		// Normally we would return ErrNotFound in this scenario but we do not want
		// to leak to an unauthenticated user which emails are in the system.
		if err == sql.ErrNoRows {
			return auth.Claims{}, user.ErrAuthenticationFailure
		}

		return auth.Claims{}, errors.Wrap(err, "selecting single user")
	}

	// Compare the provided password with the saved hash. Use the bcrypt
	// comparison function so it is cryptographically secure.
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return auth.Claims{}, user.ErrAuthenticationFailure
	}

	// If we are this far the request is valid. Create some claims for the user
	// and generate their token.
	claims := auth.NewClaims(u.ID, u.Roles, now, time.Hour)
	return claims, nil
}

// StatusCheck returns nil if it can successfully talk to the database. It
// returns a non-nil error otherwise.
func (st Postgres) StatusCheck(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.postgres.StatusCheck")
	defer span.End()

	// Run a simple query to determine connectivity. The db has a "Ping" method
	// but it can false-positive when it was previously able to talk to the
	// database but the database has since gone away. Running this query forces a
	// round trip to the database.
	const q = `SELECT true`
	var tmp bool
	return st.DB.QueryRowContext(ctx, q).Scan(&tmp)
}
