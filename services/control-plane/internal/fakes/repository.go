package fakes

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

type Repository struct {
	mu        sync.RWMutex
	sandboxes map[sandbox.SandboxID]sandbox.Sandbox
	Updates   []StateUpdate
}

type StateUpdate struct {
	ID       sandbox.SandboxID
	Expected sandbox.State
	Next     sandbox.State
	Reason   string
}

func NewRepository() *Repository {
	return &Repository{sandboxes: make(map[sandbox.SandboxID]sandbox.Sandbox)}
}

func (repository *Repository) Create(ctx context.Context, value sandbox.Sandbox) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.sandboxes[value.ID]; exists {
		return fmt.Errorf("sandbox %s already exists", value.ID)
	}
	repository.sandboxes[value.ID] = cloneSandbox(value)
	return nil
}

func (repository *Repository) Get(ctx context.Context, id sandbox.SandboxID) (sandbox.Sandbox, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.Sandbox{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.sandboxes[id]
	if !ok {
		return sandbox.Sandbox{}, sandbox.ErrNotFound
	}
	return cloneSandbox(value), nil
}

func (repository *Repository) List(ctx context.Context, ownerID, cursor string, limit int) (sandbox.Page[sandbox.Sandbox], error) {
	if err := ctx.Err(); err != nil {
		return sandbox.Page[sandbox.Sandbox]{}, err
	}
	if limit <= 0 {
		return sandbox.Page[sandbox.Sandbox]{}, fmt.Errorf("limit must be positive")
	}
	start, err := parseCursor(cursor)
	if err != nil {
		return sandbox.Page[sandbox.Sandbox]{}, err
	}
	repository.mu.RLock()
	values := make([]sandbox.Sandbox, 0, len(repository.sandboxes))
	for _, value := range repository.sandboxes {
		if value.OwnerID == ownerID {
			values = append(values, cloneSandbox(value))
		}
	}
	repository.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	if start >= len(values) {
		return sandbox.Page[sandbox.Sandbox]{}, nil
	}
	end := start + limit
	if end > len(values) {
		end = len(values)
	}
	page := sandbox.Page[sandbox.Sandbox]{Items: values[start:end]}
	if end < len(values) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (repository *Repository) UpdateState(ctx context.Context, id sandbox.SandboxID, expected, next sandbox.State, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	value, ok := repository.sandboxes[id]
	if !ok {
		return sandbox.ErrNotFound
	}
	if value.State != expected {
		return fmt.Errorf("state conflict: expected %s, got %s", expected, value.State)
	}
	if err := sandbox.Transition(&value, next, reason); err != nil {
		return err
	}
	repository.sandboxes[id] = cloneSandbox(value)
	repository.Updates = append(repository.Updates, StateUpdate{ID: id, Expected: expected, Next: next, Reason: reason})
	return nil
}

func (repository *Repository) SaveTaskEndpoint(ctx context.Context, id sandbox.SandboxID, task sandbox.TaskRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	value, ok := repository.sandboxes[id]
	if !ok {
		return sandbox.ErrNotFound
	}
	taskCopy := task
	value.Task = &taskCopy
	repository.sandboxes[id] = cloneSandbox(value)
	return nil
}

func parseCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(cursor)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return value, nil
}

func cloneSandbox(value sandbox.Sandbox) sandbox.Sandbox {
	clone := value
	if value.Task != nil {
		task := *value.Task
		clone.Task = &task
	}
	if value.Repository != nil {
		repository := *value.Repository
		clone.Repository = &repository
	}
	if value.Config.Repository != nil {
		repository := *value.Config.Repository
		clone.Config.Repository = &repository
	}
	if value.Config.SnapshotID != nil {
		snapshotID := *value.Config.SnapshotID
		clone.Config.SnapshotID = &snapshotID
	}
	if value.CurrentCommand != nil {
		command := *value.CurrentCommand
		clone.CurrentCommand = &command
	}
	clone.SnapshotIDs = append([]sandbox.SnapshotID(nil), value.SnapshotIDs...)
	clone.Config.Environment = cloneStringMap(value.Config.Environment)
	return clone
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}
