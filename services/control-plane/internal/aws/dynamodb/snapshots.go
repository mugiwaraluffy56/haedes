package dynamodb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	sdkDynamoDB "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"

	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

type SnapshotRepository struct {
	client Client
	config Config
}

var _ sandbox.SnapshotRepository = (*SnapshotRepository)(nil)

func NewSnapshotRepository(client Client, config Config) (*SnapshotRepository, error) {
	if client == nil || config.TableName == "" {
		return nil, errors.New("dynamodb snapshot table and client are required")
	}
	if config.IndexName == "" {
		config.IndexName = "sandbox-createdAt-index"
	}
	return &SnapshotRepository{client: client, config: config}, nil
}

func (repository *SnapshotRepository) Create(ctx context.Context, value sandbox.SnapshotMetadata) error {
	item, err := marshalSnapshot(value)
	if err != nil {
		return err
	}
	err = repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.PutItem(callContext, &sdkDynamoDB.PutItemInput{
			TableName: aws.String(repository.config.TableName), Item: item,
			ConditionExpression:      aws.String("attribute_not_exists(#id)"),
			ExpressionAttributeNames: map[string]string{"#id": "snapshotId"},
		})
		return callErr
	})
	return mapSnapshotError(err)
}

func (repository *SnapshotRepository) Get(ctx context.Context, id sandbox.SnapshotID) (sandbox.SnapshotMetadata, error) {
	if id == "" {
		return sandbox.SnapshotMetadata{}, sandbox.ErrNotFound
	}
	var output *sdkDynamoDB.GetItemOutput
	err := repository.call(ctx, func(callContext context.Context) error {
		var callErr error
		output, callErr = repository.client.GetItem(callContext, &sdkDynamoDB.GetItemInput{
			TableName: aws.String(repository.config.TableName), Key: map[string]types.AttributeValue{"snapshotId": &types.AttributeValueMemberS{Value: string(id)}}, ConsistentRead: aws.Bool(true),
		})
		return callErr
	})
	if err != nil {
		return sandbox.SnapshotMetadata{}, mapSnapshotError(err)
	}
	if output == nil || len(output.Item) == 0 {
		return sandbox.SnapshotMetadata{}, sandbox.ErrNotFound
	}
	return unmarshalSnapshot(output.Item)
}

func (repository *SnapshotRepository) List(ctx context.Context, sandboxID sandbox.SandboxID, cursor string, limit int) (sandbox.Page[sandbox.SnapshotMetadata], error) {
	if sandboxID == "" || limit <= 0 {
		return sandbox.Page[sandbox.SnapshotMetadata]{}, errors.New("sandbox ID and positive limit are required")
	}
	input := &sdkDynamoDB.QueryInput{
		TableName: aws.String(repository.config.TableName), IndexName: aws.String(repository.config.IndexName),
		KeyConditionExpression:    aws.String("#sandbox = :sandbox"),
		ExpressionAttributeNames:  map[string]string{"#sandbox": "sandboxId"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":sandbox": &types.AttributeValueMemberS{Value: string(sandboxID)}},
		Limit:                     aws.Int32(int32(limit)), ScanIndexForward: aws.Bool(true),
	}
	if cursor != "" {
		key, err := decodeSnapshotCursor(cursor)
		if err != nil {
			return sandbox.Page[sandbox.SnapshotMetadata]{}, err
		}
		input.ExclusiveStartKey = key
	}
	var output *sdkDynamoDB.QueryOutput
	err := repository.call(ctx, func(callContext context.Context) error {
		var callErr error
		output, callErr = repository.client.Query(callContext, input)
		return callErr
	})
	if err != nil {
		return sandbox.Page[sandbox.SnapshotMetadata]{}, mapSnapshotError(err)
	}
	if output == nil {
		return sandbox.Page[sandbox.SnapshotMetadata]{Items: []sandbox.SnapshotMetadata{}}, nil
	}
	page := sandbox.Page[sandbox.SnapshotMetadata]{Items: make([]sandbox.SnapshotMetadata, 0, len(output.Items))}
	for _, item := range output.Items {
		value, err := unmarshalSnapshot(item)
		if err != nil {
			return sandbox.Page[sandbox.SnapshotMetadata]{}, err
		}
		page.Items = append(page.Items, value)
	}
	if len(output.LastEvaluatedKey) > 0 {
		page.NextCursor, _ = encodeSnapshotCursor(output.LastEvaluatedKey)
	}
	return page, nil
}

func (repository *SnapshotRepository) Update(ctx context.Context, value sandbox.SnapshotMetadata) error {
	item, err := marshalSnapshot(value)
	if err != nil {
		return err
	}
	err = repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.PutItem(callContext, &sdkDynamoDB.PutItemInput{TableName: aws.String(repository.config.TableName), Item: item})
		return callErr
	})
	return mapSnapshotError(err)
}

func (repository *SnapshotRepository) UpdateState(ctx context.Context, id sandbox.SnapshotID, expected, next string) error {
	if id == "" || expected == "" || next == "" {
		return errors.New("snapshot ID and states are required")
	}
	err := repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.UpdateItem(callContext, &sdkDynamoDB.UpdateItemInput{
			TableName: aws.String(repository.config.TableName), Key: map[string]types.AttributeValue{"snapshotId": &types.AttributeValueMemberS{Value: string(id)}},
			UpdateExpression: aws.String("SET #state = :next"), ConditionExpression: aws.String("#state = :expected"),
			ExpressionAttributeNames: map[string]string{"#state": "state"}, ExpressionAttributeValues: map[string]types.AttributeValue{":next": &types.AttributeValueMemberS{Value: next}, ":expected": &types.AttributeValueMemberS{Value: expected}},
		})
		return callErr
	})
	return mapSnapshotError(err)
}

func (repository *SnapshotRepository) call(ctx context.Context, operation func(context.Context) error) error {
	return internalaws.Retry(ctx, repository.config.Retry, operation)
}

type snapshotItem struct {
	SnapshotID string  `dynamodbav:"snapshotId"`
	SandboxID  string  `dynamodbav:"sandboxId"`
	State      string  `dynamodbav:"state"`
	ObjectKey  string  `dynamodbav:"objectKey,omitempty"`
	ByteSize   int64   `dynamodbav:"byteSize"`
	SHA256     string  `dynamodbav:"sha256,omitempty"`
	CreatedAt  string  `dynamodbav:"createdAt"`
	ExpiresAt  *string `dynamodbav:"expiresAt,omitempty"`
}

func marshalSnapshot(value sandbox.SnapshotMetadata) (map[string]types.AttributeValue, error) {
	created := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	item := snapshotItem{SnapshotID: string(value.ID), SandboxID: string(value.SandboxID), State: value.State, ObjectKey: value.ObjectKey, ByteSize: value.ByteSize, SHA256: value.SHA256, CreatedAt: created}
	if value.ExpiresAt != nil {
		expires := value.ExpiresAt.UTC().Format(time.RFC3339Nano)
		item.ExpiresAt = &expires
	}
	return attributevalue.MarshalMap(item)
}

func unmarshalSnapshot(item map[string]types.AttributeValue) (sandbox.SnapshotMetadata, error) {
	var value snapshotItem
	if err := attributevalue.UnmarshalMap(item, &value); err != nil {
		return sandbox.SnapshotMetadata{}, err
	}
	created, err := time.Parse(time.RFC3339Nano, value.CreatedAt)
	if err != nil {
		return sandbox.SnapshotMetadata{}, err
	}
	result := sandbox.SnapshotMetadata{ID: sandbox.SnapshotID(value.SnapshotID), SandboxID: sandbox.SandboxID(value.SandboxID), State: value.State, ObjectKey: value.ObjectKey, ByteSize: value.ByteSize, SHA256: value.SHA256, CreatedAt: created}
	if value.ExpiresAt != nil {
		expires, parseErr := time.Parse(time.RFC3339Nano, *value.ExpiresAt)
		if parseErr != nil {
			return sandbox.SnapshotMetadata{}, parseErr
		}
		result.ExpiresAt = &expires
	}
	return result, nil
}

func mapSnapshotError(err error) error {
	if err == nil {
		return nil
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) && apiError.ErrorCode() == "ConditionalCheckFailedException" {
		return sandbox.ErrConflict
	}
	return fmt.Errorf("dynamodb snapshot operation: %w", err)
}

func encodeSnapshotCursor(key map[string]types.AttributeValue) (string, error) {
	cursor := snapshotCursor{}
	for name, value := range key {
		member, ok := value.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		switch name {
		case "snapshotId":
			cursor.SnapshotID = member.Value
		case "sandboxId":
			cursor.SandboxID = member.Value
		case "createdAt":
			cursor.CreatedAt = member.Value
		}
	}
	body, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func decodeSnapshotCursor(encoded string) (map[string]types.AttributeValue, error) {
	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("invalid snapshot cursor")
	}
	var cursor snapshotCursor
	if err := json.Unmarshal(body, &cursor); err != nil {
		return nil, errors.New("invalid snapshot cursor")
	}
	if cursor.SnapshotID == "" || cursor.SandboxID == "" || cursor.CreatedAt == "" {
		return nil, errors.New("invalid snapshot cursor")
	}
	return map[string]types.AttributeValue{
		"snapshotId": &types.AttributeValueMemberS{Value: cursor.SnapshotID},
		"sandboxId":  &types.AttributeValueMemberS{Value: cursor.SandboxID},
		"createdAt":  &types.AttributeValueMemberS{Value: cursor.CreatedAt},
	}, nil
}

type snapshotCursor struct {
	SnapshotID string `json:"snapshotId"`
	SandboxID  string `json:"sandboxId"`
	CreatedAt  string `json:"createdAt"`
}
