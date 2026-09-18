package fakes

import (
	"context"
	"fmt"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	ErrTaskNotFound = fmt.Errorf("task not found")
	ErrTaskStopped  = fmt.Errorf("task is stopped")
)

type ComputeEvent struct {
	Kind      string
	SandboxID sandbox.SandboxID
	Task      sandbox.TaskRef
}

type Compute struct {
	mu         sync.RWMutex
	clock      sandbox.Clock
	next       int
	StartTask  sandbox.TaskRef
	StartError error
	StopError  error
	Events     []ComputeEvent
	statuses   map[string]sandbox.TaskStatus
}

func NewCompute(clock sandbox.Clock) *Compute {
	if clock == nil {
		clock = realClock{}
	}
	return &Compute{clock: clock, statuses: make(map[string]sandbox.TaskStatus)}
}

func (compute *Compute) Start(ctx context.Context, request sandbox.ProvisionRequest) (sandbox.TaskRef, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.TaskRef{}, err
	}
	compute.mu.Lock()
	defer compute.mu.Unlock()
	compute.next++
	task := compute.StartTask
	if task.ARN == "" {
		task = sandbox.TaskRef{
			ARN:          fmt.Sprintf("task_%03d", compute.next),
			PrivateIP:    fmt.Sprintf("10.0.0.%d", compute.next),
			Endpoint:     fmt.Sprintf("http://runtime/task_%03d", compute.next),
			RuntimeToken: request.Token,
		}
	}
	if task.RuntimeToken == "" {
		task.RuntimeToken = request.Token
	}
	compute.Events = append(compute.Events, ComputeEvent{Kind: "start", SandboxID: request.SandboxID, Task: task})
	if compute.StartError != nil {
		return sandbox.TaskRef{}, compute.StartError
	}
	compute.statuses[task.ARN] = sandbox.TaskStatus{ARN: task.ARN, Phase: "running", PrivateIP: task.PrivateIP}
	return task, nil
}

func (compute *Compute) Stop(ctx context.Context, taskARN string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	compute.mu.Lock()
	defer compute.mu.Unlock()
	status, ok := compute.statuses[taskARN]
	if !ok {
		return ErrTaskNotFound
	}
	if status.Phase == "stopped" {
		return nil
	}
	compute.Events = append(compute.Events, ComputeEvent{Kind: "stop", Task: sandbox.TaskRef{ARN: taskARN}})
	if compute.StopError != nil {
		return compute.StopError
	}
	status.Phase = "stopped"
	compute.statuses[taskARN] = status
	return nil
}

func (compute *Compute) Describe(ctx context.Context, taskARN string) (sandbox.TaskStatus, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.TaskStatus{}, err
	}
	compute.mu.RLock()
	defer compute.mu.RUnlock()
	status, ok := compute.statuses[taskARN]
	if !ok {
		return sandbox.TaskStatus{}, ErrTaskNotFound
	}
	return status, nil
}
