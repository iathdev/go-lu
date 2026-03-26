package config

import "time"

type OCRConfig struct {
	OCRServiceURL                string        `mapstructure:"OCR_SERVICE_URL"`
	GoogleApplicationCredentials string        `mapstructure:"GOOGLE_APPLICATION_CREDENTIALS"`
	BaiduOCRAPIKey               string        `mapstructure:"BAIDU_OCR_API_KEY"`
	BaiduOCRSecretKey            string        `mapstructure:"BAIDU_OCR_SECRET_KEY"`
	OCRRetryMax                  int           `mapstructure:"OCR_RETRY_MAX"`
	OCRRetryDelay                time.Duration `mapstructure:"OCR_RETRY_DELAY"`
}

func (config *Config) GetOCRRetryMax() int {
	if config.OCRRetryMax == 0 {
		return 3
	}
	return config.OCRRetryMax
}

func (config *Config) GetOCRRetryDelay() time.Duration {
	if config.OCRRetryDelay == 0 {
		return 200 * time.Millisecond
	}
	return config.OCRRetryDelay
}
