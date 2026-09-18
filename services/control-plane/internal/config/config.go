package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
	Environment             string
	LogLevel                string
	AWSRegion               string
	AWSEnabled              bool
	ECSCluster              string
	ECSTaskDefinition       string
	ECSContainerName        string
	PrivateSubnetIDs        []string
	SandboxSecurityGroupIDs []string
	DynamoDBTable           string
	DynamoDBIndex           string
	SnapshotTable           string
	SnapshotBucket          string
	APIBaseURL              string
	SigningSecret           []byte
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
		Environment:             environment,
		LogLevel:                valueOr(lookup, "HAEDES_LOG_LEVEL", "info"),
		AWSRegion:               valueOr(lookup, "HAEDES_AWS_REGION", "us-east-1"),
		AWSEnabled:              boolOr(lookup, "HAEDES_AWS_ENABLED", false),
		ECSCluster:              valueOr(lookup, "HAEDES_ECS_CLUSTER", ""),
		ECSTaskDefinition:       valueOr(lookup, "HAEDES_ECS_TASK_DEFINITION", ""),
		ECSContainerName:        valueOr(lookup, "HAEDES_ECS_CONTAINER_NAME", "runtime"),
		PrivateSubnetIDs:        splitCSV(valueOr(lookup, "HAEDES_PRIVATE_SUBNET_IDS", "")),
		SandboxSecurityGroupIDs: splitCSV(valueOr(lookup, "HAEDES_SANDBOX_SECURITY_GROUP_IDS", "")),
		DynamoDBTable:           valueOr(lookup, "HAEDES_DYNAMODB_TABLE", ""),
		DynamoDBIndex:           valueOr(lookup, "HAEDES_DYNAMODB_INDEX", "owner-createdAt-index"),
		SnapshotTable:           valueOr(lookup, "HAEDES_SNAPSHOT_TABLE", ""),
		SnapshotBucket:          valueOr(lookup, "HAEDES_SNAPSHOT_BUCKET", ""),
		APIBaseURL:              valueOr(lookup, "HAEDES_API_BASE_URL", "http://localhost:8080"),
	}
	if raw, ok := lookup("HAEDES_AWS_ENABLED"); ok && strings.TrimSpace(raw) != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%w: HAEDES_AWS_ENABLED must be true or false", ErrInvalid)
		}
		config.AWSEnabled = parsed
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
	if config.AWSEnabled {
		for name, value := range map[string]string{
			"HAEDES_ECS_CLUSTER":         config.ECSCluster,
			"HAEDES_ECS_TASK_DEFINITION": config.ECSTaskDefinition,
			"HAEDES_ECS_CONTAINER_NAME":  config.ECSContainerName,
			"HAEDES_DYNAMODB_TABLE":      config.DynamoDBTable,
			"HAEDES_SNAPSHOT_TABLE":      config.SnapshotTable,
			"HAEDES_SNAPSHOT_BUCKET":     config.SnapshotBucket,
		} {
			if strings.TrimSpace(value) == "" {
				return Config{}, fmt.Errorf("%w: %s", ErrMissing, name)
			}
		}
		if len(config.PrivateSubnetIDs) == 0 || len(config.SandboxSecurityGroupIDs) == 0 {
			return Config{}, fmt.Errorf("%w: HAEDES_PRIVATE_SUBNET_IDS and HAEDES_SANDBOX_SECURITY_GROUP_IDS", ErrMissing)
		}
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

func boolOr(lookup LookupEnv, name string, fallback bool) bool {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
