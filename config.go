package traefik_throttle

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

// Config struct holds the configuration for rate limiting.
type Config struct {
	EndpointsConfigLocation string                        `json:"endpointsConfigLocation" yaml:"endpointsConfigLocation"`
	MaxRequests             int                           `json:"maxRequests" yaml:"maxRequests"`
	MaxQueue                int                           `json:"maxQueue" yaml:"maxQueue"`
	RetryCount              int                           `json:"retryCount" yaml:"retryCount"`
	RetryDelay              string                        `json:"retryDelay" yaml:"retryDelay"`
	retryDelayDuration      time.Duration                 `json:"-" yaml:"-"`
	Endpoints               map[string]map[string]*Config `json:"endpoints" yaml:"endpoints"` // Per-endpoint and method-specific rate limits
	UserMaxRequests         int                           `json:"userMaxRequests" yaml:"userMaxRequests"`
	UserRetryDelay          string                        `json:"userRetryDelay" yaml:"userRetryDelay"`
	userRetryDelayDuration  time.Duration                 `json:"-" yaml:"-"`
	JWTSecret               string                        `json:"jwtSecret" yaml:"jwtSecret"`
}

// CreateConfig initializes a default configuration for rate limiting.
func CreateConfig() *Config {
	return &Config{
		MaxRequests:     10,
		MaxQueue:        0,
		RetryCount:      3,
		RetryDelay:      "200ms",
		Endpoints:       make(map[string]map[string]*Config),
		UserMaxRequests: 1,
		UserRetryDelay:  "1s",
	}
}

// LoadConfigFromFile loads the configuration from a file.
func loadConfigFromFile(config *Config) error {
	file, err := os.Open(config.EndpointsConfigLocation)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	var fileConfig Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	*config = fileConfig
	return nil
}
