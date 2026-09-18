package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	ErrInvalidAPIKey = errors.New("invalid API key")
	ErrInvalidRecord = errors.New("invalid API key record")
)

const (
	saltSize   = 16
	digestSize = sha256.Size
)

// APIKeyRecord is the persisted representation of an API key. Plaintext API
// keys are deliberately not part of this type so callers cannot accidentally
// persist or log them.
type APIKeyRecord struct {
	KeyID   string
	OwnerID string
	Salt    []byte
	Digest  []byte
}

// NewAPIKeyRecord hashes plaintext with a random per-key salt. Only the
// returned record should be stored; the plaintext belongs at the provisioning
// boundary and is never retained by the authenticator.
func NewAPIKeyRecord(keyID, ownerID, plaintext string) (APIKeyRecord, error) {
	return newAPIKeyRecord(rand.Reader, keyID, ownerID, plaintext)
}

func newAPIKeyRecord(random io.Reader, keyID, ownerID, plaintext string) (APIKeyRecord, error) {
	if keyID == "" || ownerID == "" || plaintext == "" {
		return APIKeyRecord{}, ErrInvalidRecord
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(random, salt); err != nil {
		return APIKeyRecord{}, err
	}
	return APIKeyRecord{
		KeyID:   keyID,
		OwnerID: ownerID,
		Salt:    salt,
		Digest:  digest(salt, plaintext),
	}, nil
}

func digest(salt []byte, plaintext string) []byte {
	hash := sha256.New()
	_, _ = hash.Write(salt)
	_, _ = hash.Write([]byte(plaintext))
	return hash.Sum(nil)
}

func validateRecord(record APIKeyRecord) error {
	if record.KeyID == "" || record.OwnerID == "" || len(record.Salt) != saltSize || len(record.Digest) != digestSize {
		return ErrInvalidRecord
	}
	return nil
}

// Store supplies hashed records to the authenticator. Implementations may be
// backed by a database or secret manager without changing authentication.
type Store interface {
	List(ctx context.Context) ([]APIKeyRecord, error)
}

// MemoryStore is useful for local development and deterministic tests. It
// stores only APIKeyRecord values, never plaintext keys.
type MemoryStore struct {
	mu      sync.RWMutex
	records []APIKeyRecord
}

func NewMemoryStore(records ...APIKeyRecord) (*MemoryStore, error) {
	store := &MemoryStore{}
	for _, record := range records {
		if err := store.Add(record); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *MemoryStore) Add(record APIKeyRecord) error {
	if err := validateRecord(record); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, cloneRecord(record))
	return nil
}

func (s *MemoryStore) List(ctx context.Context) ([]APIKeyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]APIKeyRecord, len(s.records))
	for i, record := range s.records {
		records[i] = cloneRecord(record)
	}
	return records, nil
}

func cloneRecord(record APIKeyRecord) APIKeyRecord {
	return APIKeyRecord{
		KeyID:   record.KeyID,
		OwnerID: record.OwnerID,
		Salt:    append([]byte(nil), record.Salt...),
		Digest:  append([]byte(nil), record.Digest...),
	}
}

// Service authenticates bearer keys against salted hashes and maps successful
// keys to owner-scoped principals. Every record is compared before returning,
// avoiding an early-exit timing signal for the matching key.
type Service struct {
	store Store
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	return &Service{store: store}, nil
}

func (s *Service) Authenticate(ctx context.Context, apiKey string) (sandbox.Principal, error) {
	if apiKey == "" {
		return sandbox.Principal{}, ErrInvalidAPIKey
	}
	records, err := s.store.List(ctx)
	if err != nil {
		return sandbox.Principal{}, err
	}

	var principal sandbox.Principal
	matched := 0
	for _, record := range records {
		if validateRecord(record) != nil {
			continue
		}
		candidate := digest(record.Salt, apiKey)
		equal := subtle.ConstantTimeCompare(candidate, record.Digest)
		if equal == 1 {
			principal = sandbox.Principal{OwnerID: record.OwnerID, KeyID: record.KeyID}
		}
		matched |= equal
	}
	if matched != 1 {
		return sandbox.Principal{}, ErrInvalidAPIKey
	}
	return principal, nil
}
