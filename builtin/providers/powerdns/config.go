package powerdns

import "log"

type Config struct {
	ServerUrl string
	ApiKey    string
}

// Client returns a new client for accessing PowerDNS
func (c *Config) Client() (*Client, error) {
	client := NewClient(c.ServerUrl, c.ApiKey)

	log.Printf("[INFO] PowerDNS Client configured for server %s", c.ServerUrl)

	return client, nil
}
