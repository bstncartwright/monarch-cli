package monarch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginSuccessAndMFAAndFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login/" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["username"] {
		case "mfa@example.com":
			if body["totp"] == nil || body["totp"] == "" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"detail":"Multi-Factor Auth Required"}`))
				return
			}
			_, _ = w.Write([]byte(`{"token":"mfa-token"}`))
		case "bad@example.com":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
		default:
			_, _ = w.Write([]byte(`{"token":"plain-token"}`))
		}
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})

	resp, err := client.Login(context.Background(), "user@example.com", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "plain-token" {
		t.Fatalf("unexpected token %q", resp.Token)
	}

	_, err = client.Login(context.Background(), "mfa@example.com", "secret", "")
	if err != ErrMFARequired {
		t.Fatalf("expected MFA required, got %v", err)
	}

	resp, err = client.Login(context.Background(), "mfa@example.com", "secret", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "mfa-token" {
		t.Fatalf("unexpected token %q", resp.Token)
	}

	if _, err = client.Login(context.Background(), "bad@example.com", "secret", ""); err == nil {
		t.Fatal("expected invalid credentials error")
	}
}

func TestGraphQLRequestHeadersAndVariables(t *testing.T) {
	var authHeader string
	var body GraphQLRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"data":{"me":{"id":"user_1","name":"Ada","displayName":"Ada","email":"ada@example.com","hasMfaOn":true,"hasPassword":true,"householdRole":"owner","createdAt":"2026-03-09T00:00:00Z","timezone":"America/Denver"}}}`))
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL, Token: "abc123", HTTPClient: server.Client()})
	resp, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if authHeader != "Token abc123" {
		t.Fatalf("unexpected authorization header %q", authHeader)
	}
	if body.Query == "" {
		t.Fatal("expected graphql query body")
	}
	if resp.Data.Me.Email != "ada@example.com" {
		t.Fatalf("unexpected email %q", resp.Data.Me.Email)
	}
}

func TestListTransactionsPassesPaginationAndDateFilters(t *testing.T) {
	var body GraphQLRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"data":{"allTransactions":{"totalCount":1,"results":[{"id":"tx_1","date":"2026-03-01","amount":12.34,"pending":false,"notes":"","isRecurring":false,"needsReview":false,"hideFromReports":false,"merchant":{"id":"m1","name":"Coffee"},"category":{"id":"c1","name":"Food","group":{"id":"g1","name":"Food","type":"expense"}},"account":{"id":"a1","displayName":"Checking"},"tags":[]}]}}}`))
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL, Token: "tok", HTTPClient: server.Client()})
	_, err := client.ListTransactions(context.Background(), TransactionsListParams{
		Limit:     25,
		Offset:    50,
		OrderBy:   "inverse_date",
		StartDate: "2026-02-01",
		EndDate:   "2026-02-29",
		Search:    "coffee",
	})
	if err != nil {
		t.Fatal(err)
	}
	if body.Variables["limit"].(float64) != 25 {
		t.Fatalf("unexpected limit: %#v", body.Variables["limit"])
	}
	if body.Variables["offset"].(float64) != 50 {
		t.Fatalf("unexpected offset: %#v", body.Variables["offset"])
	}
	filters := body.Variables["filters"].(map[string]any)
	if filters["startDate"] != "2026-02-01" || filters["endDate"] != "2026-02-29" || filters["search"] != "coffee" {
		t.Fatalf("unexpected filters: %#v", filters)
	}
}

func TestGetAccountHoldingsUsesTodayDate(t *testing.T) {
	var body GraphQLRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[]}}}}`))
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL, Token: "tok", HTTPClient: server.Client()})
	_, err := client.GetAccountHoldings(context.Background(), "acct_1", time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	input := body.Variables["input"].(map[string]any)
	if input["startDate"] != "2026-03-09" || input["endDate"] != "2026-03-09" {
		t.Fatalf("unexpected holdings dates: %#v", input)
	}
}
