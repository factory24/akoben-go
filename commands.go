package connectivity

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// SendCommand queues an opaque payload for delivery to a device and returns
// the command id. The platform never interprets the bytes. The id is stable:
// it names the same queued item in ListPendingCommands until delivery or
// ClearCommands.
func (c *Client) SendCommand(ctx context.Context, deviceID string, payload []byte, opts CommandOptions) (string, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("connectivity: SendCommand requires a payload")
	}
	body := map[string]any{
		"payloadHex": hex.EncodeToString(payload),
		"confirmed":  opts.Confirmed,
	}
	if opts.Port > 0 {
		body["port"] = opts.Port
	}
	if opts.Reference != "" {
		body["reference"] = opts.Reference
	}
	if opts.ExpiresIn > 0 {
		body["expiresInSeconds"] = int(opts.ExpiresIn / time.Second)
	}
	var out struct {
		CommandID string `json:"commandId"`
	}
	err := c.do(ctx, http.MethodPost, "/devices/"+esc(deviceID)+"/commands", nil, body, &out)
	if err != nil {
		return "", err
	}
	return out.CommandID, nil
}

// ListPendingCommands returns the device's undelivered commands.
func (c *Client) ListPendingCommands(ctx context.Context, deviceID string) ([]Command, error) {
	var out []Command
	if err := c.do(ctx, http.MethodGet, "/devices/"+esc(deviceID)+"/commands", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClearCommands retracts the device's undelivered commands and returns how
// many were retracted. A command already dispatched to the bearer survives a
// clear by design.
func (c *Client) ClearCommands(ctx context.Context, deviceID string) (int64, error) {
	var out struct {
		Retracted int64 `json:"retracted"`
	}
	err := c.do(ctx, http.MethodDelete, "/devices/"+esc(deviceID)+"/commands", nil, nil, &out)
	if err != nil {
		return 0, err
	}
	return out.Retracted, nil
}
