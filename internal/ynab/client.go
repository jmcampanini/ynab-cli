// Package ynab is a hand-written client for the YNAB API v1 covering the
// endpoints the CLI uses. Every call takes a context and returns typed
// records; API errors surface as *Error.
package ynab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the production API root.
const DefaultBaseURL = "https://api.ynab.com/v1"

// maxErrorBody bounds how much of an unexpected response is read for the
// error message.
const maxErrorBody = 4096

// Client calls the YNAB API with one personal access token.
type Client struct {
	// BaseURL is the API root without a trailing slash. Empty means
	// DefaultBaseURL.
	BaseURL string
	// HTTPClient performs the requests. Nil means a client with a 30-second
	// timeout.
	HTTPClient *http.Client
	// Token is the personal access token sent as a bearer token.
	Token string
	// UserAgent identifies the CLI, such as "ynab-cli/v0.1.0".
	UserAgent string
}

// get performs a GET request and decodes the "data" envelope into out.
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, nil, out)
}

// getQuery performs a GET request with a query string and decodes the
// "data" envelope into out.
func (c *Client) getQuery(ctx context.Context, path string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out)
}

// do performs one request and decodes the "data" envelope into out. A
// non-nil body is sent as JSON. Any 2xx status is a success; the API
// answers some creates with 200 and others with 201.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request %s: %w", path, err)
		}
		payload = encoded
	}
	response, err := c.send(ctx, method, path, payload)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return decodeError(response)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}
	return nil
}

// RawResponse is the status and body of a request made with Raw, exactly
// as the API returned them.
type RawResponse struct {
	Body   []byte
	Status int
}

// Raw performs one request with the path and JSON body as given and
// returns the status and body without decoding them, so a non-2xx status
// is a response, not an error. A nil body sends no body.
func (c *Client) Raw(ctx context.Context, method, path string, body []byte) (RawResponse, error) {
	response, err := c.send(ctx, method, path, body)
	if err != nil {
		return RawResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()

	read, err := io.ReadAll(response.Body)
	if err != nil {
		return RawResponse{}, fmt.Errorf("read response from %s: %w", path, err)
	}
	return RawResponse{Body: read, Status: response.StatusCode}, nil
}

// URL returns the absolute URL a path relative to the API root resolves to.
func (c *Client) URL(path string) string {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return base + path
}

// send performs one request with the token and the CLI's headers. A
// non-nil payload is sent as JSON.
func (c *Client) send(ctx context.Context, method, path string, payload []byte) (*http.Response, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.URL(path), reader)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", path, err)
	}
	request.Header.Set("Authorization", "Bearer "+c.Token)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		request.Header.Set("User-Agent", c.UserAgent)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return nil, fmt.Errorf("request %s timed out after 30s", path)
		}
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	return response, nil
}

// decodeError maps a non-2xx response to *Error. A body that is not the
// documented error envelope still yields an *Error carrying the status.
func decodeError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
	var envelope struct {
		Error Error `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.ID != "" {
		apiError := envelope.Error
		apiError.Status = response.StatusCode
		return &apiError
	}
	return &Error{
		Status: response.StatusCode,
		ID:     fmt.Sprint(response.StatusCode),
		Name:   http.StatusText(response.StatusCode),
		Detail: string(body),
	}
}
