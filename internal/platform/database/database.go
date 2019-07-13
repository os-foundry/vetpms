package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // The database driver in use.
	"github.com/pkg/errors"
)

// Config is the required properties to use the database.
type Config struct {
	User       string
	Password   string
	Host       string
	Name       string
	DisableTLS bool
}

// Open knows how to open a database connection based on the configuration.
func Open(cfg Config) (*sqlx.DB, error) {

	// Define SSL mode.
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	// Query parameters.
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	// Construct url.
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	return sqlx.Open("postgres", u.String())
}

// StatusChecker is an interface for types with a database connection,
// for example a Storage interface. A StatusChecker should return
// nil if it can successfully talk to the database or a non-nil error
// otherwise.
type StatusChecker interface {
	StatusCheck(ctx context.Context) error
}

// CheckAndPrepareBolt checks if the filepath is valid and creates it
// on the system.
func CheckAndPrepareBolt(file string, perm os.FileMode) error {
	absp, err := filepath.Abs(file)
	if err != nil {
		return errors.Wrap(err, "getting absolute filepath")
	}

	dir, file := path.Split(absp)
	if dir == "" || file == "" {
		return fmt.Errorf("%s is an invalid filepath", file)
	}

	// Create the path. Continue if the path already exists.
	if err := os.MkdirAll(dir, perm); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "creating %s", dir)
	}
	return nil
}
