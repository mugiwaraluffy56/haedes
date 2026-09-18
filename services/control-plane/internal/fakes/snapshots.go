package fakes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var ErrObjectNotFound = fmt.Errorf("snapshot object not found")

type SnapshotRepository struct {
	mu        sync.RWMutex
	snapshots map[sandbox.SnapshotID]sandbox.SnapshotMetadata
}

func NewSnapshotRepository() *SnapshotRepository {
	return &SnapshotRepository{snapshots: make(map[sandbox.SnapshotID]sandbox.SnapshotMetadata)}
}

func (repository *SnapshotRepository) Create(ctx context.Context, snapshot sandbox.SnapshotMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.snapshots[snapshot.ID]; exists {
		return fmt.Errorf("snapshot %s already exists", snapshot.ID)
	}
	repository.snapshots[snapshot.ID] = snapshot
	return nil
}

func (repository *SnapshotRepository) Get(ctx context.Context, id sandbox.SnapshotID) (sandbox.SnapshotMetadata, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.SnapshotMetadata{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	snapshot, ok := repository.snapshots[id]
	if !ok {
		return sandbox.SnapshotMetadata{}, sandbox.ErrNotFound
	}
	return snapshot, nil
}

func (repository *SnapshotRepository) List(ctx context.Context, sandboxID sandbox.SandboxID, cursor string, limit int) (sandbox.Page[sandbox.SnapshotMetadata], error) {
	if err := ctx.Err(); err != nil {
		return sandbox.Page[sandbox.SnapshotMetadata]{}, err
	}
	if limit <= 0 {
		return sandbox.Page[sandbox.SnapshotMetadata]{}, fmt.Errorf("limit must be positive")
	}
	start := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			return sandbox.Page[sandbox.SnapshotMetadata]{}, fmt.Errorf("invalid cursor")
		}
		start = parsed
	}
	repository.mu.RLock()
	items := make([]sandbox.SnapshotMetadata, 0, len(repository.snapshots))
	for _, snapshot := range repository.snapshots {
		if snapshot.SandboxID == sandboxID {
			items = append(items, snapshot)
		}
	}
	repository.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if start >= len(items) {
		return sandbox.Page[sandbox.SnapshotMetadata]{}, nil
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	page := sandbox.Page[sandbox.SnapshotMetadata]{Items: items[start:end]}
	if end < len(items) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (repository *SnapshotRepository) Update(ctx context.Context, snapshot sandbox.SnapshotMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.snapshots[snapshot.ID]; !exists {
		return sandbox.ErrNotFound
	}
	repository.snapshots[snapshot.ID] = snapshot
	return nil
}

func (repository *SnapshotRepository) UpdateState(ctx context.Context, id sandbox.SnapshotID, expected, next string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	snapshot, ok := repository.snapshots[id]
	if !ok {
		return sandbox.ErrNotFound
	}
	if snapshot.State != expected {
		return fmt.Errorf("snapshot state conflict: expected %s, got %s", expected, snapshot.State)
	}
	snapshot.State = next
	repository.snapshots[id] = snapshot
	return nil
}

func (repository *SnapshotRepository) Save(snapshot sandbox.SnapshotMetadata) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.snapshots[snapshot.ID] = snapshot
}

type StoreEvent struct {
	Kind       string
	SnapshotID sandbox.SnapshotID
}

type ObjectStore struct {
	mu      sync.RWMutex
	objects map[sandbox.SnapshotID]storedObject
	Events  []StoreEvent
}

type storedObject struct {
	body []byte
	info sandbox.ArchiveInfo
}

func NewObjectStore() *ObjectStore {
	return &ObjectStore{objects: make(map[sandbox.SnapshotID]storedObject)}
}

func (store *ObjectStore) Put(ctx context.Context, id sandbox.SnapshotID, archive io.Reader, info sandbox.ArchiveInfo) (sandbox.ArchiveInfo, error) {
	if err := ctx.Err(); err != nil {
		return sandbox.ArchiveInfo{}, err
	}
	body, err := io.ReadAll(archive)
	if err != nil {
		return sandbox.ArchiveInfo{}, err
	}
	actual := archiveInfo(body)
	if info.ByteSize != 0 && info.ByteSize != actual.ByteSize {
		return sandbox.ArchiveInfo{}, fmt.Errorf("%w: archive byte size mismatch", sandbox.ErrSnapshotChecksumMismatch)
	}
	if info.SHA256 != "" && info.SHA256 != actual.SHA256 {
		return sandbox.ArchiveInfo{}, fmt.Errorf("%w: archive checksum mismatch", sandbox.ErrSnapshotChecksumMismatch)
	}
	if info.MediaType != "" {
		actual.MediaType = info.MediaType
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.objects[id] = storedObject{body: append([]byte(nil), body...), info: actual}
	store.Events = append(store.Events, StoreEvent{Kind: "put", SnapshotID: id})
	return actual, nil
}

func (store *ObjectStore) Open(ctx context.Context, id sandbox.SnapshotID) (io.ReadCloser, sandbox.ArchiveInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, sandbox.ArchiveInfo{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	object, ok := store.objects[id]
	if !ok {
		return nil, sandbox.ArchiveInfo{}, ErrObjectNotFound
	}
	store.Events = append(store.Events, StoreEvent{Kind: "open", SnapshotID: id})
	return io.NopCloser(bytes.NewReader(append([]byte(nil), object.body...))), object.info, nil
}
