package auth

import (
	"context"
	"testing"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestServiceAuthenticatesAndScopesOwner(t *testing.T) {
	record, err := NewAPIKeyRecord("key_123", "owner_456", "secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if string(record.Digest) == "secret-value" {
		t.Fatal("record must not store plaintext API key")
	}
	store, err := NewMemoryStore(record)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}

	principal, err := service.Authenticate(context.Background(), "secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if principal != (sandbox.Principal{OwnerID: "owner_456", KeyID: "key_123"}) {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestServiceRejectsUnknownKey(t *testing.T) {
	record, err := NewAPIKeyRecord("key_123", "owner_456", "secret-value")
	if err != nil {
		t.Fatal(err)
	}
	store, _ := NewMemoryStore(record)
	service, _ := NewService(store)
	if _, err := service.Authenticate(context.Background(), "wrong-value"); err != ErrInvalidAPIKey {
		t.Fatalf("expected invalid API key, got %v", err)
	}
}

func TestMemoryStoreDoesNotExposeMutableRecords(t *testing.T) {
	record, err := NewAPIKeyRecord("key_123", "owner_456", "secret-value")
	if err != nil {
		t.Fatal(err)
	}
	store, _ := NewMemoryStore(record)
	records, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	records[0].Digest[0] ^= 0xff
	again, _ := store.List(context.Background())
	if string(records[0].Digest) == string(again[0].Digest) {
		t.Fatal("store returned mutable internal record")
	}
}
