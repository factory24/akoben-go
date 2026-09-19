package connectivity

import (
	"context"
	"net/http"
)

// Destination is a customer platform that receives this business's messages.
// Its auth secret is never returned — HasSecret says only whether one is held.
type Destination struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	AuthHeader string `json:"authHeader,omitempty"`
	HasSecret  bool   `json:"hasSecret"`
	Enabled    bool   `json:"enabled"`
	IsDefault  bool   `json:"isDefault"`
	CreatedAt  int64  `json:"createdAt"`
}

// CreateDestinationRequest registers a platform of yours.
type CreateDestinationRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	// AuthHeader names the header carrying AuthSecret on every delivery.
	AuthHeader string `json:"authHeader,omitempty"`
	// AuthSecret is write-only: it is never read back.
	AuthSecret string `json:"authSecret,omitempty"`
	IsDefault  bool   `json:"isDefault"`
}

// UpdateDestinationRequest changes a destination. An empty string leaves that
// field alone — including AuthSecret, which cannot be cleared this way, only
// replaced. Enabled and IsDefault are pointers so that false is distinct from
// absent.
type UpdateDestinationRequest struct {
	Name       string `json:"name,omitempty"`
	URL        string `json:"url,omitempty"`
	AuthHeader string `json:"authHeader,omitempty"`
	AuthSecret string `json:"authSecret,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	IsDefault  *bool  `json:"isDefault,omitempty"`
}

// ListDestinations lists the business's platforms.
func (c *Client) ListDestinations(ctx context.Context) ([]Destination, error) {
	var out []Destination
	err := c.do(ctx, http.MethodGet, "/platforms", nil, nil, &out)
	return out, err
}

// CreateDestination registers a platform. It is enabled on creation, and the
// first one a business registers becomes the default whatever IsDefault says.
func (c *Client) CreateDestination(ctx context.Context, req CreateDestinationRequest) (*Destination, error) {
	var out Destination
	if err := c.do(ctx, http.MethodPost, "/platforms", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDestination changes a registered platform.
func (c *Client) UpdateDestination(ctx context.Context, destinationID string, req UpdateDestinationRequest) (*Destination, error) {
	var out Destination
	if err := c.do(ctx, http.MethodPut, "/platforms/"+esc(destinationID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDestination removes a registered platform.
func (c *Client) DeleteDestination(ctx context.Context, destinationID string) error {
	return c.do(ctx, http.MethodDelete, "/platforms/"+esc(destinationID), nil, nil, nil)
}

// DeviceAlert is one firing of an event rule against a device.
type DeviceAlert struct {
	ID         string `json:"id"`
	DeviceID   string `json:"deviceId"`
	RuleName   string `json:"ruleName"`
	Open       bool   `json:"open"`
	OpenedAt   int64  `json:"openedAt"`
	ResolvedAt *int64 `json:"resolvedAt,omitempty"`
	LastSeenAt *int64 `json:"lastSeenAt,omitempty"`
}

// NetworkAlerts returns the alerts raised on a network's devices.
func (c *Client) NetworkAlerts(ctx context.Context, networkID string) ([]DeviceAlert, error) {
	var out []DeviceAlert
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/alerts", nil, nil, &out)
	return out, err
}

// GetNetworkApplication returns a network's application.
//
// NetworkName and Bearer are not populated on this route — the server sends
// them empty. Read them from the network itself.
func (c *Client) GetNetworkApplication(ctx context.Context, networkID string) (*Application, error) {
	var out Application
	if err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/application", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
