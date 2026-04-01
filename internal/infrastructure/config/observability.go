package config

type ObservabilityConfig struct {
	OTLPEndpoint string `mapstructure:"OTLP_ENDPOINT"`
	OTLPInsecure bool   `mapstructure:"OTLP_INSECURE"`
	ServiceName  string `mapstructure:"SERVICE_NAME"`

	SentryDSN         string  `mapstructure:"SENTRY_DSN"`
	SentryEnvironment string  `mapstructure:"SENTRY_ENVIRONMENT"`
	SentrySampleRate  float64 `mapstructure:"SENTRY_SAMPLE_RATE"`
}

func (config *Config) GetServiceName() string {
	if config.ServiceName == "" {
		return "learning-utilities"
	}
	return config.ServiceName
}

func (config *Config) GetSentrySampleRate() float64 {
	if config.SentrySampleRate == 0 {
		return 1.0
	}
	return config.SentrySampleRate
}
