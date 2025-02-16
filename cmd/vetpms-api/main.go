package main

import (
	"context"
	"crypto/rsa"
	"expvar" // Register the expvar handlers
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof" // Register the pprof handlers
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"contrib.go.opencensus.io/exporter/zipkin"
	jwt "github.com/dgrijalva/jwt-go"
	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/os-foundry/vetpms/cmd/vetpms-api/internal/handlers"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/conf"
	"github.com/os-foundry/vetpms/internal/platform/database"
	"github.com/os-foundry/vetpms/internal/platform/logtracer"
	"github.com/os-foundry/vetpms/internal/product"
	productBolt "github.com/os-foundry/vetpms/internal/product/bolt"
	productPq "github.com/os-foundry/vetpms/internal/product/postgres"
	"github.com/os-foundry/vetpms/internal/user"
	userBolt "github.com/os-foundry/vetpms/internal/user/bolt"
	userPq "github.com/os-foundry/vetpms/internal/user/postgres"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

/*
ZipKin: http://localhost:9411
AddLoad: hey -m GET -c 10 -n 10000 "http://localhost:3000/v1/users"
expvarmon -ports=":3001" -endpoint="/metrics" -vars="requests,goroutines,errors,mem:memstats.Alloc"
*/

/*
Need to figure out timeouts for http service.
You might want to reset your DB_HOST env var during test tear down.
Service should start even without a DB running yet.
symbols in profiles: https://github.com/golang/go/issues/23376 / https://github.com/google/pprof/pull/366
*/

// build is the git version of this program. It is set using build flags in the makefile.
var build = "develop"

func main() {
	if err := run(); err != nil {
		log.Println("error :", err)
		os.Exit(1)
	}
}

func run() error {

	// =========================================================================
	// Logging

	log := log.New(os.Stdout, "VETPMS : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// =========================================================================
	// Configuration

	var cfg struct {
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        // `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
		}
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
		Auth struct {
			KeyID          string `conf:"default:1"`
			PrivateKeyFile string `conf:"default:/app/private.pem"`
			Algorithm      string `conf:"default:RS256"`
		}
		Zipkin struct {
			Enabled       bool    `conf:"default:false"`
			LocalEndpoint string  `conf:"default:0.0.0.0:3000"`
			ReporterURI   string  `conf:"default:http://zipkin:9411/api/v2/spans"`
			ServiceName   string  `conf:"default:vetpms-api"`
			Probability   float64 `conf:"default:0.05"`
		}
	}

	if err := conf.Parse(os.Args[1:], "VETPMS", &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			usage, err := conf.Usage("VETPMS", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	// =========================================================================
	// App Starting

	// Print the build version for our logs. Also expose it under /debug/vars.
	expvar.NewString("build").Set(build)
	log.Printf("main : Started : Application initializing : version %q", build)
	defer log.Println("main : Completed")

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main : Config :\n%v\n", out)

	// =========================================================================
	// Initialize authentication support

	log.Println("main : Started : Initializing authentication support")

	keyContents, err := ioutil.ReadFile(cfg.Auth.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading auth private key")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyContents)
	if err != nil {
		return errors.Wrap(err, "parsing auth private key")
	}

	f := auth.NewSimpleKeyLookupFunc(cfg.Auth.KeyID, privateKey.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(privateKey, cfg.Auth.KeyID, cfg.Auth.Algorithm, f)
	if err != nil {
		return errors.Wrap(err, "constructing authenticator")
	}

	// =========================================================================
	// Start Database and initialize storages

	log.Println("main : Started : Initializing database support")

	var (
		ust user.Storage
		pst product.Storage
	)
	switch strings.ToLower(cfg.DB.Type) {

	case "postgres":
		db, err := database.Open(database.Config{
			User:       cfg.DB.User,
			Password:   cfg.DB.Password,
			Host:       cfg.DB.Host,
			Name:       cfg.DB.Name,
			DisableTLS: cfg.DB.DisableTLS,
		})
		if err != nil {
			return errors.Wrap(err, "connecting to db")
		}

		ust = userPq.Postgres{db}
		pst = productPq.Postgres{db}

		defer func() {
			log.Printf("main : Database Stopping : %s", cfg.DB.Host)
			db.Close()
		}()

	case "bolt":
		if err := database.CheckAndPrepareBolt(cfg.DB.File, cfg.DB.Permissions); err != nil {
			return errors.Wrap(err, "preparing bolt filepath")
		}

		db, err := bolt.Open(cfg.DB.File, cfg.DB.Permissions, &bolt.Options{Timeout: cfg.DB.Timeout})
		if err != nil {
			return errors.Wrap(err, "connecting to db")
		}

		ust = userBolt.Bolt{db}
		pst = productBolt.Bolt{db}

		defer func() {
			log.Printf("main : Database Stopping : %s", cfg.DB.Host)
			db.Close()
		}()
	}

	// =========================================================================
	// Start Tracing Support

	if cfg.Zipkin.Enabled {
		log.Println("main : Started : Initializing zipkin tracing support")
		localEndpoint, err := openzipkin.NewEndpoint("vetpms-api", cfg.Zipkin.LocalEndpoint)
		if err != nil {
			return err
		}

		reporter := zipkinHTTP.NewReporter(cfg.Zipkin.ReporterURI)
		ze := zipkin.NewExporter(reporter, localEndpoint)

		trace.RegisterExporter(ze)
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.ProbabilitySampler(cfg.Zipkin.Probability),
		})

		defer func() {
			log.Printf("main : Tracing Stopping : %s", cfg.Zipkin.LocalEndpoint)
			reporter.Close()
		}()
	} else {
		log.Println("main : Started : Initializing logtracer tracing support")
		trace.RegisterExporter(&logtracer.Tracer{log})
	}

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	if cfg.Web.DebugHost != "" {
		log.Println("main : Started : Initializing debugging support")

		go func() {
			log.Printf("main : Debug Listening %s", cfg.Web.DebugHost)
			log.Printf("main : Debug Listener closed : %v", http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux))
		}()
	}

	// =========================================================================
	// Start API Service

	log.Println("main : Started : Initializing API support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handlers.API(shutdown, log, ust, pst, authenticator),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("main : API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "starting server")

	case sig := <-shutdown:
		log.Printf("main : %v : Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := api.Shutdown(ctx)
		if err != nil {
			log.Printf("main : Graceful shutdown did not complete in %v : %v", cfg.Web.ShutdownTimeout, err)
			err = api.Close()
		}

		// Log the status of this shutdown.
		switch {
		case sig == syscall.SIGSTOP:
			return errors.New("integrity issue caused shutdown")
		case err != nil:
			return errors.Wrap(err, "could not stop server gracefully")
		}
	}

	return nil
}
