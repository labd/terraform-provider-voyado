package engage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "terraform-provider-voyado/dev"

// Client calls Voyado Engage HTTP API v3.
type Client struct {
	baseURL    string
	apiKey     string
	userAgent  string
	httpClient *http.Client
}

// NewClient builds a client for the given Engage API base URL (scheme, host, optional path prefix).
func NewClient(apiURL, apiKey string, httpClient *http.Client) (*Client, error) {
	baseURL, err := normalizeBaseAPIURL(apiURL)
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key must not be empty")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		userAgent:  defaultUserAgent,
		httpClient: httpClient,
	}, nil
}

func normalizeBaseAPIURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("api_url must not be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse api_url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("api_url must use http or https (got scheme %q)", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("api_url must include a host")
	}
	u.Fragment = ""
	u.RawQuery = ""
	switch {
	case u.Path == "" || u.Path == "/":
		u.Path = "/"
	case !strings.HasSuffix(u.Path, "/"):
		u.Path += "/"
	}
	return u.String(), nil
}

// WithUserAgent overrides the default User-Agent header (recommended by Voyado).
func (c *Client) WithUserAgent(ua string) {
	if strings.TrimSpace(ua) != "" {
		c.userAgent = ua
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	path = strings.TrimPrefix(path, "/")
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

// CreateInteractionSchema POST /api/v3/interactionschemas
func (c *Client) CreateInteractionSchema(ctx context.Context, jsonBody []byte) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "api/v3/interactionschemas", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	body, code, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, statusErr(code, body)
	}
	return body, nil
}

// GetInteractionSchema GET /api/v3/interactionschemas/{id}
func (c *Client) GetInteractionSchema(ctx context.Context, schemaID string) ([]byte, error) {
	path := fmt.Sprintf("api/v3/interactionschemas/%s", url.PathEscape(schemaID))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	body, code, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, statusErr(code, body)
	}
	return body, nil
}

// DeleteInteractionSchema DELETE /api/v3/interactionschemas/{id}
func (c *Client) DeleteInteractionSchema(ctx context.Context, schemaID string) error {
	path := fmt.Sprintf("api/v3/interactionschemas/%s", url.PathEscape(schemaID))
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	body, code, err := c.do(req)
	if err != nil {
		return err
	}
	if code == http.StatusOK || code == http.StatusNoContent || code == http.StatusNotFound {
		return nil
	}
	return statusErr(code, body)
}

func statusErr(code int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if len(msg) > 2048 {
		msg = msg[:2048] + "…"
	}
	if msg == "" {
		return fmt.Errorf("engage API returned HTTP %d", code)
	}
	return fmt.Errorf("engage API returned HTTP %d: %s", code, msg)
}
