package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bstn/monarch-cli/internal/app"
	"github.com/bstn/monarch-cli/internal/config"
	"github.com/bstn/monarch-cli/internal/secrets"
)

func TestCommandGoldenJSON(t *testing.T) {
	server := newCommandServer(t)
	defer server.Close()

	cfgManager, rt, store := newTestRuntime(t, server)
	if _, err := rt.UpsertProfile("default", "ada@example.com", "stored-token"); err != nil {
		t.Fatal(err)
	}

	t.Run("me get", func(t *testing.T) {
		var stdout bytes.Buffer
		if err := ExecuteWithOptions([]string{"me", "get"}, ExecuteOptions{
			Stdout:        &stdout,
			Stderr:        ioDiscard{},
			ConfigManager: cfgManager,
			SecretStore:   store,
			HTTPClient:    server.Client(),
			BaseURL:       server.URL,
		}); err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("testdata", "me_get.golden.json"))
		if err != nil {
			t.Fatal(err)
		}
		if stdout.String() != string(expected) {
			t.Fatalf("unexpected me output\nwant:\n%s\ngot:\n%s", string(expected), stdout.String())
		}
	})

	t.Run("accounts list", func(t *testing.T) {
		var stdout bytes.Buffer
		if err := ExecuteWithOptions([]string{"accounts", "list"}, ExecuteOptions{
			Stdout:        &stdout,
			Stderr:        ioDiscard{},
			ConfigManager: cfgManager,
			SecretStore:   store,
			HTTPClient:    server.Client(),
			BaseURL:       server.URL,
		}); err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("testdata", "accounts_list.golden.json"))
		if err != nil {
			t.Fatal(err)
		}
		if stdout.String() != string(expected) {
			t.Fatalf("unexpected accounts output\nwant:\n%s\ngot:\n%s", string(expected), stdout.String())
		}
	})
}

func TestTableOutputSmoke(t *testing.T) {
	server := newCommandServer(t)
	defer server.Close()
	cfgManager, rt, store := newTestRuntime(t, server)
	if _, err := rt.UpsertProfile("default", "ada@example.com", "stored-token"); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := ExecuteWithOptions([]string{"--format", "table", "accounts", "list"}, ExecuteOptions{
		Stdout:        &stdout,
		Stderr:        ioDiscard{},
		ConfigManager: cfgManager,
		SecretStore:   store,
		HTTPClient:    server.Client(),
		BaseURL:       server.URL,
	}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "Checking") {
		t.Fatalf("unexpected table output: %s", out)
	}
}

func TestGraphQLRawQueryWithVariables(t *testing.T) {
	var seenQuery string
	var seenVariables map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		seenQuery = body["query"].(string)
		seenVariables = body["variables"].(map[string]any)
		_, _ = w.Write([]byte(`{"data":{"me":{"id":"user_1","email":"ada@example.com"}}}`))
	}))
	defer server.Close()

	cfgManager, _, store := newTestRuntime(t, server)
	var stdout bytes.Buffer
	if err := ExecuteWithOptions([]string{
		"--token", "override-token",
		"graphql", "query",
		"query Me($id: ID!) { me { id email } }",
		"--variables", `{"id":"user_1"}`,
	}, ExecuteOptions{
		Stdout:        &stdout,
		Stderr:        ioDiscard{},
		ConfigManager: cfgManager,
		SecretStore:   store,
		HTTPClient:    server.Client(),
		BaseURL:       server.URL,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seenQuery, "query Me") {
		t.Fatalf("unexpected query: %s", seenQuery)
	}
	if seenVariables["id"] != "user_1" {
		t.Fatalf("unexpected variables: %#v", seenVariables)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	data := payload["data"].(map[string]any)
	me := data["me"].(map[string]any)
	if me["email"] != "ada@example.com" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func newTestRuntime(t *testing.T, server *httptest.Server) (*config.Manager, *app.Runtime, *secrets.MemoryStore) {
	t.Helper()
	cfgManager, err := config.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	store := secrets.NewMemoryStore()
	rt, err := app.New(app.Options{
		Profile:       "default",
		ConfigManager: cfgManager,
		SecretStore:   store,
		HTTPClient:    server.Client(),
		BaseURL:       server.URL,
		Now: func() time.Time {
			return time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return cfgManager, rt, store
}

func newCommandServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			query := body["query"].(string)
			switch {
			case strings.Contains(query, "query Me"):
				_, _ = w.Write([]byte(`{"data":{"me":{"id":"user_1","name":"Ada Lovelace","displayName":"Ada","email":"ada@example.com","hasMfaOn":true,"hasPassword":true,"householdRole":"owner","createdAt":"2026-03-09T00:00:00Z","timezone":"America/Denver"}}}`))
			case strings.Contains(query, "query GetAccounts"):
				_, _ = w.Write([]byte(`{"data":{"accounts":[{"id":"acct_1","displayName":"Checking","currentBalance":1234.56,"displayLastUpdatedAt":"2026-03-09T00:00:00Z","includeInNetWorth":true,"hideFromList":false,"hideTransactionsFromReports":false,"isAsset":true,"isHidden":false,"isManual":false,"mask":"1234","transactionsCount":42,"holdingsCount":0,"dataProvider":"plaid","logoUrl":"","type":{"name":"depository","display":"Depository"},"subtype":{"name":"checking","display":"Checking"},"institution":{"id":"inst_1","name":"Bank","url":"https://bank.example.com"}}]}}`))
			default:
				t.Fatalf("unexpected query: %s", query)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}
