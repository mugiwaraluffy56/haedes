package dynamodb

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

type sandboxItem struct {
	ID                         string            `dynamodbav:"sandboxId"`
	OwnerID                    string            `dynamodbav:"ownerId"`
	State                      string            `dynamodbav:"state"`
	Image                      string            `dynamodbav:"image"`
	CPUMillis                  int               `dynamodbav:"cpuMillis"`
	MemoryMiB                  int               `dynamodbav:"memoryMiB"`
	StorageGiB                 int               `dynamodbav:"storageGiB"`
	MaxLifetimeNanos           int64             `dynamodbav:"maxLifetimeNanos"`
	DefaultCommandTimeoutNanos int64             `dynamodbav:"defaultCommandTimeoutNanos"`
	Environment                map[string]string `dynamodbav:"environment,omitempty"`
	Repository                 *repositoryItem   `dynamodbav:"repository,omitempty"`
	SnapshotID                 *string           `dynamodbav:"snapshotId,omitempty"`
	Task                       *taskItem         `dynamodbav:"task,omitempty"`
	CreatedAt                  string            `dynamodbav:"createdAt"`
	ExpiresAt                  string            `dynamodbav:"expiresAt"`
	LastActivityAt             string            `dynamodbav:"lastActivityAt"`
	CurrentCommand             *string           `dynamodbav:"currentCommand,omitempty"`
	SnapshotIDs                []string          `dynamodbav:"snapshotIds,omitempty"`
	FailureReason              string            `dynamodbav:"failureReason,omitempty"`
}

type repositoryItem struct {
	Provider string `dynamodbav:"provider"`
	URL      string `dynamodbav:"url"`
	Ref      string `dynamodbav:"ref"`
	Path     string `dynamodbav:"path"`
}

type taskItem struct {
	ARN          string `dynamodbav:"arn"`
	PrivateIP    string `dynamodbav:"privateIp"`
	Endpoint     string `dynamodbav:"endpoint"`
	RuntimeToken string `dynamodbav:"runtimeToken"`
}

func marshalSandbox(value sandbox.Sandbox) (map[string]types.AttributeValue, error) {
	item := sandboxItem{
		ID:                         string(value.ID),
		OwnerID:                    value.OwnerID,
		State:                      string(value.State),
		Image:                      value.Config.Image,
		CPUMillis:                  value.Config.CPUMillis,
		MemoryMiB:                  value.Config.MemoryMiB,
		StorageGiB:                 value.Config.StorageGiB,
		MaxLifetimeNanos:           int64(value.Config.MaxLifetime),
		DefaultCommandTimeoutNanos: int64(value.Config.DefaultCommandTimeout),
		Environment:                value.Config.Environment,
		CreatedAt:                  value.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt:                  value.ExpiresAt.UTC().Format(time.RFC3339Nano),
		LastActivityAt:             value.LastActivityAt.UTC().Format(time.RFC3339Nano),
		FailureReason:              value.FailureReason,
	}
	if value.Config.Repository != nil {
		item.Repository = &repositoryItem{
			Provider: value.Config.Repository.Provider,
			URL:      value.Config.Repository.URL,
			Ref:      value.Config.Repository.Ref,
			Path:     value.Config.Repository.Path,
		}
	}
	if value.Config.SnapshotID != nil {
		snapshotID := string(*value.Config.SnapshotID)
		item.SnapshotID = &snapshotID
	}
	if value.Task != nil {
		item.Task = &taskItem{
			ARN:          value.Task.ARN,
			PrivateIP:    value.Task.PrivateIP,
			Endpoint:     value.Task.Endpoint,
			RuntimeToken: value.Task.RuntimeToken,
		}
	}
	if value.CurrentCommand != nil {
		currentCommand := string(*value.CurrentCommand)
		item.CurrentCommand = &currentCommand
	}
	item.SnapshotIDs = make([]string, len(value.SnapshotIDs))
	for index, snapshotID := range value.SnapshotIDs {
		item.SnapshotIDs[index] = string(snapshotID)
	}
	return attributevalue.MarshalMap(item)
}

func unmarshalSandbox(values map[string]types.AttributeValue) (sandbox.Sandbox, error) {
	var item sandboxItem
	if err := attributevalue.UnmarshalMap(values, &item); err != nil {
		return sandbox.Sandbox{}, fmt.Errorf("decode sandbox item: %w", err)
	}
	createdAt, err := parseTime("createdAt", item.CreatedAt)
	if err != nil {
		return sandbox.Sandbox{}, err
	}
	expiresAt, err := parseTime("expiresAt", item.ExpiresAt)
	if err != nil {
		return sandbox.Sandbox{}, err
	}
	lastActivityAt, err := parseTime("lastActivityAt", item.LastActivityAt)
	if err != nil {
		return sandbox.Sandbox{}, err
	}

	value := sandbox.Sandbox{
		ID:      sandbox.SandboxID(item.ID),
		OwnerID: item.OwnerID,
		State:   sandbox.State(item.State),
		Config: sandbox.SandboxConfig{
			Image:                 item.Image,
			CPUMillis:             item.CPUMillis,
			MemoryMiB:             item.MemoryMiB,
			StorageGiB:            item.StorageGiB,
			MaxLifetime:           time.Duration(item.MaxLifetimeNanos),
			DefaultCommandTimeout: time.Duration(item.DefaultCommandTimeoutNanos),
			Environment:           item.Environment,
		},
		CreatedAt:      createdAt,
		ExpiresAt:      expiresAt,
		LastActivityAt: lastActivityAt,
		FailureReason:  item.FailureReason,
	}
	if item.Repository != nil {
		value.Repository = &sandbox.RepositoryConfig{
			Provider: item.Repository.Provider,
			URL:      item.Repository.URL,
			Ref:      item.Repository.Ref,
			Path:     item.Repository.Path,
		}
		value.Config.Repository = value.Repository
	}
	if item.SnapshotID != nil {
		snapshotID := sandbox.SnapshotID(*item.SnapshotID)
		value.Config.SnapshotID = &snapshotID
	}
	if item.Task != nil {
		value.Task = &sandbox.TaskRef{
			ARN:          item.Task.ARN,
			PrivateIP:    item.Task.PrivateIP,
			Endpoint:     item.Task.Endpoint,
			RuntimeToken: item.Task.RuntimeToken,
		}
	}
	if item.CurrentCommand != nil {
		currentCommand := sandbox.CommandID(*item.CurrentCommand)
		value.CurrentCommand = &currentCommand
	}
	value.SnapshotIDs = make([]sandbox.SnapshotID, len(item.SnapshotIDs))
	for index, snapshotID := range item.SnapshotIDs {
		value.SnapshotIDs[index] = sandbox.SnapshotID(snapshotID)
	}
	return value, nil
}

func parseTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("decode sandbox %s: %w", field, err)
	}
	return parsed, nil
}

type listCursor struct {
	OwnerID   string `json:"ownerId"`
	CreatedAt string `json:"createdAt"`
	SandboxID string `json:"sandboxId"`
}

func encodeCursor(cursor listCursor) string {
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeCursor(value string) (listCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return listCursor{}, fmt.Errorf("decode pagination cursor: %w", err)
	}
	var cursor listCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.OwnerID == "" || cursor.CreatedAt == "" || cursor.SandboxID == "" {
		return listCursor{}, fmt.Errorf("decode pagination cursor: invalid cursor")
	}
	return cursor, nil
}
