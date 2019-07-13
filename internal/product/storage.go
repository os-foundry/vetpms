package product

import (
	"context"
	"time"

	"github.com/os-foundry/vetpms/internal/platform/auth"
)

type Storage interface {
	List(ctx context.Context) ([]Product, error)
	Create(ctx context.Context, user auth.Claims, np NewProduct, now time.Time) (*Product, error)
	Retrieve(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, user auth.Claims, id string, update UpdateProduct, now time.Time) error
	Delete(ctx context.Context, id string) error
}
