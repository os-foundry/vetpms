package main

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	EnableTLS   bool
	API         string
	APIVersion  int
	ReadTimeout time.Duration
	Lang        string
}

func (c *Config) Validate() error {
	// Check and correct API
	{
		if !strings.HasPrefix(strings.ToLower(c.API), "http") {
			if c.EnableTLS {
				c.API = "https://" + c.API + "/"
			} else {
				c.API = "http://" + c.API + "/"
			}
		}

		if !strings.HasSuffix(c.API, "/") {
			c.API = c.API + "/"
		}
	}
	return nil
}

func (c *Config) APIurl() string {
	return fmt.Sprintf("%sv%d/", c.API, c.APIVersion)
}
