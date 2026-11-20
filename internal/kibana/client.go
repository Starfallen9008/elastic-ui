package kibana

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"elastic-ui/internal/auth"
)

type Client struct {
	baseURL string
	space   string
	auth    auth.Auth
	http    *http.Client

	Version string
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

func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Status == http.StatusNotFound
}

func IsForbidden(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && (apiErr.Status == http.StatusForbidden || apiErr.Status == http.StatusUnauthorized)
}

func New(baseURL string, a auth.Auth, skipTLS bool, space string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid kibana url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid kibana url: %s", baseURL)
	}
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	c := &Client{
		baseURL: baseURL,
		space:   strings.TrimSpace(space),
		auth:    a,
		http: &http.Client{
			Transport: transport,
			Timeout:   45 * time.Second,
		},
	}
	return c, nil
}

func (c *Client) SpacePrefix() string {
	if c.space == "" || c.space == "default" {
		return ""
	}
	return "/s/" + c.space
}

func (c *Client) url(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + c.SpacePrefix() + path
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) Put(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) Delete(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodDelete, path, nil, out)
}

// Download 下载二进制响应，用于 get_file 返回的 ZIP。
func (c *Client) Download(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/zip, application/octet-stream, */*")
	req.Header.Set("User-Agent", "ElasticDefendConsole/0.1")
	c.auth.Apply(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Kind: "network", Message: err.Error()}
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if readErr != nil {
		return nil, &APIError{Kind: "network", Message: readErr.Error()}
	}
	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, data)
	}
	return data, nil
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

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ElasticDefendConsole/0.1")
	if method != http.MethodGet {
		req.Header.Set("kbn-xsrf", "true")
	}
	c.auth.Apply(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return &APIError{Kind: "network", Message: err.Error()}
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if readErr != nil {
		return &APIError{Kind: "network", Message: readErr.Error()}
	}

	if resp.StatusCode >= 400 {
		return parseError(resp.StatusCode, data)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func parseError(status int, data []byte) *APIError {
	apiErr := &APIError{Status: status, Body: string(data)}
	var payload struct {
		StatusCode int    `json:"statusCode"`
		Error      string `json:"error"`
		Message    string `json:"message"`
	}
	if json.Unmarshal(data, &payload) == nil {
		apiErr.Message = payload.Message
		if apiErr.Message == "" {
			apiErr.Message = payload.Error
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(data))
	}
	lower := strings.ToLower(string(data))
	if status == http.StatusForbidden && (strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<html")) {
		apiErr.Kind = "waf"
		apiErr.Message = "请求被 WAF 或反向代理拦截（HTTP 403）"
		return apiErr
	}
	if status >= 500 && (strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<html")) {
		apiErr.Kind = "gateway"
		apiErr.Message = fmt.Sprintf("上游暂时不可用（HTTP %d）", status)
		return apiErr
	}
	switch status {
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

func (c *Client) LoadVersion(ctx context.Context) error {
	var status struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	if err := c.Get(ctx, "/api/status", &status); err != nil {
		return err
	}
	c.Version = status.Version.Number
	return nil
}
