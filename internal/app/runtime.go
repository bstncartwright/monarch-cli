package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bstn/monarch-cli/internal/config"
	"github.com/bstn/monarch-cli/internal/monarch"
	"github.com/bstn/monarch-cli/internal/output"
	"github.com/bstn/monarch-cli/internal/secrets"
)

type Options struct {
	Stdout        io.Writer
	Stderr        io.Writer
	Format        string
	Raw           bool
	Profile       string
	TokenOverride string
	ConfigPath    string
	BaseURL       string
	HTTPClient    *http.Client
	ConfigManager *config.Manager
	SecretStore   secrets.Store
	Now           func() time.Time
}

type Runtime struct {
	Stdout io.Writer
	Stderr io.Writer

	Format string
	Raw    bool

	profileName string
	baseURL     string
	httpClient  *http.Client
	config      *config.Manager
	secrets     secrets.Store
	now         func() time.Time
	tokenFlag   string
}

func New(opts Options) (*Runtime, error) {
	cfgManager := opts.ConfigManager
	if cfgManager == nil {
		var err error
		cfgManager, err = config.NewManager(opts.ConfigPath)
		if err != nil {
			return nil, err
		}
	}
	store := opts.SecretStore
	if store == nil {
		var err error
		store, err = secrets.NewKeyringStore()
		if err != nil {
			return nil, err
		}
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	profile := opts.Profile
	if profile == "" {
		profile = "default"
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("MONARCH_BASE_URL")
	}
	if baseURL == "" {
		baseURL = monarch.DefaultBaseURL
	}
	return &Runtime{
		Stdout:      defaultWriter(opts.Stdout, os.Stdout),
		Stderr:      defaultWriter(opts.Stderr, os.Stderr),
		Format:      opts.Format,
		Raw:         opts.Raw,
		profileName: profile,
		baseURL:     strings.TrimRight(baseURL, "/"),
		httpClient:  httpClient,
		config:      cfgManager,
		secrets:     store,
		now:         nowFn,
		tokenFlag:   opts.TokenOverride,
	}, nil
}

func defaultWriter(given io.Writer, fallback io.Writer) io.Writer {
	if given != nil {
		return given
	}
	return fallback
}

func (r *Runtime) Print(data any, table output.Table) error {
	if r.Format == "table" && len(table.Headers) > 0 {
		return output.WriteTable(r.Stdout, table)
	}
	return output.WriteJSON(r.Stdout, data)
}

func (r *Runtime) PrintJSON(data any) error {
	return output.WriteJSON(r.Stdout, data)
}

func (r *Runtime) LoadConfig() (*config.File, error) {
	return r.config.Load()
}

func (r *Runtime) SaveConfig(cfg *config.File) error {
	return r.config.Save(cfg)
}

func (r *Runtime) ActiveProfileName() string {
	return r.profileName
}

func (r *Runtime) ConfigPath() string {
	return r.config.Path
}

func (r *Runtime) ResolveProfile(cfg *config.File) *config.Profile {
	if cfg == nil {
		return nil
	}
	name := r.profileName
	if name == "" {
		name = cfg.DefaultProfile
	}
	if name == "" {
		name = "default"
	}
	return cfg.Profiles[name]
}

func (r *Runtime) ResolveToken(cfg *config.File) (string, string, error) {
	if strings.TrimSpace(r.tokenFlag) != "" {
		return strings.TrimSpace(r.tokenFlag), "flag", nil
	}
	if value := strings.TrimSpace(os.Getenv("MONARCH_TOKEN")); value != "" {
		return value, "env", nil
	}
	profile := r.ResolveProfile(cfg)
	if profile == nil || profile.KeyringKey == "" {
		return "", "", fmt.Errorf("no stored token for profile %q", r.profileName)
	}
	token, err := r.secrets.Get(profile.KeyringKey)
	if err != nil {
		return "", "", err
	}
	return token, "keyring", nil
}

func (r *Runtime) UpsertProfile(name, email, token string) (*config.Profile, error) {
	cfg, err := r.LoadConfig()
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	key := "profile:" + name + ":token"
	profile := cfg.Profiles[name]
	if profile == nil {
		profile = &config.Profile{
			Name:       name,
			CreatedAt:  now,
			KeyringKey: key,
		}
	}
	profile.Email = email
	profile.KeyringKey = key
	profile.UpdatedAt = now
	profile.LastSuccessfulAPIURL = r.baseURL
	cfg.Profiles[name] = profile
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := r.secrets.Set(key, token); err != nil {
		return nil, err
	}
	if err := r.SaveConfig(cfg); err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *Runtime) DeleteProfileToken(name string) error {
	cfg, err := r.LoadConfig()
	if err != nil {
		return err
	}
	profile := cfg.Profiles[name]
	if profile == nil {
		return nil
	}
	if profile.KeyringKey != "" {
		if err := r.secrets.Delete(profile.KeyringKey); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) ClientFromConfig(ctx context.Context) (*monarch.Client, *config.Profile, string, error) {
	cfg, err := r.LoadConfig()
	if err != nil {
		return nil, nil, "", err
	}
	token, source, err := r.ResolveToken(cfg)
	if err != nil {
		return nil, r.ResolveProfile(cfg), "", err
	}
	client := monarch.NewClient(monarch.Options{
		BaseURL:    r.baseURL,
		Token:      token,
		HTTPClient: r.httpClient,
	})
	return client, r.ResolveProfile(cfg), source, nil
}

func (r *Runtime) NewLoginClient() *monarch.Client {
	return monarch.NewClient(monarch.Options{
		BaseURL:    r.baseURL,
		HTTPClient: r.httpClient,
	})
}
