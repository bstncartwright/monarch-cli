package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bstn/monarch-cli/internal/config"
	"github.com/bstn/monarch-cli/internal/secrets"
)

func TestResolveTokenPrecedence(t *testing.T) {
	cfgManager, err := config.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	store := secrets.NewMemoryStore()
	rt, err := New(Options{
		Profile:       "default",
		ConfigManager: cfgManager,
		SecretStore:   store,
		Now: func() time.Time {
			return time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.UpsertProfile("default", "user@example.com", "stored-token"); err != nil {
		t.Fatal(err)
	}

	cfg, err := rt.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("MONARCH_TOKEN", "env-token")
	token, source, err := rt.ResolveToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if token != "env-token" || source != "env" {
		t.Fatalf("expected env token precedence, got token=%q source=%q", token, source)
	}

	rtFlag, err := New(Options{
		Profile:       "default",
		TokenOverride: "flag-token",
		ConfigManager: cfgManager,
		SecretStore:   store,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, source, err = rtFlag.ResolveToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if token != "flag-token" || source != "flag" {
		t.Fatalf("expected flag token precedence, got token=%q source=%q", token, source)
	}

	t.Setenv("MONARCH_TOKEN", "")
	token, source, err = rt.ResolveToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if token != "stored-token" || source != "keyring" {
		t.Fatalf("expected stored token fallback, got token=%q source=%q", token, source)
	}
}

func TestUpsertProfilePersistsTokenAndConfig(t *testing.T) {
	cfgManager, err := config.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	store := secrets.NewMemoryStore()
	rt, err := New(Options{
		Profile:       "work",
		ConfigManager: cfgManager,
		SecretStore:   store,
		Now: func() time.Time {
			return time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := rt.UpsertProfile("work", "work@example.com", "token-123")
	if err != nil {
		t.Fatal(err)
	}
	if profile.KeyringKey == "" {
		t.Fatal("expected keyring key to be set")
	}
	token, err := store.Get(profile.KeyringKey)
	if err != nil {
		t.Fatal(err)
	}
	if token != "token-123" {
		t.Fatalf("unexpected stored token %q", token)
	}
	cfg, err := rt.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Profiles["work"]
	if got == nil || got.Email != "work@example.com" {
		t.Fatalf("unexpected stored profile: %#v", got)
	}
}
