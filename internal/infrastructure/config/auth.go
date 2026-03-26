package config

import "time"

const (
	defaultPrepAuthMeEndpoint      = "/auth/api/v1.1/auth/me"
	defaultPrepAuthTokenEndpoint = "/api/v1.1/auth/token"
	defaultPrepTokenCacheTTL   = 5 * time.Minute
)

type AuthConfig struct {
	PrepAPIGatewayDomain string        `mapstructure:"PREP_API_GATEWAY_DOMAIN"`
	PrepAuthTokenEndpoint string        `mapstructure:"PREP_AUTH_TOKEN_ENDPOINT"`
	PrepMeEndpoint        string        `mapstructure:"PREP_ME_ENDPOINT"`
	PrepHTTPClientTimeout time.Duration `mapstructure:"PREP_HTTP_CLIENT_TIMEOUT"`
	PrepTokenCacheTTL     time.Duration `mapstructure:"PREP_TOKEN_CACHE_TTL"`
}

func (config *Config) GetPrepMeEndpoint() string {
	if config.PrepMeEndpoint == "" {
		return defaultPrepAuthMeEndpoint
	}
	return config.PrepMeEndpoint
}

func (config *Config) GetPrepHTTPClientTimeout() time.Duration {
	if config.PrepHTTPClientTimeout == 0 {
		return 10 * time.Second
	}
	return config.PrepHTTPClientTimeout
}

func (config *Config) GetPrepAuthTokenEndpoint() string {
	if config.PrepAuthTokenEndpoint == "" {
		return defaultPrepAuthTokenEndpoint
	}
	return config.PrepAuthTokenEndpoint
}

func (config *Config) GetPrepTokenCacheTTL() time.Duration {
	if config.PrepTokenCacheTTL == 0 {
		return defaultPrepTokenCacheTTL
	}
	return config.PrepTokenCacheTTL
}
