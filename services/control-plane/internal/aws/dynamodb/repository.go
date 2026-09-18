package dynamodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	sdkDynamoDB "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"

	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

type Client interface {
	PutItem(context.Context, *sdkDynamoDB.PutItemInput, ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.PutItemOutput, error)
	GetItem(context.Context, *sdkDynamoDB.GetItemInput, ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.GetItemOutput, error)
	Query(context.Context, *sdkDynamoDB.QueryInput, ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.QueryOutput, error)
	UpdateItem(context.Context, *sdkDynamoDB.UpdateItemInput, ...func(*sdkDynamoDB.Options)) (*sdkDynamoDB.UpdateItemOutput, error)
}

type Config struct {
	TableName string
	IndexName string
	Retry     internalaws.RetryPolicy
}

type Repository struct {
	client Client
	config Config
}

var _ sandbox.SandboxRepository = (*Repository)(nil)

func NewRepository(client Client, config Config) (*Repository, error) {
	if client == nil {
		return nil, errors.New("dynamodb client is required")
	}
	if config.TableName == "" {
		return nil, errors.New("dynamodb table name is required")
	}
	if config.IndexName == "" {
		config.IndexName = "owner-createdAt-index"
	}
	return &Repository{client: client, config: config}, nil
}

func (repository *Repository) Create(ctx context.Context, value sandbox.Sandbox) error {
	item, err := marshalSandbox(value)
	if err != nil {
		return err
	}
	err = repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.PutItem(callContext, &sdkDynamoDB.PutItemInput{
			TableName:           aws.String(repository.config.TableName),
			Item:                item,
			ConditionExpression: aws.String("attribute_not_exists(#id)"),
			ExpressionAttributeNames: map[string]string{
				"#id": "sandboxId",
			},
		})
		return callErr
	})
	return mapError(err)
}

func (repository *Repository) Get(ctx context.Context, id sandbox.SandboxID) (sandbox.Sandbox, error) {
	if id == "" {
		return sandbox.Sandbox{}, sandbox.ErrNotFound
	}
	var output *sdkDynamoDB.GetItemOutput
	err := repository.call(ctx, func(callContext context.Context) error {
		var callErr error
		output, callErr = repository.client.GetItem(callContext, &sdkDynamoDB.GetItemInput{
			TableName:      aws.String(repository.config.TableName),
			Key:            map[string]types.AttributeValue{"sandboxId": stringValue(string(id))},
			ConsistentRead: aws.Bool(true),
		})
		return callErr
	})
	if err != nil {
		return sandbox.Sandbox{}, mapError(err)
	}
	if output == nil || len(output.Item) == 0 {
		return sandbox.Sandbox{}, sandbox.ErrNotFound
	}
	return unmarshalSandbox(output.Item)
}

func (repository *Repository) List(ctx context.Context, ownerID, cursor string, limit int) (sandbox.Page[sandbox.Sandbox], error) {
	if ownerID == "" {
		return sandbox.Page[sandbox.Sandbox]{}, errors.New("owner ID is required")
	}
	if limit <= 0 {
		return sandbox.Page[sandbox.Sandbox]{}, errors.New("limit must be positive")
	}
	input := &sdkDynamoDB.QueryInput{
		TableName:              aws.String(repository.config.TableName),
		IndexName:              aws.String(repository.config.IndexName),
		KeyConditionExpression: aws.String("#owner = :owner"),
		ExpressionAttributeNames: map[string]string{
			"#owner": "ownerId",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":owner": stringValue(ownerID),
		},
		Limit:            aws.Int32(int32(limit)),
		ScanIndexForward: aws.Bool(true),
	}
	if cursor != "" {
		decoded, err := decodeCursor(cursor)
		if err != nil {
			return sandbox.Page[sandbox.Sandbox]{}, err
		}
		if decoded.OwnerID != ownerID {
			return sandbox.Page[sandbox.Sandbox]{}, errors.New("pagination cursor belongs to another owner")
		}
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"ownerId":   stringValue(decoded.OwnerID),
			"createdAt": stringValue(decoded.CreatedAt),
			"sandboxId": stringValue(decoded.SandboxID),
		}
	}

	var output *sdkDynamoDB.QueryOutput
	err := repository.call(ctx, func(callContext context.Context) error {
		var callErr error
		output, callErr = repository.client.Query(callContext, input)
		return callErr
	})
	if err != nil {
		return sandbox.Page[sandbox.Sandbox]{}, mapError(err)
	}
	if output == nil {
		return sandbox.Page[sandbox.Sandbox]{}, errors.New("dynamodb query returned an empty response")
	}
	page := sandbox.Page[sandbox.Sandbox]{Items: make([]sandbox.Sandbox, 0, len(output.Items))}
	for _, item := range output.Items {
		value, decodeErr := unmarshalSandbox(item)
		if decodeErr != nil {
			return sandbox.Page[sandbox.Sandbox]{}, decodeErr
		}
		page.Items = append(page.Items, value)
	}
	if len(output.LastEvaluatedKey) > 0 {
		page.NextCursor = encodeCursor(listCursor{
			OwnerID:   stringAttribute(output.LastEvaluatedKey, "ownerId"),
			CreatedAt: stringAttribute(output.LastEvaluatedKey, "createdAt"),
			SandboxID: stringAttribute(output.LastEvaluatedKey, "sandboxId"),
		})
	}
	return page, nil
}

func (repository *Repository) UpdateState(ctx context.Context, id sandbox.SandboxID, expected, next sandbox.State, reason string) error {
	current, err := repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.State != expected {
		return fmt.Errorf("%w: expected %s, got %s", sandbox.ErrConflict, expected, current.State)
	}
	err = repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.UpdateItem(callContext, &sdkDynamoDB.UpdateItemInput{
			TableName:           aws.String(repository.config.TableName),
			Key:                 map[string]types.AttributeValue{"sandboxId": stringValue(string(id))},
			UpdateExpression:    aws.String("SET #state = :next, #reason = :reason"),
			ConditionExpression: aws.String("#state = :expected"),
			ExpressionAttributeNames: map[string]string{
				"#state":  "state",
				"#reason": "failureReason",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":expected": stringValue(string(expected)),
				":next":     stringValue(string(next)),
				":reason":   stringValue(reason),
			},
		})
		return callErr
	})
	return mapError(err)
}

func (repository *Repository) SaveTaskEndpoint(ctx context.Context, id sandbox.SandboxID, task sandbox.TaskRef) error {
	if _, err := repository.Get(ctx, id); err != nil {
		return err
	}
	encodedTask, err := attributevalue.Marshal(taskItem{
		ARN:          task.ARN,
		PrivateIP:    task.PrivateIP,
		Endpoint:     task.Endpoint,
		RuntimeToken: task.RuntimeToken,
	})
	if err != nil {
		return fmt.Errorf("encode task endpoint: %w", err)
	}
	err = repository.call(ctx, func(callContext context.Context) error {
		_, callErr := repository.client.UpdateItem(callContext, &sdkDynamoDB.UpdateItemInput{
			TableName:           aws.String(repository.config.TableName),
			Key:                 map[string]types.AttributeValue{"sandboxId": stringValue(string(id))},
			UpdateExpression:    aws.String("SET #task = :task"),
			ConditionExpression: aws.String("attribute_exists(#id)"),
			ExpressionAttributeNames: map[string]string{
				"#id":   "sandboxId",
				"#task": "task",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{":task": encodedTask},
		})
		return callErr
	})
	return mapError(err)
}

func (repository *Repository) call(ctx context.Context, operation func(context.Context) error) error {
	return internalaws.Retry(ctx, repository.config.Retry, operation)
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) && apiError.ErrorCode() == "ConditionalCheckFailedException" {
		return fmt.Errorf("%w: conditional write rejected", sandbox.ErrConflict)
	}
	return err
}

func stringValue(value string) types.AttributeValue {
	return &types.AttributeValueMemberS{Value: value}
}

func stringAttribute(values map[string]types.AttributeValue, name string) string {
	value, ok := values[name].(*types.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return value.Value
}
