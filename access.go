package connectivity

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// FramePage is one page of a cellular device's wire traffic, newest first.
// HasMore says another page is behind it: ask again with Before set to the
// oldest At you received.
type FramePage struct {
	Entries []Frame `json:"entries"`
	HasMore bool    `json:"hasMore"`
}

// FrameQuery pages a frame listing. A zero value asks for the newest frames.
type FrameQuery struct {
	Limit  int
	Before time.Time
}

func (q FrameQuery) values() url.Values {
	v := url.Values{}
	if q.Limit > 0 {
		v.Set("limit", fmt.Sprint(q.Limit))
	}
	if !q.Before.IsZero() {
		v.Set("before", fmt.Sprint(q.Before.Unix()))
	}
	return v
}

// The cellular gate, for an operator token. A network API key reaches the same
// gate through PlatformClient; these are the console-side routes, so one
// principal can provision and key a device without also holding a network key.

// DeviceAccess returns a cellular device's gate: whether its traffic goes
// anywhere, and what is missing if it does not.
func (c *Client) DeviceAccess(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/access", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetDeviceKey sets or replaces a cellular device's key, 32 hexadecimal
// characters. There is no read-back.
func (c *Client) SetDeviceKey(ctx context.Context, deviceID, keyHex string) (*DeviceAccess, error) {
	var out DeviceAccess
	body := map[string]string{"key": keyHex}
	if err := c.do(ctx, http.MethodPut, "/devices/"+esc(deviceID)+"/key", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ClearDeviceKey removes the key and disables the device with it.
func (c *Client) ClearDeviceKey(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := c.do(ctx, http.MethodDelete, "/devices/"+esc(deviceID)+"/key", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EnableDevice lets a cellular device's traffic through. A refusal is a
// conflict (IsConflict) whose message names what is missing; the same reasons
// are listed in DeviceAccess.Blockers.
func (c *Client) EnableDevice(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := c.do(ctx, http.MethodPost, "/devices/"+esc(deviceID)+"/enable", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DisableDevice stops a cellular device's traffic at the gate. The key is kept.
func (c *Client) DisableDevice(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := c.do(ctx, http.MethodPost, "/devices/"+esc(deviceID)+"/disable", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeviceFrames returns a page of a cellular device's wire traffic. Keepalives
// are sampled; every reading and every command is listed.
func (c *Client) DeviceFrames(ctx context.Context, deviceID string, q FrameQuery) (*FramePage, error) {
	var out FramePage
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/frames", q.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeviceLogs returns a page of a device's transport log — what the connection
// itself did, as opposed to what it carried.
func (c *Client) DeviceLogs(ctx context.Context, deviceID string, q FrameQuery) (*LogPage, error) {
	var out LogPage
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/logs", q.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
