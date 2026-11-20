package es

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"elastic-ui/internal/auth"
)

type Client struct {
	baseURL string
	auth    auth.Auth
	http    *http.Client
}

type APIError struct {
	Status  int
	Kind    string
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: http %d", e.Kind, e.Status)
}

func New(baseURL string, a auth.Auth, skipTLS bool) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("elasticsearch url is empty")
	}
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	return &Client{
		baseURL: baseURL,
		auth:    a,
		http: &http.Client{
			Transport: transport,
			Timeout:   45 * time.Second,
		},
	}, nil
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Search(ctx context.Context, index string, body any, out any) error {
	return c.do(ctx, http.MethodPost, "/"+index+"/_search", body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ElasticDefendConsole/0.1")
	c.auth.Apply(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return &APIError{Kind: "network", Message: err.Error()}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return &APIError{Kind: "network", Message: err.Error()}
	}
	if resp.StatusCode >= 400 {
		apiErr := &APIError{Status: resp.StatusCode, Body: string(data)}
		var payload struct {
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(data, &payload) == nil && len(payload.Error) > 0 {
			var nested struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			}
			if json.Unmarshal(payload.Error, &nested) == nil {
				apiErr.Message = nested.Reason
			}
		}
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(data))
		}
		lower := strings.ToLower(string(data))
		if resp.StatusCode == http.StatusForbidden && (strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype")) {
			apiErr.Kind = "waf"
			apiErr.Message = "请求被 WAF 拦截（HTTP 403）"
			return apiErr
		}
		if resp.StatusCode >= 500 && (strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype")) {
			apiErr.Kind = "gateway"
			apiErr.Message = fmt.Sprintf("Elasticsearch 暂时不可用（HTTP %d）", resp.StatusCode)
			return apiErr
		}
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			apiErr.Kind = "unauthorized"
		case http.StatusForbidden:
			apiErr.Kind = "forbidden"
		case http.StatusNotFound:
			apiErr.Kind = "not_found"
		default:
			apiErr.Kind = "http"
		}
		return apiErr
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
