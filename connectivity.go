// Package connectivity is the Go client for the Network Co. device-connectivity
// platform. It is the package a customer imports instead of a vendor client:
// every type here is owned by this SDK, nothing names or hints at the
// technology delivering the connectivity, and no generated protocol types leak
// through the surface.
//
// Vocabulary (see docs/API-CONTRACT.md): a connected unit is a Device,
// identified by DeviceID; radio infrastructure is an AccessPoint; a device
// type template is a DeviceClass; downlinks are Commands; uplinks are
// Messages, delivered to the customer's own endpoint, never fetched.
//
// There is deliberately no way to read a device secret back. SetDeviceSecret
// is write-only: a credential that can be fetched is a credential that leaks
// through any read-only role.
package connectivity

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Bearer is how a device's connectivity is delivered.
type Bearer string

const (
	BearerLPWAN    Bearer = "lorwan"
	BearerCellular Bearer = "cellular"
)

// DeviceState is the lifecycle state of a device.
type DeviceState string

const (
	// StateProvisioned means the device is registered but has never been seen.
	StateProvisioned DeviceState = "provisioned"
	StateActive      DeviceState = "active"
	StateInactive    DeviceState = "inactive"
)

// Client is a connectivity API client. Construct with New.
type Client struct {
	baseURL    string
	tokens     TokenSource
	businessID string
	httpClient *http.Client
	disabled   error
}

// NewDisabled returns a Client whose every call fails with reason. It lets a
// consumer wire the client unconditionally and only enable it where
// configuration exists, without nil checks at every call site.
func NewDisabled(reason string) *Client {
	return &Client{disabled: fmt.Errorf("connectivity: client disabled: %s", reason)}
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying *http.Client (default: 30s timeout).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithBusinessID sets the business acted for on every request. Only staff
// principals need this; a customer token already carries its business.
func WithBusinessID(id string) Option {
	return func(c *Client) { c.businessID = id }
}

// New returns a Client for the API at baseURL (e.g. "https://api.example.com/v1"
// or "http://localhost:8099/v1"). tokens supplies the bearer credential per
// request; use StaticToken or NewClientCredentialsTokenSource.
func New(baseURL string, tokens TokenSource, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		tokens:     tokens,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetDeviceState returns just the lifecycle state of a device. It is a
// convenience over GetDevice for callers that only need to know whether the
// device has ever been seen ("provisioned" means it has not).
func (c *Client) GetDeviceState(ctx context.Context, deviceID string) (DeviceState, error) {
	d, err := c.GetDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	return d.State, nil
}