package fakes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	ErrRuntimeNotFound = fmt.Errorf("runtime task not found")
	ErrFileNotFound    = fmt.Errorf("file not found")
	ErrCommandNotFound = sandbox.ErrCommandNotFound
)

type RuntimeEvent struct {
	Kind      string
	SandboxID sandbox.SandboxID
	CommandID sandbox.CommandID
	Path      string
}

type Runtime struct {
	mu                sync.RWMutex
	clock             sandbox.Clock
	eventBus          *EventBus
	nextCommand       int
	ReadyError        error
	StartCommandError error
	Tasks             map[string]sandbox.SandboxID
	Commands          map[sandbox.CommandID]sandbox.CommandRequest
	RuntimeEvents     []RuntimeEvent
	files             map[string]map[string][]byte
	commandSandboxIDs map[sandbox.CommandID]sandbox.SandboxID
}

func NewRuntime(clock sandbox.Clock) *Runtime {
	if clock == nil {
		clock = realClock{}
	}
	return &Runtime{
		clock:             clock,
		eventBus:          NewEventBus(clock),
		Tasks:             make(map[string]sandbox.SandboxID),
		Commands:          make(map[sandbox.CommandID]sandbox.CommandRequest),
		files:             make(map[string]map[string][]byte),
		commandSandboxIDs: make(map[sandbox.CommandID]sandbox.SandboxID),
	}
}

func (runtime *Runtime) Register(sandboxID sandbox.SandboxID, task sandbox.TaskRef) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.Tasks[runtime.taskKey(task.Endpoint, task.RuntimeToken)] = sandboxID
	if runtime.files[string(sandboxID)] == nil {
		runtime.files[string(sandboxID)] = make(map[string][]byte)
	}
}

func (runtime *Runtime) WaitReady(ctx context.Context, endpoint, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	sandboxID, ok := runtime.Tasks[runtime.taskKey(endpoint, token)]
	if !ok {
		return ErrRuntimeNotFound
	}
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: "ready", SandboxID: sandboxID})
	if runtime.ReadyError != nil {
		return runtime.ReadyError
	}
	return nil
}

func (runtime *Runtime) StartCommand(ctx context.Context, endpoint, token string, request sandbox.CommandRequest) (sandbox.CommandResult, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.CommandResult{}, err
	}
	runtime.mu.Lock()
	sandboxID, ok := runtime.Tasks[runtime.taskKey(endpoint, token)]
	if !ok {
		runtime.mu.Unlock()
		return sandbox.CommandResult{}, ErrRuntimeNotFound
	}
	if runtime.StartCommandError != nil {
		err := runtime.StartCommandError
		runtime.mu.Unlock()
		return sandbox.CommandResult{}, err
	}
	runtime.nextCommand++
	commandID := sandbox.CommandID(fmt.Sprintf("cmd_%03d", runtime.nextCommand))
	request.Environment = cloneStringMap(request.Environment)
	runtime.Commands[commandID] = request
	runtime.commandSandboxIDs[commandID] = sandboxID
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: "command", SandboxID: sandboxID, CommandID: commandID})
	startedAt := runtime.clock.Now()
	zero := 0
	result := sandbox.CommandResult{
		ID:        commandID,
		SandboxID: sandboxID,
		Command:   request.Command,
		ExitCode:  &zero,
		StartedAt: startedAt,
	}
	finishedAt := runtime.clock.Now()
	result.FinishedAt = &finishedAt
	runtime.mu.Unlock()

	runtime.eventBus.Publish(commandID, "started", "", nil)
	runtime.eventBus.Publish(commandID, "completed", "", &result)
	return result, nil
}

func (runtime *Runtime) Events(ctx context.Context, endpoint, token string, commandID sandbox.CommandID, lastEventID string) (<-chan sandbox.CommandEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runtime.mu.RLock()
	_, taskExists := runtime.Tasks[runtime.taskKey(endpoint, token)]
	_, commandExists := runtime.Commands[commandID]
	runtime.mu.RUnlock()
	if !taskExists {
		return nil, ErrRuntimeNotFound
	}
	if !commandExists {
		return nil, ErrCommandNotFound
	}
	return runtime.eventBus.Events(ctx, commandID, lastEventID)
}

func (runtime *Runtime) ReadFile(ctx context.Context, endpoint, token, path string) (io.ReadCloser, error) {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return nil, err
	}
	runtime.mu.RLock()
	body, ok := runtime.files[string(sandboxID)][path]
	runtime.mu.RUnlock()
	if !ok {
		return nil, ErrFileNotFound
	}
	runtime.recordFileEvent("read", sandboxID, path)
	return io.NopCloser(bytes.NewReader(append([]byte(nil), body...))), nil
}

func (runtime *Runtime) WriteFile(ctx context.Context, endpoint, token, path string, body io.Reader) error {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return err
	}
	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	runtime.mu.Lock()
	runtime.files[string(sandboxID)][path] = append([]byte(nil), content...)
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: "write", SandboxID: sandboxID, Path: path})
	runtime.mu.Unlock()
	return nil
}

func (runtime *Runtime) ListFiles(ctx context.Context, endpoint, token, path string) ([]sandbox.FileEntry, error) {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSuffix(path, "/")
	runtime.mu.RLock()
	paths := make([]string, 0, len(runtime.files[string(sandboxID)]))
	for filePath := range runtime.files[string(sandboxID)] {
		if prefix == "" || filePath == prefix || strings.HasPrefix(filePath, prefix+"/") {
			paths = append(paths, filePath)
		}
	}
	runtime.mu.RUnlock()
	sort.Strings(paths)
	entries := make([]sandbox.FileEntry, 0, len(paths))
	for _, filePath := range paths {
		runtime.mu.RLock()
		size := int64(len(runtime.files[string(sandboxID)][filePath]))
		runtime.mu.RUnlock()
		entries = append(entries, sandbox.FileEntry{Path: filePath, Kind: "file", ByteSize: size, ModifiedAt: runtime.clock.Now()})
	}
	runtime.recordFileEvent("list", sandboxID, path)
	return entries, nil
}

func (runtime *Runtime) DeleteFile(ctx context.Context, endpoint, token, path string) error {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if _, ok := runtime.files[string(sandboxID)][path]; !ok {
		return ErrFileNotFound
	}
	delete(runtime.files[string(sandboxID)], path)
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: "delete", SandboxID: sandboxID, Path: path})
	return nil
}

func (runtime *Runtime) ExportWorkspace(ctx context.Context, endpoint, token string) (io.ReadCloser, sandbox.ArchiveInfo, error) {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return nil, sandbox.ArchiveInfo{}, err
	}
	runtime.mu.RLock()
	files := make(map[string][]byte, len(runtime.files[string(sandboxID)]))
	for path, body := range runtime.files[string(sandboxID)] {
		files[path] = append([]byte(nil), body...)
	}
	runtime.mu.RUnlock()
	archive, err := json.Marshal(files)
	if err != nil {
		return nil, sandbox.ArchiveInfo{}, err
	}
	info := archiveInfo(archive)
	runtime.recordFileEvent("export", sandboxID, "")
	return io.NopCloser(bytes.NewReader(archive)), info, nil
}

func (runtime *Runtime) RestoreWorkspace(ctx context.Context, endpoint, token string, archive io.Reader) error {
	sandboxID, err := runtime.task(ctx, endpoint, token)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(archive)
	if err != nil {
		return err
	}
	files := make(map[string][]byte)
	if err := json.Unmarshal(body, &files); err != nil {
		return err
	}
	runtime.mu.Lock()
	runtime.files[string(sandboxID)] = files
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: "restore", SandboxID: sandboxID})
	runtime.mu.Unlock()
	return nil
}

func (runtime *Runtime) task(ctx context.Context, endpoint, token string) (sandbox.SandboxID, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	runtime.mu.RLock()
	defer runtime.mu.RUnlock()
	sandboxID, ok := runtime.Tasks[runtime.taskKey(endpoint, token)]
	if !ok {
		return "", ErrRuntimeNotFound
	}
	return sandboxID, nil
}

func (runtime *Runtime) taskKey(endpoint, token string) string { return endpoint + "\x00" + token }

func (runtime *Runtime) recordFileEvent(kind string, sandboxID sandbox.SandboxID, path string) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.RuntimeEvents = append(runtime.RuntimeEvents, RuntimeEvent{Kind: kind, SandboxID: sandboxID, Path: path})
}

func archiveInfo(archive []byte) sandbox.ArchiveInfo {
	digest := sha256.Sum256(archive)
	return sandbox.ArchiveInfo{ByteSize: int64(len(archive)), SHA256: hex.EncodeToString(digest[:]), MediaType: "application/json"}
}
