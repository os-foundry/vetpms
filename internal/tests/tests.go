package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/database"
	"github.com/os-foundry/vetpms/internal/platform/database/databasetest"
	"github.com/os-foundry/vetpms/internal/platform/web"
	"github.com/os-foundry/vetpms/internal/product"
	boltProduct "github.com/os-foundry/vetpms/internal/product/bolt"
	"github.com/os-foundry/vetpms/internal/schema"
	"github.com/os-foundry/vetpms/internal/user"
	boltUser "github.com/os-foundry/vetpms/internal/user/bolt"

	// boltProduct "github.com/os-foundry/vetpms/internal/product/bolt"
	pqProduct "github.com/os-foundry/vetpms/internal/product/postgres"
	"github.com/os-foundry/vetpms/internal/user/postgres"
	pqUser "github.com/os-foundry/vetpms/internal/user/postgres"
	bolt "go.etcd.io/bbolt"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

// These are the IDs in the seed data for admin@example.com and
// user@example.com.
const (
	AdminID = "5cf37266-3473-4006-984f-9325122678b7"
	UserID  = "45b5fbd3-755f-4379-8f07-a58d4a30fa2f"
)

// NewUnit creates a test database inside a Docker container. It creates the
// required table structure but the database is otherwise empty.
//
// It does not return errors as this intended for testing only. Instead it will
// call Fatal on the provided testing.T if anything goes wrong.
//
// It returns the database to use as well as a function to call at the end of
// the test.
func NewPqUnit(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	c := databasetest.StartContainer(t)

	db, err := database.Open(database.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       c.Host,
		Name:       "postgres",
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("opening database connection: %v", err)
	}

	t.Log("waiting for database to be ready")

	// Wait for the database to be ready. Wait 100ms longer between each attempt.
	// Do not try more than 20 times.
	var pingError error
	maxAttempts := 20
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
	}

	if pingError != nil {
		databasetest.DumpContainerLogs(t, c)
		databasetest.StopContainer(t, c)
		t.Fatalf("waiting for database to be ready: %v", pingError)
	}

	if err := schema.Migrate(db); err != nil {
		databasetest.StopContainer(t, c)
		t.Fatalf("migrating: %s", err)
	}

	// teardown is the function that should be invoked when the caller is done
	// with the database.
	teardown := func() {
		t.Helper()
		db.Close()
		databasetest.StopContainer(t, c)
	}

	return db, teardown
}

// NewUnit creates a test database inside a Docker container. It creates the
// required table structure but the database is otherwise empty.
//
// It does not return errors as this intended for testing only. Instead it will
// call Fatal on the provided testing.T if anything goes wrong.
//
// It returns the database to use as well as a function to call at the end of
// the test.
func NewBoltUnit(t *testing.T) (*bolt.DB, func()) {
	t.Helper()

	db, err := bolt.Open("test.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("opening database connection: %v", err)
	}

	// teardown is the function that should be invoked when the caller is done
	// with the database.
	teardown := func() {
		t.Helper()
		db.Close()
		os.Remove("test.db")
	}

	return db, teardown
}

// NewUserStorageUnit creates user storage connected to a database,
// seeds it and constructs an authenticator.
func NewUserStorageUnit(t *testing.T, tp string) (user.Storage, func()) {
	t.Helper()

	switch tp {
	case "postgres":
		db, teardown := NewPqUnit(t)
		return pqUser.Postgres{db}, teardown
	case "bolt":
		db, teardown := NewBoltUnit(t)
		return boltUser.Bolt{db}, teardown
	}
	t.Fatal("tp should be bolt or postgres")
	return nil, nil
}

// NewProductStorageUnit creates user storage connected to a database,
// seeds it and constructs an authenticator.
func NewProductStorageUnit(t *testing.T, tp string) (product.Storage, func()) {
	t.Helper()

	switch tp {
	case "postgres":
		db, teardown := NewPqUnit(t)
		return pqProduct.Postgres{db}, teardown
	case "bolt":
		db, teardown := NewBoltUnit(t)
		return boltProduct.Bolt{db}, teardown
	}
	t.Fatal("tp should be bolt or postgres")
	return nil, nil
}

// Test owns state for running and shutting down tests.
type Test struct {
	Pq            *sqlx.DB
	Bolt          *bolt.DB
	Log           *log.Logger
	Authenticator *auth.Authenticator

	t       *testing.T
	cleanup func()
}

// NewIntegration creates a database, seeds it, constructs an authenticator.
func NewIntegration(t *testing.T, tp string) *Test {
	t.Helper()

	// Create the logger to use.
	logger := log.New(os.Stdout, "TEST : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// Create RSA keys to enable authentication in our service.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	// Build an authenticator using this static key.
	kid := "4754d86b-7a6d-4df5-9c65-224741361492"
	kf := auth.NewSimpleKeyLookupFunc(kid, key.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(key, kid, "RS256", kf)
	if err != nil {
		t.Fatal(err)
	}

	test := Test{
		Log:           logger,
		Authenticator: authenticator,
		t:             t,
	}

	switch tp {
	case "postgres":
		// Initialize and seed database. Store the cleanup function call later.
		db, cleanup := NewPqUnit(t)
		test.Pq = db
		test.cleanup = cleanup

		if err := schema.Seed(db); err != nil {
			t.Fatal(err)
		}

	case "bolt":
		db, cleanup := NewBoltUnit(t)
		test.Bolt = db
		test.cleanup = cleanup

	default:
		t.Fatal("tp should be postgres or bolt")
	}

	return &test
}

// Teardown releases any resources used for the test.
func (test *Test) Teardown() {
	test.cleanup()
}

// Token generates an authenticated token for a user.
func (test *Test) Token(email, pass string) string {
	test.t.Helper()

	var (
		claims auth.Claims
		err    error
	)

	if test.Pq != nil {
		st := postgres.Postgres{test.Pq}
		claims, err = st.Authenticate(
			context.Background(), time.Now(),
			email, pass,
		)
		if err != nil {
			test.t.Fatal(err)
		}
	}

	if test.Bolt != nil {
		st := boltUser.Bolt{test.Bolt}
		claims, err = st.Authenticate(
			context.Background(), time.Now(),
			email, pass,
		)
		if err != nil {
			test.t.Fatal(err)
		}
	}

	tkn, err := test.Authenticator.GenerateToken(claims)
	if err != nil {
		test.t.Fatal(err)
	}

	return tkn
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := web.Values{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}

	return context.WithValue(context.Background(), web.KeyValues, &values)
}

// StringPointer is a helper to get a *string from a string. It is in the tests
// package because we normally don't want to deal with pointers to basic types
// but it's useful in some tests.
func StringPointer(s string) *string {
	return &s
}

// IntPointer is a helper to get a *int from a int. It is in the tests package
// because we normally don't want to deal with pointers to basic types but it's
// useful in some tests.
func IntPointer(i int) *int {
	return &i
}
