// This program performs administrative tasks for the garage sale service.

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/conf"
	"github.com/os-foundry/vetpms/internal/platform/database"
	"github.com/os-foundry/vetpms/internal/schema"
	"github.com/os-foundry/vetpms/internal/user"
	userBolt "github.com/os-foundry/vetpms/internal/user/bolt"
	userPq "github.com/os-foundry/vetpms/internal/user/postgres"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

func main() {
	if err := run(); err != nil {
		log.Printf("error: %s", err)
		os.Exit(1)
	}
}

func run() error {

	// =========================================================================
	// Configuration

	var cfg struct {
		DB struct {
			Type       string `conf:"default:postgres"` // Can be postgres or bolt
			User       string `conf:"default:postgres"`
			Password   string `conf:"default:postgres,noprint"`
			Host       string `conf:"default:localhost"`
			Name       string `conf:"default:postgres"`
			DisableTLS bool   `conf:"default:false"`

			// Only required for bolt
			File        string        `conf:"default:/opt/vetpms/data/vetpms.db"`
			Permissions os.FileMode   `conf:"default:0660"`
			Timeout     time.Duration `conf:"default:1s"`
		}
		Args conf.Args
	}

	if err := conf.Parse(os.Args[1:], "VETPMS", &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			usage, err := conf.Usage("VETPMS", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating usage")
			}
			fmt.Println(usage)
			return nil
		}
		return errors.Wrap(err, "error: parsing config")
	}

	var (
		ust      user.Storage
		activeDB interface{}
	)

	switch cfg.DB.Type {
	case "postgres":
		dbConfig := database.Config{
			User:       cfg.DB.User,
			Password:   cfg.DB.Password,
			Host:       cfg.DB.Host,
			Name:       cfg.DB.Name,
			DisableTLS: cfg.DB.DisableTLS,
		}

		db, err := database.Open(dbConfig)
		if err != nil {
			return errors.Wrap(err, "connecting to postgres")
		}

		ust = userPq.Postgres{db}
		activeDB = db

		defer db.Close()

	case "bolt":
		if err := database.CheckAndPrepareBolt(cfg.DB.File, cfg.DB.Permissions); err != nil {
			return errors.Wrap(err, "preparing bolt filepath")
		}

		db, err := bolt.Open(cfg.DB.File, cfg.DB.Permissions, &bolt.Options{Timeout: cfg.DB.Timeout})
		if err != nil {
			return errors.Wrap(err, "connecting to bolt")
		}

		ust = userBolt.Bolt{db}
		activeDB = db

		defer db.Close()

	default:
		return fmt.Errorf("database type should be bolt or postgres")
	}

	var err error
	switch cfg.Args.Num(0) {
	case "migrate":
		err = migrate(activeDB)
	case "seed":
		err = seed(activeDB)
	case "useradd":
		err = useradd(ust, cfg.Args.Num(1), cfg.Args.Num(2))
	case "keygen":
		err = keygen(cfg.Args.Num(1))
	default:
		err = errors.New("Must specify a command")
	}

	if err != nil {
		return err
	}

	return nil
}

func migrate(db interface{}) error {
	if err := schema.Migrate(db); err != nil {
		return err
	}

	fmt.Println("Migrations complete")
	return nil
}

func seed(db interface{}) error {
	if err := schema.Seed(db); err != nil {
		return err
	}

	fmt.Println("Seed data complete")
	return nil
}

func useradd(st user.Storage, email, password string) error {
	if email == "" || password == "" {
		return errors.New("useradd command must be called with two additional arguments for email and password")
	}

	fmt.Printf("Admin user will be created with email %q and password %q\n", email, password)
	fmt.Print("Continue? (1/0) ")

	var confirm bool
	if _, err := fmt.Scanf("%t\n", &confirm); err != nil {
		return errors.Wrap(err, "processing response")
	}

	if !confirm {
		fmt.Println("Canceling")
		return nil
	}

	ctx := context.Background()

	nu := user.NewUser{
		Email:           email,
		Password:        password,
		PasswordConfirm: password,
		Roles:           []string{auth.RoleAdmin, auth.RoleUser},
	}

	u, err := st.Create(ctx, nu, time.Now())
	if err != nil {
		return err
	}

	fmt.Println("User created with id:", u.ID)
	return nil
}

// keygen creates an x509 private key for signing auth tokens.
func keygen(path string) error {
	if path == "" {
		return errors.New("keygen missing argument for key path")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return errors.Wrap(err, "generating keys")
	}

	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "creating private file")
	}
	defer file.Close()

	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	if err := pem.Encode(file, &block); err != nil {
		return errors.Wrap(err, "encoding to private file")
	}

	if err := file.Close(); err != nil {
		return errors.Wrap(err, "closing private file")
	}

	return nil
}
