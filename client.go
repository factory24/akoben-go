package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// APIError is a non-2xx answer from the platform.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("connectivity: %s (%s, http %d)", e.Message, e.Code, e.Status)
	}
	return fmt.Sprintf("connectivity: %s (http %d)", e.Message, e.Status)
}

// IsNotFound reports whether err is the platform saying the resource does not exist.
func IsNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Status == http.StatusNotFound
}

// IsConflict reports whether err is a state conflict (e.g. no active key, class in use).
func IsConflict(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Status == http.StatusConflict
}

type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Errors  []string        `json:"errors"`
	Data    json.RawMessage `json:"data"`
}

// do performs one API call. body may be nil; out may be nil for calls whose
// data is discarded.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	if c.disabled != nil {
		return c.disabled
	}
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("connectivity: encoding request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return fmt.Errorf("connectivity: obtaining token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if c.businessID != "" {
		req.Header.Set("X-Business-Id", c.businessID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connectivity: %s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("connectivity: reading response: %w", err)
	}

	var env envelope
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil && res.StatusCode < 300 {
			return fmt.Errorf("connectivity: decoding response: %w", err)
		}
	}

	if res.StatusCode >= 300 {
		apiErr := &APIError{Status: res.StatusCode, Message: env.Message}
		if len(env.Errors) > 0 {
			code, msg, found := strings.Cut(env.Errors[0], ": ")
			if found {
				apiErr.Code, apiErr.Message = code, msg
			} else {
				apiErr.Message = env.Errors[0]
			}
		}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(res.StatusCode)
		}
		return apiErr
	}

	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("connectivity: decoding %s %s data: %w", method, path, err)
		}
	}
	return nil
}

func pageQuery(page, pageSize int) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pageSize", strconv.Itoa(pageSize))
	}
	return q
}

func esc(segment string) string { return url.PathEscape(segment) }
