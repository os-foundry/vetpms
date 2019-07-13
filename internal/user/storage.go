package user

import (
	"context"
	"time"

	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/database"
)

// Storage is an entity providing access to the user database
type Storage interface {
	database.StatusChecker
	List(ctx context.Context) ([]User, error)
	Retrieve(ctx context.Context, claims auth.Claims, id string) (*User, error)
	Create(ctx context.Context, n NewUser, now time.Time) (*User, error)
	Update(ctx context.Context, claims auth.Claims, id string, upd UpdateUser, now time.Time) error
	Delete(ctx context.Context, id string) error
	Authenticate(ctx context.Context, now time.Time, email, password string) (auth.Claims, error)
}
