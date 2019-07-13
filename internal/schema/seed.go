package schema

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/os-foundry/vetpms/internal/product"
	"github.com/os-foundry/vetpms/internal/user"
	"github.com/pkg/errors"
	bbolt "go.etcd.io/bbolt"
)

// Seed runs the set of seed-data queries against db. The queries are ran in a
// transaction and rolled back if any fail.
func Seed(dbi interface{}) error {
	switch dbi.(type) {

	case *sqlx.DB:
		db := dbi.(*sqlx.DB)
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(seedsPq); err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}

		return tx.Commit()

	case *bbolt.DB:
		timestamp, err := time.Parse("2006-01-02 15:04:05", "2019-03-24 00:00:00")
		if err != nil {
			return err
		}

		db := dbi.(*bbolt.DB)
		err = db.Update(func(tx *bbolt.Tx) error {
			users, err := tx.CreateBucketIfNotExists([]byte("users"))
			if err != nil {
				return errors.Wrap(err, "creating bolt user bucket")
			}

			{
				a := user.User{
					ID:           "5cf37266-3473-4006-984f-9325122678b7",
					Name:         "Admin Gopher",
					Email:        "admin@example.com",
					Roles:        pq.StringArray{"ADMIN", "USER"},
					PasswordHash: []byte("$2a$10$1ggfMVZV6Js0ybvJufLRUOWHS5f6KneuP0XwwHpJ8L8ipdry9f2/a"),
					DateCreated:  timestamp,
					DateUpdated:  timestamp,
				}

				u := user.User{
					ID:           "45b5fbd3-755f-4379-8f07-a58d4a30fa2f",
					Name:         "User Gopher",
					Email:        "user@example.com",
					Roles:        pq.StringArray{"USER"},
					PasswordHash: []byte("$2a$10$9/XASPKBbJKVfCAZKDH.UuhsuALDr5vVm6VrYA9VFR8rccK86C1hW"),
					DateCreated:  timestamp,
					DateUpdated:  timestamp,
				}

				ab, err := a.Encode()
				if err != nil {
					return err
				}
				users.Put([]byte(a.ID), ab)
				users.Put([]byte(a.Email), []byte(a.ID))

				ub, err := u.Encode()
				if err != nil {
					return err
				}
				users.Put([]byte(u.ID), ub)
				users.Put([]byte(u.Email), []byte(u.ID))
			}

			products, err := tx.CreateBucketIfNotExists([]byte("products"))
			if err != nil {
				return errors.Wrap(err, "creating bolt products bucket")
			}

			{
				p1 := product.Product{
					ID:          "a2b0639f-2cc6-44b8-b97b-15d69dbb511e",
					Name:        "Comic Books",
					Cost:        50,
					Quantity:    41,
					DateCreated: timestamp,
					DateUpdated: timestamp,
				}

				p2 := product.Product{
					ID:          "72f8b983-3eb4-48db-9ed0-e45cc6bd716b",
					Name:        "McDonalds Toys",
					Cost:        75,
					Quantity:    120,
					DateCreated: timestamp,
					DateUpdated: timestamp,
				}

				p1b, err := p1.Encode()
				if err != nil {
					return err
				}
				products.Put([]byte(p1.ID), p1b)

				p2b, err := p2.Encode()
				if err != nil {
					return err
				}
				products.Put([]byte(p2.ID), p2b)
			}

			sales, err := tx.CreateBucketIfNotExists([]byte("sales"))
			if err != nil {
				return errors.Wrap(err, "creating bolt sales bucket")
			}
			{
				s1 := product.Sale{
					ID:          "98b6d4b8-f04b-4c79-8c2e-a0aef46854b7",
					ProductID:   "a2b0639f-2cc6-44b8-b97b-15d69dbb511e",
					Quantity:    2,
					Paid:        100,
					DateCreated: timestamp,
				}
				s1b, err := s1.Encode()
				if err != nil {
					return err
				}

				s2 := product.Sale{
					ID:          "85f6fb09-eb05-4874-ae39-82d1a30fe0d7",
					ProductID:   "a2b0639f-2cc6-44b8-b97b-15d69dbb511e",
					Quantity:    5,
					Paid:        250,
					DateCreated: timestamp,
				}
				s2b, err := s2.Encode()
				if err != nil {
					return err
				}

				s3 := product.Sale{
					ID:          "a235be9e-ab5d-44e6-a987-fa1c749264c7",
					ProductID:   "72f8b983-3eb4-48db-9ed0-e45cc6bd716b",
					Quantity:    3,
					Paid:        225,
					DateCreated: timestamp,
				}
				s3b, err := s3.Encode()
				if err != nil {
					return err
				}

				sales.Put([]byte(s1.ID), s1b)
				sales.Put([]byte(s2.ID), s2b)
				sales.Put([]byte(s3.ID), s3b)
			}

			return nil
		})
		return nil
	}

	return fmt.Errorf("unsupported database")

}

// seeds is a string constant containing all of the queries needed to get the
// db seeded to a useful state for development.
//
// Note that database servers besides PostgreSQL may not support running
// multiple queries as part of the same execution so this single large constant
// may need to be broken up.
const seedsPq = `
INSERT INTO products (product_id, name, cost, quantity, date_created, date_updated) VALUES
	('a2b0639f-2cc6-44b8-b97b-15d69dbb511e', 'Comic Books', 50, 42, '2019-03-24 00:00:00', '2019-03-24 00:00:00'),
	('72f8b983-3eb4-48db-9ed0-e45cc6bd716b', 'McDonalds Toys', 75, 120, '2019-03-24 00:00:00', '2019-03-24 00:00:00')
	ON CONFLICT DO NOTHING;

INSERT INTO sales (sale_id, product_id, quantity, paid, date_created) VALUES
	('98b6d4b8-f04b-4c79-8c2e-a0aef46854b7', 'a2b0639f-2cc6-44b8-b97b-15d69dbb511e', 2, 100, '2019-03-24 00:00:00'),
	('85f6fb09-eb05-4874-ae39-82d1a30fe0d7', 'a2b0639f-2cc6-44b8-b97b-15d69dbb511e', 5, 250, '2019-03-24 00:00:00'),
	('a235be9e-ab5d-44e6-a987-fa1c749264c7', '72f8b983-3eb4-48db-9ed0-e45cc6bd716b', 3, 225, '2019-03-24 00:00:00')
	ON CONFLICT DO NOTHING;

-- Create admin and regular User with password "gophers"
INSERT INTO users (user_id, name, email, roles, password_hash, date_created, date_updated) VALUES
	('5cf37266-3473-4006-984f-9325122678b7', 'Admin Gopher', 'admin@example.com', '{ADMIN,USER}', '$2a$10$1ggfMVZV6Js0ybvJufLRUOWHS5f6KneuP0XwwHpJ8L8ipdry9f2/a', '2019-03-24 00:00:00', '2019-03-24 00:00:00'),
	('45b5fbd3-755f-4379-8f07-a58d4a30fa2f', 'User Gopher', 'user@example.com', '{USER}', '$2a$10$9/XASPKBbJKVfCAZKDH.UuhsuALDr5vVm6VrYA9VFR8rccK86C1hW', '2019-03-24 00:00:00', '2019-03-24 00:00:00')
	ON CONFLICT DO NOTHING;
`
