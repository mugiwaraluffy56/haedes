package config

import (
	"strings"
	"testing"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	config, err := LoadFrom(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if config.Environment != EnvironmentDevelopment || config.AWSRegion != "us-east-1" || config.APIBaseURL != "http://localhost:8080" {
		t.Fatalf("unexpected defaults: %#v", config)
	}
}

func TestLoadProductionRequiresSigningAndAWSConfiguration(t *testing.T) {
	values := map[string]string{"HAEDES_ENV": EnvironmentProduction}
	if _, err := LoadFrom(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "HAEDES_SIGNING_SECRET") {
		t.Fatalf("expected missing signing secret error, got %v", err)
	}

	values["HAEDES_SIGNING_SECRET"] = strings.Repeat("s", 32)
	values["HAEDES_AWS_REGION"] = ""
	if _, err := LoadFrom(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "HAEDES_AWS_REGION") {
		t.Fatalf("expected missing AWS region error, got %v", err)
	}
}

func TestLoadProductionAcceptsCompleteConfiguration(t *testing.T) {
	values := map[string]string{
		"HAEDES_ENV":            EnvironmentProduction,
		"HAEDES_SIGNING_SECRET": strings.Repeat("s", 32),
		"HAEDES_AWS_REGION":     "eu-west-1",
		"HAEDES_API_BASE_URL":   "https://api.example.test",
	}
	config, err := LoadFrom(mapLookup(values))
	if err != nil {
		t.Fatal(err)
	}
	if string(config.SigningSecret) != values["HAEDES_SIGNING_SECRET"] {
		t.Fatal("signing secret was not loaded")
	}
}

func TestLoadAWSModeRequiresAdapterSettings(t *testing.T) {
	values := map[string]string{"HAEDES_AWS_ENABLED": "true"}
	if _, err := LoadFrom(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "HAEDES_ECS_CLUSTER") {
		t.Fatalf("expected missing ECS configuration error, got %v", err)
	}

	values["HAEDES_ECS_CLUSTER"] = "cluster"
	values["HAEDES_ECS_TASK_DEFINITION"] = "task"
	values["HAEDES_PRIVATE_SUBNET_IDS"] = "subnet-a, subnet-b"
	values["HAEDES_SANDBOX_SECURITY_GROUP_IDS"] = "sg-runtime"
	values["HAEDES_DYNAMODB_TABLE"] = "metadata"
	values["HAEDES_SNAPSHOT_TABLE"] = "snapshots"
	values["HAEDES_SNAPSHOT_BUCKET"] = "bucket"
	config, err := LoadFrom(mapLookup(values))
	if err != nil {
		t.Fatal(err)
	}
	if !config.AWSEnabled || len(config.PrivateSubnetIDs) != 2 || config.SnapshotTable != "snapshots" {
		t.Fatalf("unexpected AWS configuration: %#v", config)
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
