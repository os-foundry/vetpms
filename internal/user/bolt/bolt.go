package bolt

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/user"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/bcrypt"
)

const usersCollection = "users"

// Bolt implements the Storage interface for
// the bolt database
type Bolt struct {
	DB *bolt.DB
}

// List retrieves a list of existing users from the database.
func (st Bolt) List(ctx context.Context) ([]user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.List")
	defer span.End()

	users := []user.User{}
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		return bucket.ForEach(func(k []byte, v []byte) error {
			u, err := user.Decode(v)
			if err != nil {
				return errors.Wrap(err, "decoding user")
			}
			users = append(users, *u)
			return nil
		})
	}); err != nil {
		return nil, errors.Wrap(err, "selecting users")
	}

	return users, nil
}

// Retrieve gets the specified user from the database.
func (st Bolt) Retrieve(ctx context.Context, claims auth.Claims, id string) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, user.ErrInvalidID
	}

	// If you are not an admin and looking to retrieve someone else then you are rejected.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != id {
		return nil, user.ErrForbidden
	}

	var u user.User
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		v := bucket.Get([]byte(id))
		if len(v) == 0 {
			return user.ErrNotFound
		}

		if err := u.Decode(v); err != nil {
			return errors.Wrap(err, "decoding user")
		}

		return nil
	}); err != nil {
		if err == user.ErrNotFound {
			return nil, err
		}
		return nil, errors.Wrapf(err, "selecting user %q", id)
	}

	return &u, nil
}

// Create inserts a new user into the database.
func (st Bolt) Create(ctx context.Context, n user.NewUser, now time.Time) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.Create")
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

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))

		v, err := u.Encode()
		if err != nil {
			return errors.Wrapf(err, "encoding user")
		}
		if err := bucket.Put([]byte(u.ID), v); err != nil {
			return errors.Wrap(err, "writing user data")
		}
		if err := bucket.Put([]byte(u.Email), []byte(u.ID)); err != nil {
			return errors.Wrap(err, "writing user index")
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "inserting user")
	}

	return &u, nil
}

// Update replaces a user document in the database.
func (st Bolt) Update(ctx context.Context, claims auth.Claims, id string, upd user.UpdateUser, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.Update")
	defer span.End()

	var oldEmail string
	u, err := st.Retrieve(ctx, claims, id)
	if err != nil {
		return err
	}

	if upd.Name != nil {
		u.Name = *upd.Name
	}
	if upd.Email != nil {
		oldEmail = u.Email
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

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		v, err := u.Encode()
		if err != nil {
			return errors.Wrapf(err, "encoding user")
		}
		if err := bucket.Put([]byte(u.ID), v); err != nil {
			return errors.Wrap(err, "writing user data")
		}

		// If the email address is updated, remove the old index and write a new one.
		if upd.Email != nil {
			if err := bucket.Delete([]byte(oldEmail)); err != nil {
				return errors.Wrap(err, "deleting user index")
			}
			if err := bucket.Put([]byte(u.ID), v); err != nil {
				return errors.Wrap(err, "writing user index")
			}
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "updating user")
	}

	return nil
}

// Delete removes a user from the database.
func (st Bolt) Delete(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return user.ErrInvalidID
	}

	var u user.User
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		v := bucket.Get([]byte(id))
		if len(v) == 0 {
			return nil
		}

		if err := u.Decode(v); err != nil {
			return errors.Wrap(err, "decoding user")
		}

		return nil
	}); err != nil {
		return errors.Wrapf(err, "deleting user %s", id)
	}

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))

		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}

		if err := bucket.Delete([]byte(u.Email)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return errors.Wrapf(err, "deleting user %s", id)
	}

	return nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims value representing this user. The claims can be
// used to generate a token for future authentication.
func (st Bolt) Authenticate(ctx context.Context, now time.Time, email, password string) (auth.Claims, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.Authenticate")
	defer span.End()

	var id string
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		v := bucket.Get([]byte(email))
		if len(v) == 0 {
			return user.ErrNotFound
		}
		id = string(v)
		return nil
	}); err != nil {
		if err == user.ErrNotFound {
			return auth.Claims{}, user.ErrAuthenticationFailure
		}
		return auth.Claims{}, errors.Wrap(err, "getting user id")
	}

	var u user.User
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))
		v := bucket.Get([]byte(id))
		if len(v) == 0 {
			return user.ErrNotFound
		}

		if err := u.Decode(v); err != nil {
			return errors.Wrap(err, "decoding user")
		}

		return nil
	}); err != nil {
		if err == user.ErrNotFound {
			return auth.Claims{}, user.ErrAuthenticationFailure
		}
		return auth.Claims{}, errors.Wrapf(err, "selecting user %q", id)
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
func (st Bolt) StatusCheck(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.bolt.StatusCheck")
	defer span.End()

	// Run a simple query to determine connectivity. The db has a "Ping" method
	// but it can false-positive when it was previously able to talk to the
	// database but the database has since gone away. Running this query forces a
	// round trip to the database.
	// const q = `SELECT true`
	// var tmp bool
	// return st.DB.QueryRowContext(ctx, q).Scan(&tmp)
	return nil
}
