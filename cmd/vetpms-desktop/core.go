package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	_ "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/wailsapp/wails"
	"golang.org/x/text/language"
)

// Core contains all Go powered core functionality of the app
type Core struct {
	log       *wails.CustomLogger
	rt        *wails.Runtime
	token     string
	user      *User
	Cfg       Config
	Localizer *i18n.Localizer
}

// NewCore returns an initialized Core.
// Returns an error if configuration validation fails.
func NewCore(api string, apiVersion int, tls bool, readTimeout time.Duration, lang string) (*Core, error) {
	// Setup the i18n bundle
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFile("active.en.toml")
	bundle.LoadMessageFile("active.nl.toml")
	bundle.LoadMessageFile("active.bg.toml")

	localizer := i18n.NewLocalizer(bundle, lang)

	c := Core{
		Cfg: Config{
			EnableTLS:   tls,
			API:         api,
			APIVersion:  apiVersion,
			ReadTimeout: readTimeout,
			Lang:        lang,
		},
		Localizer: localizer,
	}

	if err := c.Cfg.Validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

// WailsInit performs all initialization taks for Wails.
func (c *Core) WailsInit(rt *wails.Runtime) error {
	c.rt = rt
	c.log = rt.Log.New("Core")
	return nil
}

// Localize returns a localized version of the provided message.
func (c *Core) Localize(lc *i18n.LocalizeConfig) string {
	// Get the localized string
	str, err := c.Localizer.Localize(lc)
	if err != nil {
		c.log.Errorf("[core.Localize] %v", err)
		return ""
	}
	return str
}

// ServerError is a custom error in which server errors are wrapped.
type ServerError struct {
	Code int
	Err  string `json:"error"`
}

// Implemented the Error interface.
func (err *ServerError) Error() string {
	if strings.Contains(err.Err, strconv.Itoa(err.Code)) {
		return err.Err
	}

	return fmt.Sprintf("%s (%d)", err.Err, err.Code)
}

// doRequest sends a request to the backend.
func (c *Core) doRequest(r *http.Request) ([]byte, error) {

	// Add an authorization header if a token is set.
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}
	r.Header.Set("Content-Type", "application/json")

	// Make the request to the backend.
	client := http.Client{Timeout: c.Cfg.ReadTimeout}
	c.log.Infof("Making request to %s with method %s", r.URL.String(), r.Method)
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Server returned an error, so let's wrap it into a ServerError.
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		servErr := ServerError{Code: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(&err); err != nil {
			servErr.Err = resp.Status
		}
		return nil, &servErr

	}

	// Everything went well, let's return the response.
	return ioutil.ReadAll(resp.Body)
}

// Token returns the token
func (c *Core) CurrentUser() *User {
	return c.user
}

func (c *Core) Language() string {
	return c.Cfg.Lang
}
