package main

import "time"

// Sends an alert event to the UI
func (c *Core) Alert(t, txt string) {
	msg := struct {
		Type    string        `json:"type"`
		Text    string        `json:"text"`
		Timeout time.Duration `json:"timeout"`
	}{Type: t, Text: txt}
	c.rt.Events.Emit(AlertEvent, msg)
}

// ReturnWithAlert sends an alert with the provided error
// and returns the error.
// Example usage: return c.ReturnWithAlert(err)
func (c *Core) ReturnWithAlert(err error) error {
	c.Alert("error", err.Error())
	return err
}
