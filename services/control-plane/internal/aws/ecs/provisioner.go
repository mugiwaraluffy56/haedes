package ecs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsECS "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	ErrTaskNotFound              = errors.New("ecs task not found")
	ErrTaskLaunchFailed          = errors.New("ecs task launch failed")
	ErrTaskStoppedBeforeReady    = errors.New("ecs task stopped before becoming ready")
	ErrNetworkAttachmentMissing  = errors.New("ecs network attachment is unavailable")
	ErrInvalidProvisionRequest   = errors.New("invalid ecs provision request")
	ErrInvalidProvisionerConfig  = errors.New("invalid ecs provisioner configuration")
	ErrRuntimeCredentialOverride = errors.New("runtime credential environment overrides are not allowed")
)

// Client is the narrow ECS surface used by Provisioner. Keeping the SDK behind
// this interface makes launch, retry, and attachment behavior unit-testable.
type Client interface {
	RunTask(context.Context, *awsECS.RunTaskInput, ...func(*awsECS.Options)) (*awsECS.RunTaskOutput, error)
	DescribeTasks(context.Context, *awsECS.DescribeTasksInput, ...func(*awsECS.Options)) (*awsECS.DescribeTasksOutput, error)
	StopTask(context.Context, *awsECS.StopTaskInput, ...func(*awsECS.Options)) (*awsECS.StopTaskOutput, error)
}

type Config struct {
	Cluster          string
	TaskDefinition   string
	ContainerName    string
	PrivateSubnetIDs []string
	SecurityGroupIDs []string
	RuntimePort      int
	StartTimeout     time.Duration
	DescribeTimeout  time.Duration
	StopTimeout      time.Duration
	PollInterval     time.Duration
	MaxAttempts      int
}

type Provisioner struct {
	client Client
	config Config
}

func NewProvisioner(client Client, config Config) (*Provisioner, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: ECS client is required", ErrInvalidProvisionerConfig)
	}
	if config.Cluster == "" || config.TaskDefinition == "" || config.ContainerName == "" {
		return nil, fmt.Errorf("%w: cluster, task definition, and container name are required", ErrInvalidProvisionerConfig)
	}
	if len(config.PrivateSubnetIDs) == 0 || len(config.SecurityGroupIDs) == 0 {
		return nil, fmt.Errorf("%w: private subnets and security groups are required", ErrInvalidProvisionerConfig)
	}
	if config.RuntimePort == 0 {
		config.RuntimePort = 8080
	}
	if config.StartTimeout == 0 {
		config.StartTimeout = 2 * time.Minute
	}
	if config.DescribeTimeout == 0 {
		config.DescribeTimeout = 15 * time.Second
	}
	if config.StopTimeout == 0 {
		config.StopTimeout = 30 * time.Second
	}
	if config.PollInterval == 0 {
		config.PollInterval = 250 * time.Millisecond
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.RuntimePort < 1 || config.RuntimePort > 65535 || config.StartTimeout <= 0 || config.DescribeTimeout <= 0 || config.StopTimeout <= 0 || config.PollInterval <= 0 || config.MaxAttempts < 1 {
		return nil, fmt.Errorf("%w: timeouts, port, and retry settings are out of bounds", ErrInvalidProvisionerConfig)
	}

	return &Provisioner{client: client, config: cloneConfig(config)}, nil
}

func (provisioner *Provisioner) Start(ctx context.Context, request sandbox.ProvisionRequest) (sandbox.TaskRef, error) {
	if request.SandboxID == "" || request.Token == "" {
		return sandbox.TaskRef{}, fmt.Errorf("%w: sandbox ID and runtime token are required", ErrInvalidProvisionRequest)
	}
	environment, err := runtimeEnvironment(request)
	if err != nil {
		return sandbox.TaskRef{}, err
	}

	operationContext, cancel := context.WithTimeout(ctx, provisioner.config.StartTimeout)
	defer cancel()

	input := &awsECS.RunTaskInput{
		Cluster:              aws.String(provisioner.config.Cluster),
		TaskDefinition:       aws.String(provisioner.config.TaskDefinition),
		Count:                aws.Int32(1),
		ClientToken:          aws.String("haedes-" + string(request.SandboxID)),
		EnableECSManagedTags: true,
		LaunchType:           awsTypes.LaunchTypeFargate,
		NetworkConfiguration: &awsTypes.NetworkConfiguration{AwsvpcConfiguration: &awsTypes.AwsVpcConfiguration{
			Subnets:        append([]string(nil), provisioner.config.PrivateSubnetIDs...),
			SecurityGroups: append([]string(nil), provisioner.config.SecurityGroupIDs...),
			AssignPublicIp: awsTypes.AssignPublicIpDisabled,
		}},
		Overrides: &awsTypes.TaskOverride{ContainerOverrides: []awsTypes.ContainerOverride{{
			Name:        aws.String(provisioner.config.ContainerName),
			Environment: environment,
		}}},
		Tags: []awsTypes.Tag{
			{Key: aws.String("haedes:sandbox-id"), Value: aws.String(string(request.SandboxID))},
		},
	}

	var output *awsECS.RunTaskOutput
	err = retry(operationContext, provisioner.config.MaxAttempts, func(callContext context.Context) error {
		var callErr error
		output, callErr = provisioner.client.RunTask(callContext, input)
		return callErr
	})
	if err != nil {
		return sandbox.TaskRef{}, fmt.Errorf("%w: %v", ErrTaskLaunchFailed, err)
	}
	if output == nil || len(output.Tasks) == 0 {
		return sandbox.TaskRef{}, launchFailure(output)
	}
	taskARN := aws.ToString(output.Tasks[0].TaskArn)
	if taskARN == "" {
		return sandbox.TaskRef{}, fmt.Errorf("%w: ECS returned a task without an ARN", ErrTaskLaunchFailed)
	}

	task, err := provisioner.waitForRunning(operationContext, taskARN)
	if err != nil {
		_ = provisioner.stopBestEffort(context.Background(), taskARN)
		return sandbox.TaskRef{}, err
	}
	privateIP := privateIPv4Address(task)
	if privateIP == "" {
		_ = provisioner.stopBestEffort(context.Background(), taskARN)
		return sandbox.TaskRef{}, fmt.Errorf("%w: task %s", ErrNetworkAttachmentMissing, taskARN)
	}

	return sandbox.TaskRef{
		ARN:          taskARN,
		PrivateIP:    privateIP,
		Endpoint:     fmt.Sprintf("http://%s:%d", privateIP, provisioner.config.RuntimePort),
		RuntimeToken: request.Token,
	}, nil
}

func (provisioner *Provisioner) Describe(ctx context.Context, taskARN string) (sandbox.TaskStatus, error) {
	if taskARN == "" {
		return sandbox.TaskStatus{}, ErrTaskNotFound
	}
	operationContext, cancel := context.WithTimeout(ctx, provisioner.config.DescribeTimeout)
	defer cancel()
	task, err := provisioner.describeTask(operationContext, taskARN)
	if err != nil {
		return sandbox.TaskStatus{}, err
	}
	phase := strings.ToLower(aws.ToString(task.LastStatus))
	if phase == "" {
		phase = "unknown"
	}
	return sandbox.TaskStatus{
		ARN:       taskARN,
		Phase:     phase,
		PrivateIP: privateIPv4Address(task),
		Reason:    aws.ToString(task.StoppedReason),
	}, nil
}

func (provisioner *Provisioner) Stop(ctx context.Context, taskARN string) error {
	if taskARN == "" {
		return nil
	}
	operationContext, cancel := context.WithTimeout(ctx, provisioner.config.StopTimeout)
	defer cancel()
	err := retry(operationContext, provisioner.config.MaxAttempts, func(callContext context.Context) error {
		_, callErr := provisioner.client.StopTask(callContext, &awsECS.StopTaskInput{
			Cluster: aws.String(provisioner.config.Cluster),
			Task:    aws.String(taskARN),
			Reason:  aws.String("haedes sandbox cleanup"),
		})
		return callErr
	})
	if isMissingTask(err) || isAlreadyStopped(err) {
		return nil
	}
	return err
}

func (provisioner *Provisioner) waitForRunning(ctx context.Context, taskARN string) (awsTypes.Task, error) {
	for {
		task, err := provisioner.describeTask(ctx, taskARN)
		if err != nil {
			return awsTypes.Task{}, err
		}
		status := strings.ToUpper(aws.ToString(task.LastStatus))
		switch status {
		case "RUNNING":
			return task, nil
		case "STOPPED":
			return awsTypes.Task{}, fmt.Errorf("%w: %s", ErrTaskStoppedBeforeReady, aws.ToString(task.StoppedReason))
		}
		waiter := time.NewTimer(provisioner.config.PollInterval)
		select {
		case <-ctx.Done():
			waiter.Stop()
			return awsTypes.Task{}, ctx.Err()
		case <-waiter.C:
		}
	}
}

func (provisioner *Provisioner) describeTask(ctx context.Context, taskARN string) (awsTypes.Task, error) {
	var output *awsECS.DescribeTasksOutput
	err := retry(ctx, provisioner.config.MaxAttempts, func(callContext context.Context) error {
		var callErr error
		output, callErr = provisioner.client.DescribeTasks(callContext, &awsECS.DescribeTasksInput{
			Cluster: aws.String(provisioner.config.Cluster),
			Tasks:   []string{taskARN},
		})
		return callErr
	})
	if err != nil {
		return awsTypes.Task{}, mapTaskError(err)
	}
	if output == nil || len(output.Tasks) == 0 {
		if output != nil && len(output.Failures) > 0 && isMissingFailure(output.Failures[0]) {
			return awsTypes.Task{}, ErrTaskNotFound
		}
		return awsTypes.Task{}, ErrTaskNotFound
	}
	return output.Tasks[0], nil
}

func (provisioner *Provisioner) stopBestEffort(ctx context.Context, taskARN string) error {
	if taskARN == "" {
		return nil
	}
	return provisioner.Stop(ctx, taskARN)
}

func runtimeEnvironment(request sandbox.ProvisionRequest) ([]awsTypes.KeyValuePair, error) {
	values := make(map[string]string, len(request.Config.Environment)+2)
	for key, value := range request.Config.Environment {
		if isCredentialEnvironmentKey(key) || key == "HAEDES_RUNTIME_TOKEN" || key == "HAEDES_SANDBOX_ID" {
			return nil, fmt.Errorf("%w: %s", ErrRuntimeCredentialOverride, key)
		}
		values[key] = value
	}
	values["HAEDES_RUNTIME_TOKEN"] = request.Token
	values["HAEDES_SANDBOX_ID"] = string(request.SandboxID)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]awsTypes.KeyValuePair, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		environment = append(environment, awsTypes.KeyValuePair{Name: aws.String(key), Value: aws.String(value)})
	}
	return environment, nil
}

func isCredentialEnvironmentKey(key string) bool {
	key = strings.ToUpper(key)
	for _, forbidden := range []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_SECURITY_TOKEN",
		"AWS_PROFILE",
		"AWS_SHARED_CREDENTIALS_FILE",
		"AWS_CONFIG_FILE",
	} {
		if key == forbidden {
			return true
		}
	}
	return false
}

func privateIPv4Address(task awsTypes.Task) string {
	for _, attachment := range task.Attachments {
		if aws.ToString(attachment.Type) != "ElasticNetworkInterface" {
			continue
		}
		for _, detail := range attachment.Details {
			if aws.ToString(detail.Name) == "privateIPv4Address" {
				return aws.ToString(detail.Value)
			}
		}
	}
	return ""
}

func launchFailure(output *awsECS.RunTaskOutput) error {
	if output != nil && len(output.Failures) > 0 {
		failure := output.Failures[0]
		return fmt.Errorf("%w: %s %s", ErrTaskLaunchFailed, aws.ToString(failure.Reason), aws.ToString(failure.Detail))
	}
	return fmt.Errorf("%w: ECS returned no task", ErrTaskLaunchFailed)
}

func mapTaskError(err error) error {
	if isMissingTask(err) {
		return ErrTaskNotFound
	}
	return err
}

func isMissingTask(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "missing") || strings.Contains(message, "not found") || strings.Contains(message, "resource not found")
}

func isAlreadyStopped(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already stopped") || strings.Contains(message, "not in a stoppable state") || strings.Contains(message, "stopped")
}

func isMissingFailure(failure awsTypes.Failure) bool {
	reason := strings.ToLower(aws.ToString(failure.Reason))
	detail := strings.ToLower(aws.ToString(failure.Detail))
	return strings.Contains(reason, "missing") || strings.Contains(detail, "missing") || strings.Contains(reason, "not found")
}

func retry(ctx context.Context, maxAttempts int, operation func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		lastErr = operation(ctx)
		if lastErr == nil || !retryable(lastErr) || attempt == maxAttempts-1 {
			return lastErr
		}
		backoff := 50 * time.Millisecond * time.Duration(1<<attempt)
		if backoff > 500*time.Millisecond {
			backoff = 500 * time.Millisecond
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func retryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "ThrottlingException", "TooManyRequestsException", "RequestLimitExceeded", "ServiceUnavailableException", "ServerException", "InternalError":
			return true
		}
		return apiError.ErrorFault() == smithy.FaultServer
	}
	return false
}

func cloneConfig(config Config) Config {
	config.PrivateSubnetIDs = append([]string(nil), config.PrivateSubnetIDs...)
	config.SecurityGroupIDs = append([]string(nil), config.SecurityGroupIDs...)
	return config
}
