package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (clock testClock) Now() time.Time { return clock.now }

type testIDs struct{ next int }

func (ids *testIDs) NewSnapshotID() string {
	ids.next++
	return "snp_test"
}

type testRuntime struct {
	body        []byte
	info        ArchiveInfo
	restored    []byte
	restoreCall int
}

func (runtime *testRuntime) ExportWorkspace(context.Context, string) (io.ReadCloser, ArchiveInfo, error) {
	return io.NopCloser(bytes.NewReader(runtime.body)), runtime.info, nil
}

func (runtime *testRuntime) RestoreWorkspace(_ context.Context, _ string, body io.Reader) error {
	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	runtime.restored = content
	runtime.restoreCall++
	return nil
}

type testMetadata struct {
	items map[string]SnapshotMetadata
}

func newTestMetadata() *testMetadata { return &testMetadata{items: make(map[string]SnapshotMetadata)} }

func (metadata *testMetadata) Create(_ context.Context, snapshot SnapshotMetadata) error {
	if _, exists := metadata.items[snapshot.ID]; exists {
		return errors.New("duplicate snapshot")
	}
	metadata.items[snapshot.ID] = snapshot
	return nil
}

func (metadata *testMetadata) Get(_ context.Context, snapshotID string) (SnapshotMetadata, error) {
	snapshot, ok := metadata.items[snapshotID]
	if !ok {
		return SnapshotMetadata{}, ErrUnknownSnapshot
	}
	return snapshot, nil
}

func (metadata *testMetadata) UpdateState(_ context.Context, snapshotID, expected, next string) error {
	snapshot, ok := metadata.items[snapshotID]
	if !ok {
		return ErrUnknownSnapshot
	}
	if snapshot.State != expected {
		return errors.New("snapshot state conflict")
	}
	snapshot.State = next
	metadata.items[snapshotID] = snapshot
	return nil
}

func (metadata *testMetadata) ListExpired(_ context.Context, now time.Time, limit int) ([]SnapshotMetadata, error) {
	items := make([]SnapshotMetadata, 0, limit)
	for _, snapshot := range metadata.items {
		if len(items) == limit {
			break
		}
		if snapshot.State != StateDeleted && snapshot.ExpiresAt != nil && !snapshot.ExpiresAt.After(now) {
			items = append(items, snapshot)
		}
	}
	return items, nil
}

type storedObject struct {
	body    []byte
	info    ArchiveInfo
	options PutOptions
}

type testObjects struct {
	objects  map[string]storedObject
	deleted  []string
	putError error
}

func newTestObjects() *testObjects { return &testObjects{objects: make(map[string]storedObject)} }

func (objects *testObjects) Put(_ context.Context, key string, body io.Reader, options PutOptions) (ArchiveInfo, error) {
	if objects.putError != nil {
		return ArchiveInfo{}, objects.putError
	}
	content, err := io.ReadAll(body)
	if err != nil {
		return ArchiveInfo{}, err
	}
	info := infoFor(content)
	objects.objects[key] = storedObject{body: content, info: info, options: options}
	return info, nil
}

func (objects *testObjects) Open(_ context.Context, key string) (io.ReadCloser, ArchiveInfo, error) {
	object, ok := objects.objects[key]
	if !ok {
		return nil, ArchiveInfo{}, ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(object.body)), object.info, nil
}

func (objects *testObjects) Delete(_ context.Context, key string) error {
	objects.deleted = append(objects.deleted, key)
	if _, ok := objects.objects[key]; !ok {
		return ErrObjectNotFound
	}
	delete(objects.objects, key)
	return nil
}

func infoFor(body []byte) ArchiveInfo {
	digest := sha256.Sum256(body)
	return ArchiveInfo{ByteSize: int64(len(body)), SHA256: hex.EncodeToString(digest[:]), MediaType: MediaType}
}

func newTestService(body []byte) (*Service, *testRuntime, *testMetadata, *testObjects) {
	runtime := &testRuntime{body: body, info: infoFor(body)}
	metadata := newTestMetadata()
	objects := newTestObjects()
	service := NewService(Dependencies{
		Runtime:  runtime,
		Metadata: metadata,
		Objects:  objects,
		IDs:      &testIDs{},
		Clock:    testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	})
	return service, runtime, metadata, objects
}

func TestCreateVerifiesArchiveBeforePublishingMetadata(t *testing.T) {
	service, _, metadata, objects := newTestService([]byte("archive"))
	expires := service.clock.Now().Add(time.Hour)
	snapshot, err := service.Create(context.Background(), "sbx_test", &expires)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != StateAvailable || len(metadata.items) != 1 {
		t.Fatalf("snapshot = %+v, metadata = %+v", snapshot, metadata.items)
	}
	key := "sandboxes/sbx_test/snapshots/snp_test.tar.zst"
	object, ok := objects.objects[key]
	if !ok {
		t.Fatalf("object %q was not stored", key)
	}
	if object.options.ACL != ObjectACL || object.options.ServerSideEncryption != ServerSideEncryption {
		t.Fatalf("put options = %+v", object.options)
	}

	service, _, metadata, objects = newTestService([]byte("archive"))
	service.runtime.(*testRuntime).info.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = service.Create(context.Background(), "sbx_test", nil)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("checksum error = %v", err)
	}
	if len(metadata.items) != 0 || len(objects.deleted) != 1 {
		t.Fatalf("metadata = %+v, deleted = %v", metadata.items, objects.deleted)
	}
}

func TestRestoreReturnsTypedUnknownAndExpiredOutcomes(t *testing.T) {
	service, _, metadata, _ := newTestService([]byte("archive"))
	if err := service.Restore(context.Background(), "sbx_test", "snp_missing"); !errors.Is(err, ErrUnknownSnapshot) {
		t.Fatalf("unknown snapshot error = %v", err)
	}
	expired := service.clock.Now().Add(-time.Minute)
	metadata.items["snp_expired"] = SnapshotMetadata{
		ID: "snp_expired", SandboxID: "sbx_test", State: StateAvailable,
		ByteSize: 7, SHA256: infoFor([]byte("archive")).SHA256, ExpiresAt: &expired,
	}
	if err := service.Restore(context.Background(), "sbx_test", "snp_expired"); !errors.Is(err, ErrSnapshotExpired) {
		t.Fatalf("expired snapshot error = %v", err)
	}
}

func TestRestoreVerifiesStoredArchiveBeforeCallingRuntime(t *testing.T) {
	service, runtime, _, objects := newTestService([]byte("archive"))
	key := "sandboxes/sbx_test/snapshots/snp_test.tar.zst"
	objects.objects[key] = storedObject{body: []byte("archive"), info: infoFor([]byte("archive"))}
	service.metadata.(*testMetadata).items["snp_test"] = SnapshotMetadata{
		ID: "snp_test", SandboxID: "sbx_test", State: StateAvailable,
		ObjectKey: key, ByteSize: 7, SHA256: infoFor([]byte("archive")).SHA256,
	}
	if err := service.Restore(context.Background(), "sbx_test", "snp_test"); err != nil {
		t.Fatal(err)
	}
	if runtime.restoreCall != 1 || string(runtime.restored) != "archive" {
		t.Fatalf("restore calls = %d, body = %q", runtime.restoreCall, runtime.restored)
	}
}

func TestDeleteExpiredUsesCanonicalKeyAndIsIdempotent(t *testing.T) {
	service, _, metadata, objects := newTestService([]byte("archive"))
	expired := service.clock.Now().Add(-time.Minute)
	metadata.items["snp_test"] = SnapshotMetadata{
		ID: "snp_test", SandboxID: "sbx_test", State: StateAvailable,
		ObjectKey: "incorrect-key", ExpiresAt: &expired,
	}
	objects.objects["sandboxes/sbx_test/snapshots/snp_test.tar.zst"] = storedObject{}

	deleted, err := service.DeleteExpired(context.Background(), service.clock.Now(), 10)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted = %d, error = %v", deleted, err)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != "sandboxes/sbx_test/snapshots/snp_test.tar.zst" {
		t.Fatalf("deleted keys = %v", objects.deleted)
	}
	deleted, err = service.DeleteExpired(context.Background(), service.clock.Now(), 10)
	if err != nil || deleted != 0 {
		t.Fatalf("second deletion = %d, error = %v", deleted, err)
	}
}
