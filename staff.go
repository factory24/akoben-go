package connectivity

import (
	"context"
	"net/http"
)

// Everything in this file needs a platform-staff principal — an operator of
// the network, not a customer of it. A customer token is refused with 403, so
// these are here for our own tooling, not for integration.

// UnclaimedDevice is hardware seen on the network that belongs to no business
// yet. The listing is deliberately unscoped: it is what staff claim devices
// from. Busiest first.
type UnclaimedDevice struct {
	DeviceID      string `json:"deviceId"`
	NetworkID     string `json:"networkId,omitempty"`
	ProvisionedOn string `json:"provisionedOn,omitempty"`
	LastSeenAt    int64  `json:"lastSeenAt"`
	Messages      int64  `json:"messages"`
	Commands      int64  `json:"commands"`
	BytesIn       int64  `json:"bytesIn"`
	BytesOut      int64  `json:"bytesOut"`
}

// ListUnclaimedDevices lists hardware that has reported but has no owner.
// Platform staff only.
func (c *Client) ListUnclaimedDevices(ctx context.Context) ([]UnclaimedDevice, error) {
	var out []UnclaimedDevice
	err := c.do(ctx, http.MethodGet, "/devices/unclaimed", nil, nil, &out)
	return out, err
}

// StaffPlan is a plan as staff see it: the overage rates and visibility a
// customer's Plan leaves out, and how many businesses are on it.
type StaffPlan struct {
	ID           string            `json:"id"`
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	PriceMonthly string            `json:"priceMonthly"`
	Currency     string            `json:"currency"`
	Included     map[string]string `json:"included"`
	Overage      map[string]string `json:"overage"`
	Features     []string          `json:"features"`
	IsPublic     bool              `json:"isPublic"`
	SortOrder    int               `json:"sortOrder"`
	Subscribers  int64             `json:"subscribers"`
}

// StaffPlanRequest creates or replaces a plan. PriceMonthly is a decimal
// string and Currency an ISO 4217 code — never assume one.
type StaffPlanRequest struct {
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	PriceMonthly string            `json:"priceMonthly"`
	Currency     string            `json:"currency"`
	Included     map[string]string `json:"included,omitempty"`
	Overage      map[string]string `json:"overage,omitempty"`
	Features     []string          `json:"features,omitempty"`
	IsPublic     bool              `json:"isPublic"`
	SortOrder    int               `json:"sortOrder"`
}

// StaffSubscriber is one business's standing on the price list.
type StaffSubscriber struct {
	BusinessID   string `json:"businessId"`
	BusinessName string `json:"businessName"`
	PlanID       string `json:"planId,omitempty"`
	PlanName     string `json:"planName,omitempty"`
	Status       string `json:"status"`
	PriceMonthly string `json:"priceMonthly,omitempty"`
	Currency     string `json:"currency,omitempty"`
	StartedAt    int64  `json:"startedAt,omitempty"`
	RenewsAt     int64  `json:"renewsAt,omitempty"`
	Devices      int64  `json:"devices"`
}

// StaffBilling is the book of business. Monthly totals are per currency
// because subscribers are not all billed in one.
type StaffBilling struct {
	Businesses        int               `json:"businesses"`
	Subscribed        int               `json:"subscribed"`
	Paying            int               `json:"paying"`
	MonthlyByCurrency map[string]string `json:"monthlyByCurrency"`
	Subscribers       []StaffSubscriber `json:"subscribers"`
}

// StaffBillingSummary returns the book of business. Platform staff only.
func (c *Client) StaffBillingSummary(ctx context.Context) (*StaffBilling, error) {
	var out StaffBilling
	if err := c.do(ctx, http.MethodGet, "/admin/billing", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StaffListPlans lists every plan, published or not. Platform staff only.
func (c *Client) StaffListPlans(ctx context.Context) ([]StaffPlan, error) {
	var out []StaffPlan
	err := c.do(ctx, http.MethodGet, "/admin/plans", nil, nil, &out)
	return out, err
}

// StaffCreatePlan adds a plan to the price list. Platform staff only.
func (c *Client) StaffCreatePlan(ctx context.Context, req StaffPlanRequest) (*StaffPlan, error) {
	var out StaffPlan
	if err := c.do(ctx, http.MethodPost, "/admin/plans", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StaffUpdatePlan replaces a plan. Platform staff only.
func (c *Client) StaffUpdatePlan(ctx context.Context, planID string, req StaffPlanRequest) (*StaffPlan, error) {
	var out StaffPlan
	if err := c.do(ctx, http.MethodPut, "/admin/plans/"+esc(planID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StaffDeletePlan removes a plan. Refused with a conflict (IsConflict) while a
// subscription references it. Platform staff only.
func (c *Client) StaffDeletePlan(ctx context.Context, planID string) error {
	return c.do(ctx, http.MethodDelete, "/admin/plans/"+esc(planID), nil, nil, nil)
}

// StaffAssignPlan puts a business on a plan and returns the subscription.
// Platform staff only.
func (c *Client) StaffAssignPlan(ctx context.Context, businessID, planID string) (*Subscription, error) {
	var out Subscription
	body := map[string]string{"planId": planID}
	if err := c.do(ctx, http.MethodPut, "/admin/businesses/"+esc(businessID)+"/plan", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
