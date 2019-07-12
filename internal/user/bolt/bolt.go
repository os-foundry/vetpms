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

// List retrieves a list of existing users from the database.
func List(ctx context.Context, db *bolt.DB) ([]user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.List")
	defer span.End()

	users := []user.User{}
	if err := db.View(func(tx *bolt.Tx) error {
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
func Retrieve(ctx context.Context, claims auth.Claims, db *bolt.DB, id string) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, user.ErrInvalidID
	}

	// If you are not an admin and looking to retrieve someone else then you are rejected.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != id {
		return nil, user.ErrForbidden
	}

	var u user.User
	if err := db.View(func(tx *bolt.Tx) error {
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
func Create(ctx context.Context, db *bolt.DB, n user.NewUser, now time.Time) (*user.User, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.Create")
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

	if err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(usersCollection))
		if err != nil {
			return errors.Wrap(err, "getting bucket")
		}

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
func Update(ctx context.Context, claims auth.Claims, db *bolt.DB, id string, upd user.UpdateUser, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.Update")
	defer span.End()

	var oldEmail string
	u, err := Retrieve(ctx, claims, db, id)
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

	if err := db.Update(func(tx *bolt.Tx) error {
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
func Delete(ctx context.Context, db *bolt.DB, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.user.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return user.ErrInvalidID
	}

	var u user.User
	if err := db.View(func(tx *bolt.Tx) error {
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
		return err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(usersCollection))

		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}

		if err := bucket.Delete([]byte(u.Email)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "deleting user")
	}

	return nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims value representing this user. The claims can be
// used to generate a token for future authentication.
func Authenticate(ctx context.Context, db *bolt.DB, now time.Time, email, password string) (auth.Claims, error) {
	ctx, span := trace.StartSpan(ctx, "internal.user.Authenticate")
	defer span.End()

	var id string
	if err := db.View(func(tx *bolt.Tx) error {
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
	if err := db.View(func(tx *bolt.Tx) error {
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
