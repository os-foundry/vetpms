package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/product"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const productsCollection = "products"

// Bolt implements the Storage interface for
// the bolt database
type Bolt struct {
	DB *bolt.DB
}

// List gets all Products from the database.
func (st Bolt) List(ctx context.Context) ([]product.Product, error) {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.List")
	defer span.End()

	products := []product.Product{}
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(productsCollection))
		if err := bucket.ForEach(func(k []byte, v []byte) error {
			u, err := product.Decode(v)
			if err != nil {
				return errors.Wrap(err, "decoding product")
			}
			products = append(products, *u)
			return nil
		}); err != nil {
			return err
		}

		pmap := make(map[string]int)
		for k, v := range products {
			pmap[v.ID] = k
		}

		salesb := tx.Bucket([]byte("sales"))
		if err := salesb.ForEach(func(k []byte, v []byte) error {
			s, err := product.DecodeSale(v)
			if err != nil {
				return errors.Wrap(err, "decoding sale")
			}

			// Get the product index,
			// skip if it doesn't exist in the map
			i, ok := pmap[s.ProductID]
			if !ok {
				return nil
			}
			// Skip if another product ID
			if s.ProductID != products[i].ID {
				return nil
			}
			products[i].Sold += s.Quantity
			products[i].Revenue += s.Paid

			return nil
		}); err != nil {
			return errors.Wrap(err, "getting sales")
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "selecting products")
	}

	return products, nil
}

// Create adds a Product to the database. It returns the created Product with
// fields like ID and DateCreated populated..
func (st Bolt) Create(ctx context.Context, user auth.Claims, np product.NewProduct, now time.Time) (*product.Product, error) {
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

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(productsCollection))

		v, err := p.Encode()
		if err != nil {
			return errors.Wrapf(err, "encoding product")
		}
		if err := bucket.Put([]byte(p.ID), v); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "inserting product")
	}

	return &p, nil
}

// Retrieve finds the product identified by a given ID.
func (st Bolt) Retrieve(ctx context.Context, id string) (*product.Product, error) {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, product.ErrInvalidID
	}

	var p product.Product
	if err := st.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(productsCollection))
		v := bucket.Get([]byte(id))
		if len(v) == 0 {
			return product.ErrNotFound
		}

		if err := p.Decode(v); err != nil {
			return errors.Wrap(err, "decoding product")
		}

		salesb := tx.Bucket([]byte("sales"))
		if err := salesb.ForEach(func(k []byte, v []byte) error {
			s, err := product.DecodeSale(v)
			if err != nil {
				return errors.Wrap(err, "decoding sale")
			}

			// Skip if another product ID
			if s.ProductID != p.ID {
				return nil
			}
			p.Sold += s.Quantity
			p.Revenue += s.Paid

			return nil
		}); err != nil {
			return errors.Wrap(err, "getting sales")
		}

		return nil
	}); err != nil {
		if err == product.ErrNotFound {
			return nil, err
		}
		return nil, errors.Wrapf(err, "selecting product %q", id)
	}

	return &p, nil
}

// Update modifies data about a Product. It will error if the specified ID is
// invalid or does not reference an existing Product.
func (st Bolt) Update(ctx context.Context, user auth.Claims, id string, update product.UpdateProduct, now time.Time) error {
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

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(productsCollection))
		v, err := p.Encode()
		if err != nil {
			return errors.Wrapf(err, "encoding product")
		}
		if err := bucket.Put([]byte(p.ID), v); err != nil {
			return errors.Wrap(err, "writing product data")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "updating product")
	}

	return nil
}

// Delete removes the product identified by a given ID.
func (st Bolt) Delete(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.product.postgres.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return product.ErrInvalidID
	}

	if err := st.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(productsCollection))
		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "deleting product")
	}

	return nil
}
