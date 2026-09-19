package connectivity

import (
	"context"
	"net/http"
)

// Money is always a decimal string on this API, never a float.

// Plan is one published plan on the price list.
type Plan struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceMonthly string `json:"priceMonthly"`
	Currency     string `json:"currency"`
	// Included maps a metric name to the allowance included in the plan.
	Included map[string]string `json:"included"`
	Features []string          `json:"features"`
}

// Subscription is the business's current plan.
type Subscription struct {
	ID        string `json:"id"`
	Plan      *Plan  `json:"plan,omitempty"`
	Status    string `json:"status"`
	StartedAt int64  `json:"startedAt"`
	RenewsAt  int64  `json:"renewsAt,omitempty"`
}

// InvoiceLine is one charge on an invoice.
type InvoiceLine struct {
	Description string `json:"description"`
	Metric      string `json:"metric,omitempty"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unitPrice"`
	Amount      string `json:"amount"`
}

// Invoice is a whole invoice, lines included: the listing and the document a
// customer downloads are the same record, so they cannot disagree.
type Invoice struct {
	ID          string        `json:"id"`
	Number      string        `json:"number"`
	Status      string        `json:"status"`
	BilledTo    string        `json:"billedTo,omitempty"`
	PeriodStart int64         `json:"periodStart"`
	PeriodEnd   int64         `json:"periodEnd"`
	Lines       []InvoiceLine `json:"lines"`
	Subtotal    string        `json:"subtotal"`
	Tax         string        `json:"tax"`
	Total       string        `json:"total"`
	Currency    string        `json:"currency"`
	IssuedAt    *int64        `json:"issuedAt,omitempty"`
	DueAt       *int64        `json:"dueAt,omitempty"`
	PaidAt      *int64        `json:"paidAt,omitempty"`
}

// CheckoutMethod is one way the business can pay.
type CheckoutMethod struct {
	Provider    string `json:"provider"`
	CheckoutURL string `json:"checkoutUrl,omitempty"`
}

// ListPlans returns the published price list. It is not scoped to a business.
func (c *Client) ListPlans(ctx context.Context) ([]Plan, error) {
	var out []Plan
	err := c.do(ctx, http.MethodGet, "/billing/plans", nil, nil, &out)
	return out, err
}

// GetSubscription returns the business's current subscription, or nil when it
// has none — no subscription is an answer, not an error.
func (c *Client) GetSubscription(ctx context.Context) (*Subscription, error) {
	var out *Subscription
	if err := c.do(ctx, http.MethodGet, "/billing/subscription", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListInvoices returns the business's invoices.
func (c *Client) ListInvoices(ctx context.Context) ([]Invoice, error) {
	var out []Invoice
	err := c.do(ctx, http.MethodGet, "/billing/invoices", nil, nil, &out)
	return out, err
}

// ListCheckoutMethods returns the enabled ways to pay.
func (c *Client) ListCheckoutMethods(ctx context.Context) ([]CheckoutMethod, error) {
	var out []CheckoutMethod
	err := c.do(ctx, http.MethodGet, "/billing/payment-methods", nil, nil, &out)
	return out, err
}
