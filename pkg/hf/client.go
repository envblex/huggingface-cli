package hf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client interacts with the Hugging Face Hub REST API.
type Client struct {
	Endpoint   string
	HTTPClient *http.Client
}

// NewClient returns a new Client with default or custom endpoint.
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = Endpoint()
	}
	endpoint = strings.TrimRight(endpoint, "/")

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// If redirect host differs from initial host, do not leak Auth header
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
			}
			return nil
		},
	}

	return &Client{
		Endpoint:   endpoint,
		HTTPClient: httpClient,
	}
}

// DoRequest performs an HTTP request, setting User-Agent and optionally Authorization.
func (c *Client) DoRequest(ctx context.Context, method, path string, body any, token string) (*http.Response, error) {
	var bodyReader io.Reader
	var contentType string

	if body != nil {
		switch b := body.(type) {
		case io.Reader:
			bodyReader = b
		case []byte:
			bodyReader = bytes.NewReader(b)
		case string:
			bodyReader = strings.NewReader(b)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
			contentType = "application/json"
		}
	}

	reqURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		reqURL = c.Endpoint + path
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if token == "" {
		token, _ = GetToken()
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	return c.HTTPClient.Do(req)
}
