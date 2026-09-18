package s3

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	sdkS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	ErrInvalidStoreConfig  = errors.New("invalid s3 store configuration")
	ErrChecksumUnavailable = errors.New("s3 object checksum is unavailable")
)

type Client interface {
	GetObject(context.Context, *sdkS3.GetObjectInput, ...func(*sdkS3.Options)) (*sdkS3.GetObjectOutput, error)
	DeleteObject(context.Context, *sdkS3.DeleteObjectInput, ...func(*sdkS3.Options)) (*sdkS3.DeleteObjectOutput, error)
}

type Uploader interface {
	Upload(context.Context, *sdkS3.PutObjectInput, ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

type Config struct {
	Bucket string
	Retry  internalaws.RetryPolicy
}

type Store struct {
	client   Client
	uploader Uploader
	config   Config
}

var _ sandbox.SnapshotStore = (*Store)(nil)

func NewStore(client Client, uploader Uploader, config Config) (*Store, error) {
	if client == nil || uploader == nil || strings.TrimSpace(config.Bucket) == "" {
		return nil, ErrInvalidStoreConfig
	}
	return &Store{client: client, uploader: uploader, config: config}, nil
}

func (store *Store) Put(ctx context.Context, id sandbox.SnapshotID, archive io.Reader, expected sandbox.ArchiveInfo) (sandbox.ArchiveInfo, error) {
	if id == "" || archive == nil {
		return sandbox.ArchiveInfo{}, fmt.Errorf("%w: object ID and archive are required", ErrInvalidStoreConfig)
	}
	if expected.ByteSize <= 0 || normalizeChecksum(expected.SHA256) == "" {
		return sandbox.ArchiveInfo{}, fmt.Errorf("%w: byte size and SHA-256 are required", ErrInvalidStoreConfig)
	}
	key, err := objectKey(id)
	if err != nil {
		return sandbox.ArchiveInfo{}, err
	}

	tracked := &digestReader{reader: archive, hash: sha256.New()}
	input := &sdkS3.PutObjectInput{
		Bucket:               aws.String(store.config.Bucket),
		Key:                  aws.String(key),
		Body:                 tracked,
		ContentLength:        aws.Int64(expected.ByteSize),
		ContentType:          aws.String(mediaType(expected.MediaType)),
		ChecksumAlgorithm:    types.ChecksumAlgorithmSha256,
		ServerSideEncryption: types.ServerSideEncryptionAes256,
		Metadata: map[string]string{
			"snapshot-id": string(id),
			"byte-size":   fmt.Sprintf("%d", expected.ByteSize),
			"sha256":      normalizeChecksum(expected.SHA256),
		},
	}
	if _, err := store.uploader.Upload(ctx, input); err != nil {
		return sandbox.ArchiveInfo{}, err
	}
	actual := tracked.info(expected.MediaType)
	if actual.ByteSize != expected.ByteSize || actual.SHA256 != normalizeChecksum(expected.SHA256) {
		_ = store.Delete(ctx, id)
		return sandbox.ArchiveInfo{}, fmt.Errorf("%w: expected %d/%s, got %d/%s", sandbox.ErrSnapshotChecksumMismatch, expected.ByteSize, expected.SHA256, actual.ByteSize, actual.SHA256)
	}
	return actual, nil
}

func (store *Store) Open(ctx context.Context, id sandbox.SnapshotID) (io.ReadCloser, sandbox.ArchiveInfo, error) {
	key, err := objectKey(id)
	if err != nil {
		return nil, sandbox.ArchiveInfo{}, err
	}
	var output *sdkS3.GetObjectOutput
	err = internalaws.Retry(ctx, store.config.Retry, func(callContext context.Context) error {
		var callErr error
		output, callErr = store.client.GetObject(callContext, &sdkS3.GetObjectInput{
			Bucket: aws.String(store.config.Bucket),
			Key:    aws.String(key),
		})
		return callErr
	})
	if err != nil {
		return nil, sandbox.ArchiveInfo{}, mapObjectError(err)
	}
	if output == nil || output.Body == nil {
		return nil, sandbox.ArchiveInfo{}, fmt.Errorf("%w: empty object response", ErrChecksumUnavailable)
	}
	info, err := objectInfo(output)
	if err != nil {
		_ = output.Body.Close()
		return nil, sandbox.ArchiveInfo{}, err
	}
	return &verifiedReadCloser{ReadCloser: output.Body, hash: sha256.New(), expected: info}, info, nil
}

func (store *Store) Delete(ctx context.Context, id sandbox.SnapshotID) error {
	key, err := objectKey(id)
	if err != nil {
		return err
	}
	err = internalaws.Retry(ctx, store.config.Retry, func(callContext context.Context) error {
		_, callErr := store.client.DeleteObject(callContext, &sdkS3.DeleteObjectInput{
			Bucket: aws.String(store.config.Bucket),
			Key:    aws.String(key),
		})
		return callErr
	})
	if isMissingObject(err) {
		return nil
	}
	return mapObjectError(err)
}

type digestReader struct {
	reader io.Reader
	hash   hash.Hash
	size   int64
}

func (reader *digestReader) Read(buffer []byte) (int, error) {
	read, err := reader.reader.Read(buffer)
	if read > 0 {
		reader.size += int64(read)
		_, _ = reader.hash.Write(buffer[:read])
	}
	return read, err
}

func (reader *digestReader) info(contentType string) sandbox.ArchiveInfo {
	return sandbox.ArchiveInfo{
		ByteSize:  reader.size,
		SHA256:    hex.EncodeToString(reader.hash.Sum(nil)),
		MediaType: mediaType(contentType),
	}
}

type verifiedReadCloser struct {
	io.ReadCloser
	hash     hash.Hash
	size     int64
	expected sandbox.ArchiveInfo
	verified bool
}

func (reader *verifiedReadCloser) Read(buffer []byte) (int, error) {
	read, err := reader.ReadCloser.Read(buffer)
	if read > 0 {
		reader.size += int64(read)
		_, _ = reader.hash.Write(buffer[:read])
	}
	if err == io.EOF && !reader.verified {
		reader.verified = true
		actual := sandbox.ArchiveInfo{ByteSize: reader.size, SHA256: hex.EncodeToString(reader.hash.Sum(nil))}
		if actual.ByteSize != reader.expected.ByteSize || actual.SHA256 != normalizeChecksum(reader.expected.SHA256) {
			return read, fmt.Errorf("%w: expected %d/%s, got %d/%s", sandbox.ErrSnapshotChecksumMismatch, reader.expected.ByteSize, reader.expected.SHA256, actual.ByteSize, actual.SHA256)
		}
	}
	return read, err
}

func objectInfo(output *sdkS3.GetObjectOutput) (sandbox.ArchiveInfo, error) {
	checksum := strings.TrimSpace(output.Metadata["sha256"])
	if checksum == "" {
		checksum = strings.TrimSpace(output.Metadata["SHA256"])
	}
	if checksum == "" && output.ChecksumSHA256 != nil {
		decoded, err := base64.StdEncoding.DecodeString(aws.ToString(output.ChecksumSHA256))
		if err == nil {
			checksum = hex.EncodeToString(decoded)
		}
	}
	if normalizeChecksum(checksum) == "" {
		return sandbox.ArchiveInfo{}, ErrChecksumUnavailable
	}
	return sandbox.ArchiveInfo{
		ByteSize:  aws.ToInt64(output.ContentLength),
		SHA256:    normalizeChecksum(checksum),
		MediaType: aws.ToString(output.ContentType),
	}, nil
}

func objectKey(id sandbox.SnapshotID) (string, error) {
	value := strings.TrimSpace(string(id))
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "..") {
		return "", fmt.Errorf("%w: invalid object ID", ErrInvalidStoreConfig)
	}
	return "snapshots/" + value, nil
}

func normalizeChecksum(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

func mediaType(value string) string {
	if value == "" {
		return "application/vnd.haedes.workspace+tar.zstd"
	}
	return value
}

func mapObjectError(err error) error {
	if err == nil {
		return nil
	}
	if isMissingObject(err) {
		return sandbox.ErrNotFound
	}
	return err
}

func isMissingObject(err error) bool {
	if err == nil {
		return false
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		code := apiError.ErrorCode()
		return code == "NoSuchKey" || code == "NotFound"
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
