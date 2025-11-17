package websocket

// IsSubscribed checks if client is subscribed to a specific event type
func (c *Client) IsSubscribed(eventType string) bool {
	if len(c.subscriptions) == 0 {
		// If no subscriptions, client receives all events (default behavior)
		return true
	}
	return c.subscriptions[eventType]
}

// GetSubscriptions returns a list of subscribed events
func (c *Client) GetSubscriptions() []string {
	events := make([]string, 0, len(c.subscriptions))
	for event := range c.subscriptions {
		events = append(events, event)
	}
	return events
}
