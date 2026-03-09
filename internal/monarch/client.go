package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.monarch.com"

var (
	ErrMFARequired     = errors.New("multi-factor authentication required")
	ErrInvalidResponse = errors.New("invalid response from monarch")
)

type Options struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type LoginResult struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	Detail      string `json:"detail"`
	Error       string `json:"error"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
}

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type GraphQLResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

func NewClient(opts Options) *Client {
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL: baseURL,
		token:   opts.Token,
		http:    httpClient,
	}
}

func (c *Client) Login(ctx context.Context, email, password, totp string) (*LoginResult, error) {
	payload := map[string]any{
		"username":       email,
		"password":       password,
		"supports_mfa":   true,
		"trusted_device": false,
	}
	if strings.TrimSpace(totp) != "" {
		payload["totp"] = strings.TrimSpace(totp)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Client-Platform", "web")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "monarch-cli")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result LoginResult
	_ = json.Unmarshal(responseBody, &result)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && strings.TrimSpace(totp) == "" {
			return nil, ErrMFARequired
		}
		message := loginErrorMessage(result, strings.TrimSpace(string(responseBody)))
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("login failed: %s", message)
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if result.Token == "" && result.AccessToken != "" {
		result.Token = result.AccessToken
	}
	if result.Token == "" {
		if message := loginErrorMessage(result, strings.TrimSpace(string(body))); message != "" {
			return nil, fmt.Errorf("login failed: %s", message)
		}
		return nil, fmt.Errorf("%w: missing token in login response", ErrInvalidResponse)
	}
	return &result, nil
}

func loginErrorMessage(result LoginResult, fallback string) string {
	for _, candidate := range []string{
		result.Detail,
		result.Message,
		result.Error,
		result.ErrorCode,
		fallback,
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func (c *Client) Query(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(GraphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Client-Platform", "web")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "monarch-cli")
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Token "+strings.TrimSpace(c.token))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql request failed: %s", strings.TrimSpace(string(payload)))
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return nil
}
