package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentProduction  = "production"
)

var (
	ErrMissing = errors.New("required configuration is missing")
	ErrInvalid = errors.New("configuration is invalid")
)

// Config contains control-plane settings after validation. SigningSecret is
// copied into private process memory and must never be included in logs or
// returned from an HTTP response.
type Config struct {
	Environment   string
	LogLevel      string
	AWSRegion     string
	APIBaseURL    string
	SigningSecret []byte
}

type LookupEnv func(string) (string, bool)

func Load() (Config, error) {
	return LoadFrom(os.LookupEnv)
}

func LoadFrom(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		return Config{}, fmt.Errorf("%w: environment lookup is required", ErrInvalid)
	}
	environment := strings.ToLower(valueOr(lookup, "HAEDES_ENV", EnvironmentDevelopment))
	if environment != EnvironmentDevelopment && environment != EnvironmentTest && environment != EnvironmentProduction {
		return Config{}, fmt.Errorf("%w: HAEDES_ENV must be development, test, or production", ErrInvalid)
	}
	config := Config{
		Environment: environment,
		LogLevel:    valueOr(lookup, "HAEDES_LOG_LEVEL", "info"),
		AWSRegion:   valueOr(lookup, "HAEDES_AWS_REGION", "us-east-1"),
		APIBaseURL:  valueOr(lookup, "HAEDES_API_BASE_URL", "http://localhost:8080"),
	}
	if signingSecret, ok := lookup("HAEDES_SIGNING_SECRET"); ok {
		config.SigningSecret = []byte(signingSecret)
	}

	if environment == EnvironmentProduction {
		for name, value := range map[string]string{
			"HAEDES_SIGNING_SECRET": string(config.SigningSecret),
			"HAEDES_AWS_REGION":     config.AWSRegion,
			"HAEDES_API_BASE_URL":   config.APIBaseURL,
		} {
			if value == "" {
				return Config{}, fmt.Errorf("%w: %s", ErrMissing, name)
			}
		}
		if len(config.SigningSecret) < 32 {
			return Config{}, fmt.Errorf("%w: HAEDES_SIGNING_SECRET must contain at least 32 bytes", ErrInvalid)
		}
	}
	if config.LogLevel == "" || config.AWSRegion == "" || config.APIBaseURL == "" {
		return Config{}, fmt.Errorf("%w: log level, AWS region, and API base URL must be non-empty", ErrInvalid)
	}
	return config, nil
}

func valueOr(lookup LookupEnv, name, fallback string) string {
	value, ok := lookup(name)
	if !ok {
		return fallback
	}
	return value
}
