package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultConfigRelPath = "monarch/config.json"

type Profile struct {
	Name                 string    `json:"name"`
	Email                string    `json:"email"`
	KeyringKey           string    `json:"keyring_key"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	LastSuccessfulAPIURL string    `json:"last_successful_api_url,omitempty"`
}

type File struct {
	DefaultProfile string              `json:"default_profile"`
	Profiles       map[string]*Profile `json:"profiles"`
}

type Manager struct {
	Path string
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, defaultConfigRelPath), nil
}

func NewManager(path string) (*Manager, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	return &Manager{Path: path}, nil
}

func (m *Manager) Load() (*File, error) {
	data, err := os.ReadFile(m.Path)
	if errors.Is(err, os.ErrNotExist) {
		return &File{
			DefaultProfile: "default",
			Profiles:       map[string]*Profile{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = "default"
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	return &cfg, nil
}

func (m *Manager) Save(cfg *File) error {
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = "default"
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(m.Path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
