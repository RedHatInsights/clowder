package config

type CloudWatchConfig struct {
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
	LogGroup        string `json:"logGroup"`
}

type BrokerConfig struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

type TopicConfig struct {
	Name              string `json:"name"`
	ConsumerGroupName string `json:"consumerGroup,omitempty"`
}

type KafkaConfig struct {
	Brokers []BrokerConfig `json:"brokers"`
	Topics  []TopicConfig  `json:"topics"`
}

type DatabaseConfig struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Hostname string `json:"hostname"`
	Port     int32  `json:"port"`
}

type AppConfig struct {
	WebPort     int32            `json:"webPort"`
	MetricsPort int32            `json:"metricsPort"`
	MetricsPath string           `json:"metricsPath"`
	CloudWatch  CloudWatchConfig `json:"cloudWatch"`
	Kafka       KafkaConfig      `json:"kafka"`
	Database    DatabaseConfig   `json:"database"`
}

type Option func(*AppConfig)

func CloudWatch(cwc CloudWatchConfig) Option {
	return func(c *AppConfig) {
		c.CloudWatch = cwc
	}
}

func Kafka(kc KafkaConfig) Option {
	return func(c *AppConfig) {
		c.Kafka = kc
	}
}

func Database(dc DatabaseConfig) Option {
	return func(c *AppConfig) {
		c.Database = dc
	}
}

func New(webPort int32, metricsPort int32, metricsPath string, opts ...Option) *AppConfig {
	c := &AppConfig{
		WebPort:     webPort,
		MetricsPort: metricsPort,
		MetricsPath: metricsPath,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
