package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type User struct {
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	DateCreated time.Time `json:"date_created"`
	DateUpdates time.Time `json:"date_updated"`
}

func (c *Core) Me() (*User, error) {

	r, err := http.NewRequest(http.MethodGet, c.Cfg.APIurl()+"/user", nil)
	if err != nil {
		return nil, err
	}

	b, err := c.doRequest(r)
	if err != nil {
		return nil, err
	}

	var v User
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

// Login makes a login request at the backend.
func (c *Core) Login(q, passwd string) (*User, error) {

	// Prepare the login request
	r, err := http.NewRequest(http.MethodPost, c.Cfg.APIurl()+"/users/token", nil)
	if err != nil {
		return nil, err
	}

	// Set authentication
	r.SetBasicAuth(q, passwd)

	// Execute the request
	b, err := c.doRequest(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response
	t := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}

	// Set the token and submit a login event
	c.token = t.Token

	// Get the user's details
	me, err := c.Me()
	if err != nil {
		return nil, err
	}

	c.user = me
	c.rt.Events.Emit(LoginEvent, me)
	return me, nil
}

// Logout destroys the held token and that way logs the user out.
func (c *Core) Logout() {
	c.token = ""
	c.user = nil
	c.rt.Events.Emit(LogoutEvent)
}
