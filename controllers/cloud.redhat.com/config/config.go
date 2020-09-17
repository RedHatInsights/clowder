package config

import "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

type ConfigOption func(*AppConfig)

func ObjectStore(store *ObjectStoreConfig) ConfigOption {
	return func(c *AppConfig) {
		c.ObjectStore = store
	}
}

func Logging(logging LoggingConfig) ConfigOption {
	return func(c *AppConfig) {
		c.Logging = logging
	}
}

func Database(dc *DatabaseConfig) ConfigOption {
	return func(c *AppConfig) {
		c.Database = dc
	}
}

func Web(port int) ConfigOption {
	return func(c *AppConfig) {
		c.WebPort = port
	}
}

func Metrics(path string, port int) ConfigOption {
	return func(c *AppConfig) {
		c.MetricsPath = path
		c.MetricsPort = port
	}
}

func New(opts ...ConfigOption) *AppConfig {
	c := &AppConfig{
		WebPort:     8080,
		MetricsPort: 9090,
		MetricsPath: "/metrics",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewDatabaseConfig creates a new config for a database
func NewDatabaseConfig(name string, hostname string) *DatabaseConfig {
	return &DatabaseConfig{
		Name:     name,
		Hostname: hostname,
		Username: utils.RandString(12),
		Password: utils.RandString(12),
		PgPass:   utils.RandString(12),
		Port:     5432,
	}
}
