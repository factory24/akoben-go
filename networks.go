package connectivity

import (
	"context"
	"fmt"
	"net/http"
)

// CreateNetwork creates a network for the business.
func (c *Client) CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*Network, error) {
	var out Network
	if err := c.do(ctx, http.MethodPost, "/networks", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListNetworks lists the business's networks.
func (c *Client) ListNetworks(ctx context.Context, page, pageSize int) (*PagedResult[Network], error) {
	var out PagedResult[Network]
	if err := c.do(ctx, http.MethodGet, "/networks", pageQuery(page, pageSize), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetNetwork fetches one network.
func (c *Client) GetNetwork(ctx context.Context, networkID string) (*Network, error) {
	var out Network
	if err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateNetwork renames a network.
func (c *Client) UpdateNetwork(ctx context.Context, req UpdateNetworkRequest) (*Network, error) {
	if req.NetworkID == "" {
		return nil, fmt.Errorf("connectivity: UpdateNetwork requires NetworkID")
	}
	var out Network
	err := c.do(ctx, http.MethodPut, "/networks/"+esc(req.NetworkID), nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteNetwork removes a network and its registrations.
func (c *Client) DeleteNetwork(ctx context.Context, networkID string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+esc(networkID), nil, nil, nil)
}

// CreateDeviceClass creates a device type template on a network.
func (c *Client) CreateDeviceClass(ctx context.Context, req CreateDeviceClassRequest) (*DeviceClass, error) {
	if req.NetworkID == "" {
		return nil, fmt.Errorf("connectivity: CreateDeviceClass requires NetworkID")
	}
	var out DeviceClass
	err := c.do(ctx, http.MethodPost, "/networks/"+esc(req.NetworkID)+"/device-classes", nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDeviceClasses lists a network's device classes.
func (c *Client) ListDeviceClasses(ctx context.Context, networkID string) ([]DeviceClass, error) {
	var out []DeviceClass
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/device-classes", nil, nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateDeviceClass changes a device class's name or expected message
// interval. Region is fixed at creation.
func (c *Client) UpdateDeviceClass(ctx context.Context, req UpdateDeviceClassRequest) (*DeviceClass, error) {
	if req.NetworkID == "" || req.ClassID == "" {
		return nil, fmt.Errorf("connectivity: UpdateDeviceClass requires NetworkID and ClassID")
	}
	var out DeviceClass
	err := c.do(ctx, http.MethodPut, "/networks/"+esc(req.NetworkID)+"/device-classes/"+esc(req.ClassID), nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDeviceClass removes a device class. Refused while devices use it.
func (c *Client) DeleteDeviceClass(ctx context.Context, networkID, classID string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+esc(networkID)+"/device-classes/"+esc(classID), nil, nil, nil)
}

// ListApplications lists every application in the business.
func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	var out []Application
	if err := c.do(ctx, http.MethodGet, "/applications", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListApplicationDevices returns one page of an application's devices.
func (c *Client) ListApplicationDevices(ctx context.Context, applicationID string, page, pageSize int) (*PagedResult[Device], error) {
	var out PagedResult[Device]
	err := c.do(ctx, http.MethodGet, "/applications/"+esc(applicationID)+"/devices", pageQuery(page, pageSize), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUsage returns the business's usage for the current billing period.
func (c *Client) GetUsage(ctx context.Context) (*UsageSummary, error) {
	var out UsageSummary
	if err := c.do(ctx, http.MethodGet, "/billing/usage", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
