package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	sdkS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestPutStreamsPrivateEncryptedObjectAndVerifiesChecksum(t *testing.T) {
	content := []byte("workspace archive")
	expected := archiveInfo(content)
	uploader := &fakeUploader{}
	store, err := NewStore(&fakeS3Client{}, uploader, Config{Bucket: "private-snapshots", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := store.Put(context.Background(), "snp_001", bytes.NewReader(content), expected)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected || uploader.input == nil || uploader.input.ServerSideEncryption != "AES256" || uploader.input.ChecksumAlgorithm != "SHA256" {
		t.Fatalf("actual = %+v, input = %+v", actual, uploader.input)
	}
	if uploader.input.Metadata["snapshot-id"] != "snp_001" || uploader.input.Metadata["sha256"] != expected.SHA256 {
		t.Fatalf("metadata = %+v", uploader.input.Metadata)
	}
	if uploader.input.ACL != "" {
		t.Fatalf("unexpected public ACL = %q", uploader.input.ACL)
	}
}

func TestPutDeletesObjectAfterChecksumMismatch(t *testing.T) {
	content := []byte("workspace archive")
	expected := archiveInfo(content)
	expected.SHA256 = strings.Repeat("0", 64)
	client := &fakeS3Client{}
	store, err := NewStore(client, &fakeUploader{}, Config{Bucket: "private-snapshots", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(context.Background(), "snp_002", bytes.NewReader(content), expected)
	if !errors.Is(err, sandbox.ErrSnapshotChecksumMismatch) || client.deleteCalls != 1 {
		t.Fatalf("error = %v, deletes = %d", err, client.deleteCalls)
	}
}

func TestOpenVerifiesStoredChecksumAndMapsMissingObjects(t *testing.T) {
	content := []byte("workspace archive")
	client := &fakeS3Client{getOutput: &sdkS3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(content)),
		ContentLength: aws.Int64(int64(len(content))),
		ContentType:   aws.String("application/vnd.haedes.workspace+tar.zstd"),
		Metadata:      map[string]string{"sha256": archiveInfo(content).SHA256},
	}}
	store, err := NewStore(client, &fakeUploader{}, Config{Bucket: "private-snapshots", Retry: noWaitRetry()})
	if err != nil {
		t.Fatal(err)
	}
	body, info, err := store.Open(context.Background(), "snp_003")
	if err != nil {
		t.Fatal(err)
	}
	read, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || string(read) != string(content) || info.SHA256 != archiveInfo(content).SHA256 {
		t.Fatalf("read = %q, info = %+v, err = %v", read, info, err)
	}

	client.getErr = &smithy.GenericAPIError{Code: "NoSuchKey"}
	if _, _, err := store.Open(context.Background(), "missing"); !errors.Is(err, sandbox.ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}

	client.getErr = nil
	client.getOutput = &sdkS3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader([]byte("corrupted archive"))),
		ContentLength: aws.Int64(int64(len("corrupted archive"))),
		Metadata:      map[string]string{"sha256": archiveInfo(content).SHA256},
	}
	body, _, err = store.Open(context.Background(), "corrupt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(body)
	_ = body.Close()
	if !errors.Is(err, sandbox.ErrSnapshotChecksumMismatch) {
		t.Fatalf("corrupt object error = %v", err)
	}
}

type fakeUploader struct {
	input *sdkS3.PutObjectInput
}

func (uploader *fakeUploader) Upload(_ context.Context, input *sdkS3.PutObjectInput, _ ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	uploader.input = input
	_, err := io.Copy(io.Discard, input.Body)
	return &manager.UploadOutput{}, err
}

type fakeS3Client struct {
	getOutput   *sdkS3.GetObjectOutput
	getErr      error
	deleteCalls int
}

func (client *fakeS3Client) GetObject(context.Context, *sdkS3.GetObjectInput, ...func(*sdkS3.Options)) (*sdkS3.GetObjectOutput, error) {
	if client.getErr != nil {
		return nil, client.getErr
	}
	return client.getOutput, nil
}

func (client *fakeS3Client) DeleteObject(context.Context, *sdkS3.DeleteObjectInput, ...func(*sdkS3.Options)) (*sdkS3.DeleteObjectOutput, error) {
	client.deleteCalls++
	return &sdkS3.DeleteObjectOutput{}, nil
}

func archiveInfo(content []byte) sandbox.ArchiveInfo {
	sum := sha256.Sum256(content)
	return sandbox.ArchiveInfo{ByteSize: int64(len(content)), SHA256: hex.EncodeToString(sum[:]), MediaType: "application/vnd.haedes.workspace+tar.zstd"}
}

func noWaitRetry() internalaws.RetryPolicy {
	return internalaws.RetryPolicy{MaxAttempts: 3, Sleep: func(context.Context, time.Duration) error { return nil }}
}
