package ecs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsECS "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestStartRetriesThrottlingDiscoversPrivateEndpointAndBuildsSafeOverrides(t *testing.T) {
	client := &fakeClient{}
	client.runTask = func(_ context.Context, input *awsECS.RunTaskInput) (*awsECS.RunTaskOutput, error) {
		if client.runCalls == 1 {
			return nil, &smithy.GenericAPIError{Code: "ThrottlingException", Message: "slow down", Fault: smithy.FaultServer}
		}
		if input.LaunchType != awsTypes.LaunchTypeFargate || input.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp != awsTypes.AssignPublicIpDisabled {
			t.Fatalf("task is not configured for private Fargate networking: %#v", input)
		}
		return &awsECS.RunTaskOutput{Tasks: []awsTypes.Task{{TaskArn: aws.String("arn:task/1")}}}, nil
	}
	client.describeTask = func(_ context.Context, _ *awsECS.DescribeTasksInput) (*awsECS.DescribeTasksOutput, error) {
		if client.describeCalls == 1 {
			return &awsECS.DescribeTasksOutput{Tasks: []awsTypes.Task{{LastStatus: aws.String("PENDING")}}}, nil
		}
		return &awsECS.DescribeTasksOutput{Tasks: []awsTypes.Task{{
			TaskArn:    aws.String("arn:task/1"),
			LastStatus: aws.String("RUNNING"),
			Attachments: []awsTypes.Attachment{{
				Type:    aws.String("ElasticNetworkInterface"),
				Details: []awsTypes.KeyValuePair{{Name: aws.String("privateIPv4Address"), Value: aws.String("10.0.2.17")}},
			}},
		}}}, nil
	}

	provisioner := newTestProvisioner(t, client)
	task, err := provisioner.Start(context.Background(), sandbox.ProvisionRequest{
		SandboxID: "sbx_001",
		Token:     "runtime-secret",
		Config: sandbox.SandboxConfig{Environment: map[string]string{
			"Z_MODE": "test",
			"A_MODE": "safe",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ARN != "arn:task/1" || task.PrivateIP != "10.0.2.17" || task.Endpoint != "http://10.0.2.17:8080" || task.RuntimeToken != "runtime-secret" {
		t.Fatalf("task reference = %+v", task)
	}
	if client.runCalls != 2 || client.describeCalls != 2 {
		t.Fatalf("calls = run %d, describe %d", client.runCalls, client.describeCalls)
	}
	if got := client.lastRunTask.Overrides.ContainerOverrides[0].Environment; len(got) != 4 || aws.ToString(got[0].Name) != "A_MODE" || aws.ToString(got[1].Name) != "HAEDES_RUNTIME_TOKEN" || aws.ToString(got[2].Name) != "HAEDES_SANDBOX_ID" || aws.ToString(got[3].Name) != "Z_MODE" {
		t.Fatalf("environment overrides are not deterministic or complete: %+v", got)
	}
}

func TestStartRejectsCredentialOverrides(t *testing.T) {
	provisioner := newTestProvisioner(t, &fakeClient{})
	_, err := provisioner.Start(context.Background(), sandbox.ProvisionRequest{
		SandboxID: "sbx_001",
		Token:     "runtime-secret",
		Config:    sandbox.SandboxConfig{Environment: map[string]string{"AWS_ACCESS_KEY_ID": "do-not-pass"}},
	})
	if !errors.Is(err, ErrRuntimeCredentialOverride) {
		t.Fatalf("error = %v", err)
	}
}

func TestStartMapsMissingTaskAndStopsBestEffort(t *testing.T) {
	client := &fakeClient{
		runTask: func(context.Context, *awsECS.RunTaskInput) (*awsECS.RunTaskOutput, error) {
			return &awsECS.RunTaskOutput{Tasks: []awsTypes.Task{{TaskArn: aws.String("arn:task/missing")}}}, nil
		},
		describeTask: func(context.Context, *awsECS.DescribeTasksInput) (*awsECS.DescribeTasksOutput, error) {
			return &awsECS.DescribeTasksOutput{Failures: []awsTypes.Failure{{Reason: aws.String("MISSING")}}}, nil
		},
	}
	provisioner := newTestProvisioner(t, client)
	_, err := provisioner.Start(context.Background(), sandbox.ProvisionRequest{SandboxID: "sbx_001", Token: "token"})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("error = %v", err)
	}
	if client.stopCalls != 1 {
		t.Fatalf("best-effort stop calls = %d", client.stopCalls)
	}
}

func TestStopTreatsAlreadyStoppedTaskAsSuccess(t *testing.T) {
	client := &fakeClient{
		stopTask: func(context.Context, *awsECS.StopTaskInput) (*awsECS.StopTaskOutput, error) {
			return nil, errors.New("task is not in a stoppable state because it is already stopped")
		},
	}
	provisioner := newTestProvisioner(t, client)
	if err := provisioner.Stop(context.Background(), "arn:task/stopped"); err != nil {
		t.Fatal(err)
	}
}

func TestDescribeMapsMissingTask(t *testing.T) {
	client := &fakeClient{
		describeTask: func(context.Context, *awsECS.DescribeTasksInput) (*awsECS.DescribeTasksOutput, error) {
			return &awsECS.DescribeTasksOutput{Failures: []awsTypes.Failure{{Reason: aws.String("MISSING")}}}, nil
		},
	}
	provisioner := newTestProvisioner(t, client)
	_, err := provisioner.Describe(context.Background(), "arn:task/missing")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func newTestProvisioner(t *testing.T, client *fakeClient) *Provisioner {
	t.Helper()
	provisioner, err := NewProvisioner(client, Config{
		Cluster:          "haedes-dev",
		TaskDefinition:   "haedes-sandbox:1",
		ContainerName:    "runtime",
		PrivateSubnetIDs: []string{"subnet-a", "subnet-b"},
		SecurityGroupIDs: []string{"sg-runtime"},
		StartTimeout:     2 * time.Second,
		DescribeTimeout:  500 * time.Millisecond,
		StopTimeout:      500 * time.Millisecond,
		PollInterval:     time.Millisecond,
		MaxAttempts:      3,
	})
	if err != nil {
		t.Fatal(err)
	}
	return provisioner
}

type fakeClient struct {
	runTask       func(context.Context, *awsECS.RunTaskInput) (*awsECS.RunTaskOutput, error)
	describeTask  func(context.Context, *awsECS.DescribeTasksInput) (*awsECS.DescribeTasksOutput, error)
	stopTask      func(context.Context, *awsECS.StopTaskInput) (*awsECS.StopTaskOutput, error)
	runCalls      int
	describeCalls int
	stopCalls     int
	lastRunTask   *awsECS.RunTaskInput
}

func (client *fakeClient) RunTask(ctx context.Context, input *awsECS.RunTaskInput, _ ...func(*awsECS.Options)) (*awsECS.RunTaskOutput, error) {
	client.runCalls++
	client.lastRunTask = input
	if client.runTask == nil {
		return &awsECS.RunTaskOutput{}, nil
	}
	return client.runTask(ctx, input)
}

func (client *fakeClient) DescribeTasks(ctx context.Context, input *awsECS.DescribeTasksInput, _ ...func(*awsECS.Options)) (*awsECS.DescribeTasksOutput, error) {
	client.describeCalls++
	if client.describeTask == nil {
		return &awsECS.DescribeTasksOutput{}, nil
	}
	return client.describeTask(ctx, input)
}

func (client *fakeClient) StopTask(ctx context.Context, input *awsECS.StopTaskInput, _ ...func(*awsECS.Options)) (*awsECS.StopTaskOutput, error) {
	client.stopCalls++
	if client.stopTask == nil {
		return &awsECS.StopTaskOutput{}, nil
	}
	return client.stopTask(ctx, input)
}
