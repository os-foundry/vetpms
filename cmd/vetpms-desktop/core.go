package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails"
)

const (
	LoginEvent  = "LOGIN"
	LogoutEvent = "LOGOUT"
)

// Core contains all Go powered core functionality of the app
type Core struct {
	log   *wails.CustomLogger
	rt    *wails.Runtime
	token string
	user  *User
	Cfg   Config
}

// NewCore returns an initialized Core.
// Returns an error if configuration validation fails.
func NewCore(api string, apiVersion int, tls bool, readTimeout time.Duration) (*Core, error) {
	c := Core{
		Cfg: Config{
			EnableTLS:   tls,
			API:         api,
			APIVersion:  apiVersion,
			ReadTimeout: readTimeout,
		},
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

	// Make the request to the backend.
	client := http.Client{Timeout: c.Cfg.ReadTimeout}
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
