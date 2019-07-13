package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/product"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Postgres implements the Storage interface for
// the postgres database
type Postgres struct {
	DB *sqlx.DB
}

// List gets all Products from the database.
func (st Postgres) List(ctx context.Context) ([]product.Product, error) {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.List")
	defer span.End()

	products := []product.Product{}
	const q = `SELECT
			p.*,
			COALESCE(SUM(s.quantity) ,0) AS sold,
			COALESCE(SUM(s.paid), 0) AS revenue
		FROM products AS p
		LEFT JOIN sales AS s ON p.product_id = s.product_id
		GROUP BY p.product_id`

	if err := st.DB.SelectContext(ctx, &products, q); err != nil {
		return nil, errors.Wrap(err, "selecting products")
	}

	return products, nil
}

// Create adds a Product to the database. It returns the created Product with
// fields like ID and DateCreated populated..
func (st Postgres) Create(ctx context.Context, user auth.Claims, np product.NewProduct, now time.Time) (*product.Product, error) {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Create")
	defer span.End()

	p := product.Product{
		ID:          uuid.New().String(),
		Name:        np.Name,
		Cost:        np.Cost,
		Quantity:    np.Quantity,
		UserID:      user.Subject,
		DateCreated: now.UTC(),
		DateUpdated: now.UTC(),
	}

	const q = `
		INSERT INTO products
		(product_id, user_id, name, cost, quantity, date_created, date_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := st.DB.ExecContext(ctx, q,
		p.ID, p.UserID,
		p.Name, p.Cost, p.Quantity,
		p.DateCreated, p.DateUpdated)
	if err != nil {
		return nil, errors.Wrap(err, "inserting product")
	}

	return &p, nil
}

// Retrieve finds the product identified by a given ID.
func (st Postgres) Retrieve(ctx context.Context, id string) (*product.Product, error) {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, product.ErrInvalidID
	}

	var p product.Product

	const q = `SELECT
			p.*,
			COALESCE(SUM(s.quantity), 0) AS sold,
			COALESCE(SUM(s.paid), 0) AS revenue
		FROM products AS p
		LEFT JOIN sales AS s ON p.product_id = s.product_id
		WHERE p.product_id = $1
		GROUP BY p.product_id`

	if err := st.DB.GetContext(ctx, &p, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, product.ErrNotFound
		}

		return nil, errors.Wrap(err, "selecting single product")
	}

	return &p, nil
}

// Update modifies data about a Product. It will error if the specified ID is
// invalid or does not reference an existing Product.
func (st Postgres) Update(ctx context.Context, user auth.Claims, id string, update product.UpdateProduct, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Update")
	defer span.End()

	p, err := st.Retrieve(ctx, id)
	if err != nil {
		return err
	}

	// If you do not have the admin role ...
	// and you are not the owner of this product ...
	// then get outta here!
	if !user.HasRole(auth.RoleAdmin) && p.UserID != user.Subject {
		return product.ErrForbidden
	}

	if update.Name != nil {
		p.Name = *update.Name
	}
	if update.Cost != nil {
		p.Cost = *update.Cost
	}
	if update.Quantity != nil {
		p.Quantity = *update.Quantity
	}
	p.DateUpdated = now

	const q = `UPDATE products SET
		"name" = $2,
		"cost" = $3,
		"quantity" = $4,
		"date_updated" = $5
		WHERE product_id = $1`
	_, err = st.DB.ExecContext(ctx, q, id,
		p.Name, p.Cost,
		p.Quantity, p.DateUpdated,
	)
	if err != nil {
		return errors.Wrap(err, "updating product")
	}

	return nil
}

// Delete removes the product identified by a given ID.
func (st Postgres) Delete(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return product.ErrInvalidID
	}

	const q = `DELETE FROM products WHERE product_id = $1`

	if _, err := st.DB.ExecContext(ctx, q, id); err != nil {
		return errors.Wrapf(err, "deleting product %s", id)
	}

	return nil
}
