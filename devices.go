package connectivity

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
)

// RegisterDevice creates a device on a network. The device identifier is
// validated against the network's bearer: 16 hexadecimal characters on
// lorwan, 10-20 digits on cellular.
func (c *Client) RegisterDevice(ctx context.Context, req RegisterDeviceRequest) (*Device, error) {
	if req.NetworkID == "" {
		return nil, fmt.Errorf("connectivity: RegisterDevice requires NetworkID")
	}
	var out Device
	err := c.do(ctx, http.MethodPost, "/networks/"+esc(req.NetworkID)+"/devices", nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDevice fetches one device by its device identifier (or platform row id).
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	var out Device
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDevice changes a device's name and/or class.
func (c *Client) UpdateDevice(ctx context.Context, req UpdateDeviceRequest) (*Device, error) {
	if req.DeviceID == "" {
		return nil, fmt.Errorf("connectivity: UpdateDevice requires DeviceID")
	}
	var out Device
	err := c.do(ctx, http.MethodPut, "/devices/"+esc(req.DeviceID), nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDevice removes a device.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodDelete, "/devices/"+esc(deviceID), nil, nil, nil)
}

// ListDevices returns one page of devices. Scope to a network by setting
// req.NetworkID; otherwise every device in the business is listed.
func (c *Client) ListDevices(ctx context.Context, req ListDevicesRequest) (*PagedResult[Device], error) {
	q := pageQuery(req.Page, req.PageSize)
	if req.Search != "" {
		q.Set("search", req.Search)
	}
	if req.Bearer != "" {
		q.Set("bearer", string(req.Bearer))
	}
	if req.State != "" {
		q.Set("status", string(req.State))
	}
	path := "/devices"
	if req.NetworkID != "" {
		path = "/networks/" + esc(req.NetworkID) + "/devices"
	}
	var out PagedResult[Device]
	if err := c.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetDeviceSecret sets the device's credential. Write-only: the secret goes
// to where the bearer holds it and is never readable back through this API.
// secret must be 32 hexadecimal characters. Refused on the cellular bearer,
// where credentials live in the customer's own key service.
func (c *Client) SetDeviceSecret(ctx context.Context, deviceID, secret string) error {
	if _, err := hex.DecodeString(secret); err != nil || len(secret) != 32 {
		return fmt.Errorf("connectivity: a device secret is 32 hexadecimal characters")
	}
	body := map[string]string{"secret": secret}
	return c.do(ctx, http.MethodPut, "/devices/"+esc(deviceID)+"/secret", nil, body, nil)
}

// GetDeviceActivity returns one page of the device's connection history,
// newest first.
func (c *Client) GetDeviceActivity(ctx context.Context, deviceID string, page, pageSize int) (*PagedResult[ActivityEvent], error) {
	var out PagedResult[ActivityEvent]
	q := pageQuery(page, pageSize)
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/activity", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InspectPayload forwards a payload and a key reference to the customer's own
// key service and returns what it decodes to. The platform holds no key
// material and never stores the plaintext.
func (c *Client) InspectPayload(ctx context.Context, req InspectRequest) (*InspectResult, error) {
	if req.DeviceID == "" {
		return nil, fmt.Errorf("connectivity: InspectPayload requires DeviceID")
	}
	var out InspectResult
	err := c.do(ctx, http.MethodPost, "/devices/"+esc(req.DeviceID)+"/inspect", nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateDeviceKey records a named key reference (never material) for a device.
func (c *Client) CreateDeviceKey(ctx context.Context, deviceID, name, algorithm, reference string) (*DeviceKey, error) {
	body := map[string]string{"name": name, "algorithm": algorithm, "reference": reference}
	var out DeviceKey
	err := c.do(ctx, http.MethodPost, "/devices/"+esc(deviceID)+"/keys", nil, body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDeviceKeys lists a device's named key references.
func (c *Client) ListDeviceKeys(ctx context.Context, deviceID string) ([]DeviceKey, error) {
	var out []DeviceKey
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/keys", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RevokeDeviceKey revokes a named key reference. Revoked, never deleted.
func (c *Client) RevokeDeviceKey(ctx context.Context, deviceID, keyID string) error {
	return c.do(ctx, http.MethodDelete, "/devices/"+esc(deviceID)+"/keys/"+esc(keyID), nil, nil, nil)
}

// GetDeviceMetrics returns the device's traffic/signal series for a window
// such as "24h", "7d" or "30d". The result shape is chart-oriented JSON owned
// by the caller's rendering; it is returned raw.
func (c *Client) GetDeviceMetrics(ctx context.Context, deviceID, window string) (map[string]any, error) {
	q := url.Values{}
	if window != "" {
		q.Set("window", window)
	}
	var out map[string]any
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/metrics", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDeviceRadioMetrics returns the low-power radio series (window "24h",
// "31d" or "1y"). Refused on cellular devices, which have no radio estate.
func (c *Client) GetDeviceRadioMetrics(ctx context.Context, deviceID, window string) (map[string]any, error) {
	q := url.Values{}
	if window != "" {
		q.Set("window", window)
	}
	var out map[string]any
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/radio-metrics", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}