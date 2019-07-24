package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// User is a struct for holding user data
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	DateCreated time.Time `json:"date_created"`
	DateUpdates time.Time `json:"date_updated"`
}

// Me makes a request to the server to get all data of the currently logged in user.
func (c *Core) Me() (*User, error) {

	r, err := http.NewRequest(http.MethodGet, c.Cfg.APIurl()+"user", nil)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	b, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	var v User
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	return &v, nil
}

// Login makes a login request at the backend.
func (c *Core) Login(q, passwd string) (*User, error) {

	// Prepare the login request
	r, err := http.NewRequest(http.MethodGet, c.Cfg.APIurl()+"users/token", nil)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Set authentication
	r.SetBasicAuth(q, passwd)

	// Execute the request
	b, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Unmarshal the response
	t := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Set the token and submit a login event
	c.token = t.Token

	// Get the user's details
	me, err := c.Me()
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	c.user = me
	c.rt.Events.Emit(LoginEvent, me)

	// Send a localized alert to the frontend
	lc := i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "LoginOK",
			Other: "You have succesfully logged in.",
		},
	}
	c.Alert("success", c.Localize(&lc))
	return me, nil
}

// Logout destroys the held token and that way logs the user out.
func (c *Core) Logout() {
	c.token = ""
	c.user = nil
	c.rt.Events.Emit(LogoutEvent)

	// Send a localized alert to the frontend
	lc := i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "LogoutOK",
			Other: "You have succesfully logged out.",
		},
	}
	c.Alert("success", c.Localize(&lc))
}

// UserCreate sends the provided json user input to the server for the creation of a new user.
func (c *Core) UserCreate(user string) (*User, error) {

	// Create request
	r, err := http.NewRequest(http.MethodPost, c.Cfg.APIurl()+"users", bytes.NewBuffer([]byte(user)))
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Make request
	res, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Unmarshal response
	var u User
	if err := json.Unmarshal(res, &u); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Send a localized alert to the frontend
	lc := i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "UserCreateOK",
			Other: "You have succesfully created a new user.",
		},
	}
	c.Alert("success", c.Localize(&lc))
	return &u, nil
}

// UserGet receives a particular user by its ID from the server.
func (c *Core) UserGet(id string) (*User, error) {

	// Create request
	r, err := http.NewRequest(http.MethodGet, c.Cfg.APIurl()+"users/"+id, nil)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Make request
	res, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Unmarshal response
	var u User
	if err := json.Unmarshal(res, &u); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	return &u, nil
}

// UserList receives a slice of users from the servier.
func (c *Core) UserList() ([]User, error) {

	// Create request
	r, err := http.NewRequest(http.MethodGet, c.Cfg.APIurl()+"users", nil)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Make request
	res, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Unmarshal response
	var u []User
	if err := json.Unmarshal(res, &u); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	return u, nil
}

// UserUpdate sends the provided json user input to the server to update a user's details
func (c *Core) UserUpdate(user string) (*User, error) {

	// Create request
	r, err := http.NewRequest(http.MethodPut, c.Cfg.APIurl()+"users", bytes.NewBuffer([]byte(user)))
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Make request
	res, err := c.doRequest(r)
	if err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Unmarshal response
	var u User
	if err := json.Unmarshal(res, &u); err != nil {
		return nil, c.ReturnWithAlert(err)
	}

	// Send a localized alert to the frontend
	lc := i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "UserUpdateOK",
			Other: "You have succesfully updated the user.",
		},
	}
	c.Alert("success", c.Localize(&lc))
	return &u, nil
}

// UserDelete sends a request for the removal of a user to the server.
func (c *Core) UserDelete(id string) error {
	// Create request
	r, err := http.NewRequest(http.MethodDelete, c.Cfg.APIurl()+"users/"+id, nil)
	if err != nil {
		return c.ReturnWithAlert(err)
	}

	// Make request
	if _, err := c.doRequest(r); err != nil {
		return c.ReturnWithAlert(err)
	}

	// Send a localized alert to the frontend
	lc := i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "UserDeleteOK",
			Other: "You have succesfully deleted the user.",
		},
	}
	c.Alert("success", c.Localize(&lc))
	return nil
}
