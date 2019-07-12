package user

import (
	"bytes"
	"time"

	"encoding/gob"

	"github.com/lib/pq"
)

// User represents someone with access to our system.
type User struct {
	ID           string         `db:"user_id" json:"id"`
	Name         string         `db:"name" json:"name"`
	Email        string         `db:"email" json:"email"`
	Roles        pq.StringArray `db:"roles" json:"roles"`
	PasswordHash []byte         `db:"password_hash" json:"-"`
	DateCreated  time.Time      `db:"date_created" json:"date_created"`
	DateUpdated  time.Time      `db:"date_updated" json:"date_updated"`
}

// Encode gob encodes all user data into a slice of bytes.
func (u *User) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(u); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode gob decodes a slice of bytes into the user.
func (u *User) Decode(b []byte) error {
	if err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&u); err != nil {
		return err
	}
	return nil
}

// Decode creates a new User from a gob encoded byte slice.
func Decode(b []byte) (*User, error) {
	var u User
	if err := u.Decode(b); err != nil {
		return nil, err
	}
	return &u, nil
}

// NewUser contains information needed to create a new User.
type NewUser struct {
	Name            string   `json:"name" validate:"required"`
	Email           string   `json:"email" validate:"required"`
	Roles           []string `json:"roles" validate:"required"`
	Password        string   `json:"password" validate:"required"`
	PasswordConfirm string   `json:"password_confirm" validate:"eqfield=Password"`
}

// UpdateUser defines what information may be provided to modify an existing
// User. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateUser struct {
	Name            *string  `json:"name"`
	Email           *string  `json:"email"`
	Roles           []string `json:"roles"`
	Password        *string  `json:"password"`
	PasswordConfirm *string  `json:"password_confirm" validate:"omitempty,eqfield=Password"`
}

func init() {
	gob.Register(&User{})
}
