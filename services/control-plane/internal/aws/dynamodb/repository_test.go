package dynamodb

import (
	"context"
	"errors"
	"testing"
	"time"

	sdkDynamoDB "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"

	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestRepositoryRoundTripsSandboxAndPaginatesByOwner(t *testing.T) {
	client := &fakeClient{items: make(map[string]map[string]types.AttributeValue)}
	repository, err := NewRepository(client, Config{TableName: "metadata", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	want := testSandbox("sbx_001", "owner-1", sandbox.StateRequested)
	if err := repository.Create(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repository.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.OwnerID != want.OwnerID || got.Config.Repository.URL != want.Config.Repository.URL || got.Task == nil || got.Task.RuntimeToken != want.Task.RuntimeToken {
		t.Fatalf("round trip = %+v", got)
	}

	client.queryOutput = &sdkDynamoDB.QueryOutput{
		Items: []map[string]types.AttributeValue{client.items[string(want.ID)]},
		LastEvaluatedKey: map[string]types.AttributeValue{
			"ownerId":   stringValue(want.OwnerID),
			"createdAt": stringValue(want.CreatedAt.UTC().Format(time.RFC3339Nano)),
			"sandboxId": stringValue(string(want.ID)),
		},
	}
	page, err := repository.List(context.Background(), want.OwnerID, "", 1)
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("page = %+v, err = %v", page, err)
	}
}

func TestRepositoryMapsMissingAndConditionalStateConflicts(t *testing.T) {
	client := &fakeClient{items: make(map[string]map[string]types.AttributeValue)}
	repository, err := NewRepository(client, Config{TableName: "metadata", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(context.Background(), "missing"); !errors.Is(err, sandbox.ErrNotFound) {
		t.Fatalf("missing get error = %v", err)
	}

	value := testSandbox("sbx_002", "owner-1", sandbox.StateRequested)
	client.items[string(value.ID)], _ = marshalSandbox(value)
	if err := repository.UpdateState(context.Background(), value.ID, sandbox.StateRunning, sandbox.StateStopped, ""); !errors.Is(err, sandbox.ErrConflict) {
		t.Fatalf("state mismatch error = %v", err)
	}
	client.updateErr = &smithy.GenericAPIError{Code: "ConditionalCheckFailedException"}
	if err := repository.UpdateState(context.Background(), value.ID, sandbox.StateRequested, sandbox.StateProvisioning, ""); !errors.Is(err, sandbox.ErrConflict) {
		t.Fatalf("conditional error = %v", err)
	}
}

func TestRepositoryRetriesTransientDynamoErrors(t *testing.T) {
	client := &fakeClient{items: make(map[string]map[string]types.AttributeValue), getErrs: []error{
		&smithy.GenericAPIError{Code: "ProvisionedThroughputExceededException", Fault: smithy.FaultServer},
		nil,
	}}
	value := testSandbox("sbx_003", "owner-1", sandbox.StateRunning)
	client.items[string(value.ID)], _ = marshalSandbox(value)
	repository, err := NewRepository(client, Config{TableName: "metadata", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(context.Background(), value.ID); err != nil {
		t.Fatal(err)
	}
	if client.getCalls != 2 {
		t.Fatalf("get calls = %d", client.getCalls)
	}
}

type fakeClient struct {
	items       map[string]map[string]types.AttributeValue
	queryOutput *sdkDynamoDB.QueryOutput
	getErrs     []error
	updateErr   error
	getCalls    int
}

func (client *fakeClient) PutItem(_ context.Context, input *sdkDynamoDB.PutItemInput, _ ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.PutItemOutput, error) {
	if client.items == nil {
		client.items = make(map[string]map[string]types.AttributeValue)
	}
	id := stringAttribute(input.Item, "sandboxId")
	if _, exists := client.items[id]; exists {
		return nil, &smithy.GenericAPIError{Code: "ConditionalCheckFailedException"}
	}
	client.items[id] = input.Item
	return &sdkDynamoDB.PutItemOutput{}, nil
}

func (client *fakeClient) GetItem(_ context.Context, input *sdkDynamoDB.GetItemInput, _ ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.GetItemOutput, error) {
	client.getCalls++
	if len(client.getErrs) > 0 {
		err := client.getErrs[0]
		client.getErrs = client.getErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	id := stringAttribute(input.Key, "sandboxId")
	return &sdkDynamoDB.GetItemOutput{Item: client.items[id]}, nil
}

func (client *fakeClient) Query(context.Context, *sdkDynamoDB.QueryInput, ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.QueryOutput, error) {
	return client.queryOutput, nil
}

func (client *fakeClient) UpdateItem(_ context.Context, _ *sdkDynamoDB.UpdateItemInput, _ ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.UpdateItemOutput, error) {
	if client.updateErr != nil {
		return nil, client.updateErr
	}
	return &sdkDynamoDB.UpdateItemOutput{}, nil
}

func testSandbox(id, owner string, state sandbox.State) sandbox.Sandbox {
	currentCommand := sandbox.CommandID("cmd_001")
	task := &sandbox.TaskRef{ARN: "arn:task/1", PrivateIP: "10.0.0.4", Endpoint: "http://10.0.0.4:8080", RuntimeToken: "secret"}
	now := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	repository := &sandbox.RepositoryConfig{Provider: "github", URL: "https://github.com/example/repo", Ref: "main", Path: "/workspace/repo"}
	return sandbox.Sandbox{
		ID: ownerSandboxID(id), OwnerID: owner, State: state, Task: task,
		Config: sandbox.SandboxConfig{
			Image: "runtime:dev", CPUMillis: 512, MemoryMiB: 1024, StorageGiB: 20,
			MaxLifetime: 30 * time.Minute, DefaultCommandTimeout: 45 * time.Second,
			Environment: map[string]string{"MODE": "test"}, Repository: repository,
		},
		Repository: repository,
		CreatedAt:  now, ExpiresAt: now.Add(time.Hour), LastActivityAt: now,
		CurrentCommand: &currentCommand, SnapshotIDs: []sandbox.SnapshotID{"snp_001"},
	}
}

func ownerSandboxID(id string) sandbox.SandboxID { return sandbox.SandboxID(id) }

func noWaitRetry() internalaws.RetryPolicy {
	return internalaws.RetryPolicy{MaxAttempts: 3, Sleep: func(context.Context, time.Duration) error { return nil }}
}
