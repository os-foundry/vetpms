package schema

import (
	"fmt"

	"github.com/GuiaBolso/darwin"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	bbolt "go.etcd.io/bbolt"
)

// Migrate attempts to bring the schema for db up to date with the migrations
// defined in this package.
func Migrate(dbi interface{}) error {
	switch dbi.(type) {

	case *sqlx.DB:
		db := dbi.(*sqlx.DB)
		driver := darwin.NewGenericDriver(db.DB, darwin.PostgresDialect{})

		d := darwin.New(driver, migrations, nil)

		return d.Migrate()

	case *bbolt.DB:
		db := dbi.(*bbolt.DB)
		if err := db.Update(func(tx *bbolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists([]byte("users")); err != nil {
				return errors.Wrap(err, "creating bolt user bucket")
			}

			if _, err := tx.CreateBucketIfNotExists([]byte("products")); err != nil {
				return errors.Wrap(err, "creating bolt products bucket")
			}

			if _, err := tx.CreateBucketIfNotExists([]byte("sales")); err != nil {
				return errors.Wrap(err, "creating bolt sales bucket")
			}

			return nil
		}); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unsupported database %T", dbi)

}

// migrations contains the queries needed to construct the database schema.
// Entries should never be removed from this slice once they have been ran in
// production.
//
// Using constants in a .go file is an easy way to ensure the queries are part
// of the compiled executable and avoids pathing issues with the working
// directory. It has the downside that it lacks syntax highlighting and may be
// harder to read for some cases compared to using .sql files. You may also
// consider a combined approach using a tool like packr or go-bindata.
var migrations = []darwin.Migration{
	{
		Version:     1,
		Description: "Add products",
		Script: `
CREATE TABLE products (
	product_id   UUID,
	name         TEXT,
	cost         INT,
	quantity     INT,
	date_created TIMESTAMP,
	date_updated TIMESTAMP,

	PRIMARY KEY (product_id)
);`,
	},
	{
		Version:     2,
		Description: "Add sales",
		Script: `
CREATE TABLE sales (
	sale_id      UUID,
	product_id   UUID,
	quantity     INT,
	paid         INT,
	date_created TIMESTAMP,

	PRIMARY KEY (sale_id),
	FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE CASCADE
);`,
	},
	{
		Version:     3,
		Description: "Add users",
		Script: `
CREATE TABLE users (
	user_id       UUID,
	name          TEXT,
	email         TEXT UNIQUE,
	roles         TEXT[],
	password_hash TEXT,

	date_created TIMESTAMP,
	date_updated TIMESTAMP,

	PRIMARY KEY (user_id)
);`,
	},
	{
		Version:     4,
		Description: "Add user column to products",
		Script: `
ALTER TABLE products
	ADD COLUMN user_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'
`,
	},
}
