package connectivity

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// RegisterAccessPoint registers radio infrastructure on a low-power network.
// Refused on the cellular bearer, which has no customer radio estate.
func (c *Client) RegisterAccessPoint(ctx context.Context, req RegisterAccessPointRequest) (*AccessPoint, error) {
	if req.NetworkID == "" {
		return nil, fmt.Errorf("connectivity: RegisterAccessPoint requires NetworkID")
	}
	var out AccessPoint
	err := c.do(ctx, http.MethodPost, "/networks/"+esc(req.NetworkID)+"/access-points", nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAccessPoints lists a network's access points.
func (c *Client) ListAccessPoints(ctx context.Context, networkID string) ([]AccessPoint, error) {
	var out []AccessPoint
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/access-points", nil, nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAccessPoint returns the full view of one access point.
func (c *Client) GetAccessPoint(ctx context.Context, accessPointID string) (*AccessPointDetail, error) {
	var out AccessPointDetail
	if err := c.do(ctx, http.MethodGet, "/access-points/"+esc(accessPointID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAccessPoint changes an access point's registration (name,
// description, location, report interval).
func (c *Client) UpdateAccessPoint(ctx context.Context, req UpdateAccessPointRequest) (*AccessPointDetail, error) {
	if req.AccessPointID == "" {
		return nil, fmt.Errorf("connectivity: UpdateAccessPoint requires AccessPointID")
	}
	var out AccessPointDetail
	err := c.do(ctx, http.MethodPut, "/access-points/"+esc(req.AccessPointID), nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAccessPoint removes an access point's registration.
func (c *Client) DeleteAccessPoint(ctx context.Context, accessPointID string) error {
	return c.do(ctx, http.MethodDelete, "/access-points/"+esc(accessPointID), nil, nil, nil)
}

// GetAccessPointRadioMetrics returns the access point's radio series
// (window "24h", "31d" or "1y"). Raw chart-oriented JSON.
func (c *Client) GetAccessPointRadioMetrics(ctx context.Context, accessPointID, window string) (map[string]any, error) {
	q := url.Values{}
	if window != "" {
		q.Set("window", window)
	}
	var out map[string]any
	if err := c.do(ctx, http.MethodGet, "/access-points/"+esc(accessPointID)+"/radio-metrics", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
