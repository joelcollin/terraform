package powerdns

type Config struct {
	ServerUrl string
	ApiKey    string
}

// Client returns a new client for accessing PowerDNS
func (c *Config) Client() *Client {
	return NewClient(c.ServerUrl, c.ApiKey)
}
