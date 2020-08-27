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

type AppConfig struct {
	WebPort     int              `json:"webPort"`
	MetricsPort int              `json:"metricsPort"`
	MetricsPath string           `json:"metricsPath"`
	CloudWatch  CloudWatchConfig `json:"cloudWatch"`
	Kafka       KafkaConfig      `json:"kafka"`
}

type AppConfigBuilder struct {
	config AppConfig
}

func NewBuilder() *AppConfigBuilder {
	return &AppConfigBuilder{}
}

func (c *AppConfigBuilder) CloudWatch(cw *CloudWatchConfig) *AppConfigBuilder {
	c.config.CloudWatch = *cw
	return c
}

func (c *AppConfigBuilder) Build() *AppConfig {
	return &c.config
}
