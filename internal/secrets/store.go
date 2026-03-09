package secrets

import (
	"errors"
	"fmt"

	"github.com/99designs/keyring"
)

var ErrNotFound = errors.New("secret not found")

type Store interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

type KeyringStore struct {
	ring keyring.Keyring
}

func NewKeyringStore() (*KeyringStore, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:              "monarch-cli",
		AllowedBackends:          []keyring.BackendType{keyring.KeychainBackend, keyring.SecretServiceBackend, keyring.KWalletBackend, keyring.WinCredBackend, keyring.PassBackend, keyring.FileBackend},
		FileDir:                  "",
		FilePasswordFunc:         nil,
		KeychainTrustApplication: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}
	return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) Get(key string) (string, error) {
	item, err := s.ring.Get(key)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read keyring item: %w", err)
	}
	return string(item.Data), nil
}

func (s *KeyringStore) Set(key, value string) error {
	if err := s.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(value),
	}); err != nil {
		return fmt.Errorf("write keyring item: %w", err)
	}
	return nil
}

func (s *KeyringStore) Delete(key string) error {
	err := s.ring.Remove(key)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete keyring item: %w", err)
	}
	return nil
}

type MemoryStore struct {
	Items map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{Items: map[string]string{}}
}

func (s *MemoryStore) Get(key string) (string, error) {
	value, ok := s.Items[key]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (s *MemoryStore) Set(key, value string) error {
	s.Items[key] = value
	return nil
}

func (s *MemoryStore) Delete(key string) error {
	delete(s.Items, key)
	return nil
}
