package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	fileDir, err := defaultFileDir()
	if err != nil {
		return nil, err
	}
	ring, err := keyring.Open(keyring.Config{
		ServiceName:              "monarch-cli",
		AllowedBackends:          []keyring.BackendType{keyring.KeychainBackend, keyring.SecretServiceBackend, keyring.KWalletBackend, keyring.WinCredBackend, keyring.PassBackend, keyring.FileBackend},
		FileDir:                  fileDir,
		FilePasswordFunc:         filePasswordFunc(fileDir),
		KeychainTrustApplication: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}
	return &KeyringStore{ring: ring}, nil
}

func defaultFileDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "monarch", "keyring"), nil
}

func filePasswordFunc(fileDir string) keyring.PromptFunc {
	return func(_ string) (string, error) {
		if value := os.Getenv("MONARCH_KEYRING_PASSWORD"); value != "" {
			return value, nil
		}
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown-host"
		}
		sum := sha256.Sum256([]byte("monarch-cli|" + hostname + "|" + fileDir))
		return hex.EncodeToString(sum[:]), nil
	}
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
