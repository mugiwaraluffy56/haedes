package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/snapshot-service/internal/archive"
)

const (
	StateAvailable       = "available"
	StateDeleted         = "deleted"
	MediaType            = "application/vnd.haedes.workspace+tar.zstd"
	ObjectACL            = "private"
	ServerSideEncryption = "AES256"
)

var (
	ErrUnknownSnapshot     = errors.New("snapshot is unknown")
	ErrChecksumMismatch    = errors.New("snapshot checksum mismatch")
	ErrSnapshotExpired     = errors.New("snapshot metadata has expired")
	ErrSnapshotUnavailable = errors.New("snapshot is not available")
	ErrInvalidSnapshotID   = errors.New("snapshot identifier is invalid")
	ErrRetention           = errors.New("snapshot retention failed")
	ErrObjectNotFound      = errors.New("snapshot object is not found")
)

type ErrorCode string

const (
	CodeUnknownSnapshot     ErrorCode = "snapshot_not_found"
	CodeChecksumMismatch    ErrorCode = "snapshot_checksum_mismatch"
	CodeSnapshotExpired     ErrorCode = "snapshot_expired"
	CodeSnapshotUnavailable ErrorCode = "snapshot_unavailable"
	CodeInvalidSnapshotID   ErrorCode = "snapshot_invalid_id"
	CodeRetentionFailed     ErrorCode = "snapshot_retention_failed"
)

type Error struct {
	Code       ErrorCode
	SnapshotID string
	Err        error
}

func (e *Error) Error() string {
	if e.SnapshotID == "" {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return fmt.Sprintf("%s (%s): %v", e.Code, e.SnapshotID, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type SnapshotMetadata struct {
	ID        string
	SandboxID string
	State     string
	ObjectKey string
	ByteSize  int64
	SHA256    string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type ArchiveInfo struct {
	ByteSize  int64
	SHA256    string
	MediaType string
}

type PutOptions struct {
	ContentType          string
	ACL                  string
	ServerSideEncryption string
}

type Runtime interface {
	ExportWorkspace(ctx context.Context, sandboxID string) (io.ReadCloser, ArchiveInfo, error)
	RestoreWorkspace(ctx context.Context, sandboxID string, archive io.Reader) error
}

type MetadataRepository interface {
	Create(ctx context.Context, snapshot SnapshotMetadata) error
	Get(ctx context.Context, snapshotID string) (SnapshotMetadata, error)
	UpdateState(ctx context.Context, snapshotID, expected, next string) error
	ListExpired(ctx context.Context, now time.Time, limit int) ([]SnapshotMetadata, error)
}

type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, options PutOptions) (ArchiveInfo, error)
	Open(ctx context.Context, key string) (io.ReadCloser, ArchiveInfo, error)
	Delete(ctx context.Context, key string) error
}

type IDGenerator interface {
	NewSnapshotID() string
}

type Clock interface {
	Now() time.Time
}

type Dependencies struct {
	Runtime  Runtime
	Metadata MetadataRepository
	Objects  ObjectStore
	IDs      IDGenerator
	Clock    Clock
}

type Service struct {
	runtime  Runtime
	metadata MetadataRepository
	objects  ObjectStore
	ids      IDGenerator
	clock    Clock
}

func NewService(dependencies Dependencies) *Service {
	return &Service{
		runtime:  dependencies.Runtime,
		metadata: dependencies.Metadata,
		objects:  dependencies.Objects,
		ids:      dependencies.IDs,
		clock:    dependencies.Clock,
	}
}

func (service *Service) Create(ctx context.Context, sandboxID string, expiresAt *time.Time) (SnapshotMetadata, error) {
	if err := service.validateDependencies(); err != nil {
		return SnapshotMetadata{}, err
	}
	snapshotID := service.ids.NewSnapshotID()
	key, err := ArchiveObjectKey(sandboxID, snapshotID)
	if err != nil {
		return SnapshotMetadata{}, err
	}

	content, expected, err := service.runtime.ExportWorkspace(ctx, sandboxID)
	if err != nil {
		return SnapshotMetadata{}, err
	}
	defer content.Close()

	var observed bytesDigest
	actual, err := service.objects.Put(ctx, key, observed.reader(content), PutOptions{
		ContentType:          mediaType(expected.MediaType),
		ACL:                  ObjectACL,
		ServerSideEncryption: ServerSideEncryption,
	})
	if err != nil {
		return SnapshotMetadata{}, err
	}
	if err := verifyArchive(expected, observed.info(), actual); err != nil {
		_ = service.objects.Delete(ctx, key)
		return SnapshotMetadata{}, &Error{Code: CodeChecksumMismatch, SnapshotID: snapshotID, Err: err}
	}

	now := service.clock.Now()
	snapshot := SnapshotMetadata{
		ID:        snapshotID,
		SandboxID: sandboxID,
		State:     StateAvailable,
		ObjectKey: key,
		ByteSize:  observed.size,
		SHA256:    observed.sha256,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := service.metadata.Create(ctx, snapshot); err != nil {
		_ = service.objects.Delete(ctx, key)
		return SnapshotMetadata{}, err
	}
	return snapshot, nil
}

func (service *Service) Restore(ctx context.Context, sandboxID, snapshotID string) error {
	if err := service.validateDependencies(); err != nil {
		return err
	}
	snapshot, err := service.metadata.Get(ctx, snapshotID)
	if err != nil || snapshot.ID == "" || snapshot.SandboxID != sandboxID {
		return &Error{Code: CodeUnknownSnapshot, SnapshotID: snapshotID, Err: ErrUnknownSnapshot}
	}
	if snapshot.State != StateAvailable {
		return &Error{Code: CodeSnapshotUnavailable, SnapshotID: snapshotID, Err: ErrSnapshotUnavailable}
	}
	if snapshot.ExpiresAt != nil && !service.clock.Now().Before(*snapshot.ExpiresAt) {
		return &Error{Code: CodeSnapshotExpired, SnapshotID: snapshotID, Err: ErrSnapshotExpired}
	}

	key, err := ArchiveObjectKey(snapshot.SandboxID, snapshot.ID)
	if err != nil {
		return err
	}
	content, stored, err := service.objects.Open(ctx, key)
	if err != nil {
		return &Error{Code: CodeUnknownSnapshot, SnapshotID: snapshotID, Err: ErrUnknownSnapshot}
	}
	defer content.Close()

	temporary, err := os.CreateTemp("", "haedes-snapshot-*")
	if err != nil {
		return fmt.Errorf("stage snapshot archive: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	size, checksum, err := archive.Digest(io.TeeReader(content, temporary))
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := verifyArchive(ArchiveInfo{ByteSize: snapshot.ByteSize, SHA256: snapshot.SHA256}, ArchiveInfo{ByteSize: size, SHA256: checksum}, stored); err != nil {
		return &Error{Code: CodeChecksumMismatch, SnapshotID: snapshotID, Err: err}
	}

	input, err := os.Open(temporaryName)
	if err != nil {
		return fmt.Errorf("open staged snapshot archive: %w", err)
	}
	defer input.Close()
	return service.runtime.RestoreWorkspace(ctx, sandboxID, input)
}

func (service *Service) DeleteExpired(ctx context.Context, now time.Time, batchSize int) (int, error) {
	if err := service.validateDependencies(); err != nil {
		return 0, err
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	snapshots, err := service.metadata.ListExpired(ctx, now, batchSize)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, snapshot := range snapshots {
		if snapshot.State == StateDeleted || snapshot.ExpiresAt == nil || snapshot.ExpiresAt.After(now) {
			continue
		}
		key, err := ArchiveObjectKey(snapshot.SandboxID, snapshot.ID)
		if err != nil {
			return deleted, err
		}
		if err := service.objects.Delete(ctx, key); err != nil && !errors.Is(err, ErrObjectNotFound) {
			return deleted, &Error{Code: CodeRetentionFailed, SnapshotID: snapshot.ID, Err: fmt.Errorf("%w: %v", ErrRetention, err)}
		}
		if err := service.metadata.UpdateState(ctx, snapshot.ID, snapshot.State, StateDeleted); err != nil {
			return deleted, &Error{Code: CodeRetentionFailed, SnapshotID: snapshot.ID, Err: fmt.Errorf("%w: %v", ErrRetention, err)}
		}
		deleted++
	}
	return deleted, nil
}

func (service *Service) validateDependencies() error {
	if service.runtime == nil || service.metadata == nil || service.objects == nil || service.ids == nil || service.clock == nil {
		return errors.New("snapshot service dependencies are incomplete")
	}
	return nil
}

func ArchiveObjectKey(sandboxID, snapshotID string) (string, error) {
	if !validIDSegment(sandboxID) || !validIDSegment(snapshotID) {
		return "", &Error{Code: CodeInvalidSnapshotID, Err: ErrInvalidSnapshotID}
	}
	return fmt.Sprintf("sandboxes/%s/snapshots/%s.tar.zst", sandboxID, snapshotID), nil
}

func validIDSegment(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func mediaType(value string) string {
	if value == "" {
		return MediaType
	}
	return value
}

func verifyArchive(expected, observed, stored ArchiveInfo) error {
	if expected.ByteSize != observed.ByteSize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrChecksumMismatch, expected.ByteSize, observed.ByteSize)
	}
	if normalizeChecksum(expected.SHA256) == "" || normalizeChecksum(expected.SHA256) != normalizeChecksum(observed.SHA256) {
		return fmt.Errorf("%w: expected %q, got %q", ErrChecksumMismatch, expected.SHA256, observed.SHA256)
	}
	if stored.ByteSize != 0 && stored.ByteSize != observed.ByteSize {
		return fmt.Errorf("%w: object store reported %d bytes, got %d", ErrChecksumMismatch, stored.ByteSize, observed.ByteSize)
	}
	if stored.SHA256 != "" && normalizeChecksum(stored.SHA256) != normalizeChecksum(observed.SHA256) {
		return fmt.Errorf("%w: object store checksum %q, got %q", ErrChecksumMismatch, stored.SHA256, observed.SHA256)
	}
	return nil
}

func normalizeChecksum(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

type bytesDigest struct {
	size   int64
	sha256 string
}

func (digest *bytesDigest) reader(content io.Reader) io.Reader {
	return &digestReader{reader: content, digest: digest}
}

func (digest *bytesDigest) info() ArchiveInfo {
	if digest.sha256 == "" {
		sum := sha256.Sum256(nil)
		digest.sha256 = hex.EncodeToString(sum[:])
	}
	return ArchiveInfo{ByteSize: digest.size, SHA256: digest.sha256}
}

type digestReader struct {
	reader io.Reader
	digest *bytesDigest
	hash   hash.Hash
}

func (reader *digestReader) Read(buffer []byte) (int, error) {
	if reader.hash == nil {
		reader.hash = sha256.New()
	}
	read, err := reader.reader.Read(buffer)
	if read > 0 {
		reader.digest.size += int64(read)
		_, _ = reader.hash.Write(buffer[:read])
		reader.digest.sha256 = hex.EncodeToString(reader.hash.Sum(nil))
	}
	return read, err
}
