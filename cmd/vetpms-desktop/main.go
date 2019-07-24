package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/leaanthony/mewn"
	"github.com/wailsapp/wails"
)

func main() {

	// =========================================================================
	// Logging

	// log := log.New(os.Stdout, "VETPMS_DESKTOP : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// =========================================================================
	// Configuration

	enableTLS := flag.Bool("tls", false, "Enable TLS for API requests")
	apiHost := flag.String("api", "127.0.0.1:3000", "API url")
	apiVersion := flag.Int("api-version", 1, "API version")
	readTimeout := flag.Duration("read-timeout", 3*time.Second, "Read timeout duration")
	lang := flag.String("lang", "nl", "Language to use")
	flag.Parse()

	// =========================================================================
	// Start application

	js := mewn.String("./frontend/dist/app.js")
	css := mewn.String("./frontend/dist/app.css")

	app := wails.CreateApp(&wails.AppConfig{
		Width:  1024,
		Height: 768,
		Title:  "Veterinary Practise Management Suite",
		JS:     js,
		CSS:    css,
		Colour: "#131313",
	})

	core, err := NewCore(*apiHost, *apiVersion, *enableTLS, *readTimeout, *lang)
	if err != nil {
		fmt.Printf("Couldn't start application: %v.\n", err)
		os.Exit(1)
	}

	app.Bind(core)
	app.Run()
}
