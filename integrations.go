package connectivity

import (
	"context"
	"net/http"
)

// ApiKey is a network's API key record. Token is set exactly once, in the
// answer to CreateApiKey; only its hash is stored, so a caller that does not
// keep it there can never recover it. Prefix identifies the key afterwards.
type ApiKey struct {
	ID         string `json:"id"`
	NetworkID  string `json:"networkId"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	Token      string `json:"token,omitempty"`
	LastUsedAt *int64 `json:"lastUsedAt"`
	CreatedAt  int64  `json:"createdAt"`
}

// CreateApiKey issues a key for one network. The returned Token is shown once
// and cannot be fetched again — store it before you discard the response.
func (c *Client) CreateApiKey(ctx context.Context, networkID, name string) (*ApiKey, error) {
	var out ApiKey
	body := map[string]string{"name": name}
	if err := c.do(ctx, http.MethodPost, "/networks/"+esc(networkID)+"/api-keys", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListApiKeys lists a network's live keys. Revoked keys are not listed, and
// Token is never populated here.
func (c *Client) ListApiKeys(ctx context.Context, networkID string) ([]ApiKey, error) {
	var out []ApiKey
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/api-keys", nil, nil, &out)
	return out, err
}

// RevokeApiKey revokes a key. Revoking is idempotent: an unknown or
// already-revoked id succeeds.
func (c *Client) RevokeApiKey(ctx context.Context, networkID, keyID string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+esc(networkID)+"/api-keys/"+esc(keyID), nil, nil, nil)
}

// IntegrationKind is where a network's messages are delivered.
type IntegrationKind string

const (
	// IntegrationHTTP posts each message to a URL of yours.
	IntegrationHTTP IntegrationKind = "http"
	// IntegrationQueue publishes each message onto a broker topic.
	IntegrationQueue IntegrationKind = "pulsar"
)

// Integration is one destination for a network's messages.
//
// Config never carries "authSecret" back: it is redacted on every read, and on
// the write that sets it.
type Integration struct {
	ID             string          `json:"id"`
	ApplicationID  string          `json:"applicationId"`
	Kind           IntegrationKind `json:"kind"`
	Enabled        bool            `json:"enabled"`
	Config         map[string]any  `json:"config"`
	LastDeliveryAt *int64          `json:"lastDeliveryAt"`
	LastError      string          `json:"lastError,omitempty"`
}

// HTTPConfig is the configuration of an IntegrationHTTP destination.
type HTTPConfig struct {
	// URL receives a POST per message.
	URL string
	// AuthHeader names the header carrying AuthSecret. Setting it makes
	// AuthSecret required.
	AuthHeader string
	// AuthSecret is write-only: it is never read back.
	AuthSecret string
}

// Config renders the destination as the API expects it.
func (h HTTPConfig) Config() map[string]any {
	cfg := map[string]any{"url": h.URL}
	if h.AuthHeader != "" {
		cfg["authHeader"] = h.AuthHeader
	}
	if h.AuthSecret != "" {
		cfg["authSecret"] = h.AuthSecret
	}
	return cfg
}

// QueueConfig is the configuration of an IntegrationQueue destination.
type QueueConfig struct {
	BrokerURL string
	Topic     string
}

// Config renders the destination as the API expects it.
func (q QueueConfig) Config() map[string]any {
	return map[string]any{"brokerUrl": q.BrokerURL, "topic": q.Topic}
}

// CreateIntegration adds a destination to a network. It is enabled on
// creation. A network may hold several.
func (c *Client) CreateIntegration(ctx context.Context, networkID string, kind IntegrationKind, config map[string]any) (*Integration, error) {
	var out Integration
	body := map[string]any{"kind": kind, "config": config}
	if err := c.do(ctx, http.MethodPost, "/networks/"+esc(networkID)+"/integrations", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListIntegrations lists a network's destinations.
func (c *Client) ListIntegrations(ctx context.Context, networkID string) ([]Integration, error) {
	var out []Integration
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/integrations", nil, nil, &out)
	return out, err
}

// UpdateIntegrationRequest changes a destination. A nil field is left alone.
//
// Config REPLACES the stored configuration rather than merging into it, with
// one exception: leaving "authSecret" out keeps the secret already stored, so
// a read-modify-write does not erase what the read redacted. Send an explicit
// "authSecret" to rotate it. The kind cannot be changed.
type UpdateIntegrationRequest struct {
	Enabled *bool          `json:"enabled,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

// UpdateIntegration changes a destination's configuration or enablement.
func (c *Client) UpdateIntegration(ctx context.Context, networkID, integrationID string, req UpdateIntegrationRequest) (*Integration, error) {
	var out Integration
	path := "/networks/" + esc(networkID) + "/integrations/" + esc(integrationID)
	if err := c.do(ctx, http.MethodPut, path, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteIntegration removes a destination. Idempotent: an unknown id succeeds.
func (c *Client) DeleteIntegration(ctx context.Context, networkID, integrationID string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+esc(networkID)+"/integrations/"+esc(integrationID), nil, nil, nil)
}

// IntegrationTest is the outcome of one synthetic delivery.
//
// A refused delivery is a SUCCESSFUL call: err is nil and Success is false.
// Branch on Success, not on err. StatusCode is absent when there was no HTTP
// answer at all — a transport failure, or a queue destination.
type IntegrationTest struct {
	Kind       IntegrationKind `json:"kind"`
	Success    bool            `json:"success"`
	StatusCode int             `json:"statusCode,omitempty"`
	LatencyMs  int64           `json:"latencyMs"`
	Error      string          `json:"error,omitempty"`
}

// TestIntegration sends one synthetic message through a destination, by the
// same path production traffic takes. The attempt is recorded in the delivery
// log, marked synthetic.
func (c *Client) TestIntegration(ctx context.Context, networkID, integrationID string) (*IntegrationTest, error) {
	var out IntegrationTest
	path := "/networks/" + esc(networkID) + "/integrations/" + esc(integrationID) + "/test"
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delivery is one attempt to hand a message to a destination.
//
// Retryable false means the stored payload is gone (or predates payload
// storage) and RetryDelivery will refuse: gate a retry on this flag.
type Delivery struct {
	ID            string `json:"id"`
	IntegrationID string `json:"integrationId"`
	DeviceID      string `json:"deviceId,omitempty"`
	At            int64  `json:"at"`
	Success       bool   `json:"success"`
	HTTPStatus    *int   `json:"httpStatus,omitempty"`
	Error         string `json:"error,omitempty"`
	DurationMs    int64  `json:"durationMs"`
	Synthetic     bool   `json:"synthetic"`
	Retryable     bool   `json:"retryable"`
}

// Deliveries returns one page of a destination's delivery history, newest
// first.
func (c *Client) Deliveries(ctx context.Context, networkID, integrationID string, page, pageSize int) (*PagedResult[Delivery], error) {
	var out PagedResult[Delivery]
	path := "/networks/" + esc(networkID) + "/integrations/" + esc(integrationID) + "/deliveries"
	if err := c.do(ctx, http.MethodGet, path, pageQuery(page, pageSize), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RetryDelivery replays a delivery's exact original bytes against the
// destination's CURRENT configuration, and records a new attempt.
//
// The returned Delivery is the new attempt. As with TestIntegration, a refused
// delivery is a successful call: branch on Success. A delivery with no stored
// payload is refused with a 400.
func (c *Client) RetryDelivery(ctx context.Context, networkID, integrationID, deliveryID string) (*Delivery, error) {
	var out Delivery
	path := "/networks/" + esc(networkID) + "/integrations/" + esc(integrationID) +
		"/deliveries/" + esc(deliveryID) + "/retry"
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EventRuleKind is what an event rule watches for.
type EventRuleKind string

// EventRuleQuiet fires when a device has not been heard from for longer than
// it should have been. It is the only kind today.
const EventRuleQuiet EventRuleKind = "quiet"

// EventRule is a standing watch on a network's devices.
type EventRule struct {
	ID                string        `json:"id"`
	NetworkID         string        `json:"networkId"`
	Name              string        `json:"name"`
	Kind              EventRuleKind `json:"kind"`
	Enabled           bool          `json:"enabled"`
	QuietAfterSeconds int           `json:"quietAfterSeconds"`
	DeviceClassID     *string       `json:"deviceClassId,omitempty"`
}

// CreateEventRuleRequest adds a watch to a network.
type CreateEventRuleRequest struct {
	Name string        `json:"name"`
	Kind EventRuleKind `json:"kind"`
	// QuietAfterSeconds is 60..2592000. Zero judges each device against its own
	// class's message interval instead.
	QuietAfterSeconds int `json:"quietAfterSeconds,omitempty"`
	// DeviceClassID narrows the rule to one class. Nil watches the network.
	DeviceClassID *string `json:"deviceClassId,omitempty"`
}

// CreateEventRule adds a watch. It is enabled on creation, and there is no
// route to disable one — delete it instead.
func (c *Client) CreateEventRule(ctx context.Context, networkID string, req CreateEventRuleRequest) (*EventRule, error) {
	var out EventRule
	if err := c.do(ctx, http.MethodPost, "/networks/"+esc(networkID)+"/event-rules", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventRules lists a network's watches.
func (c *Client) ListEventRules(ctx context.Context, networkID string) ([]EventRule, error) {
	var out []EventRule
	err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/event-rules", nil, nil, &out)
	return out, err
}

// DeleteEventRule removes a watch. Idempotent: an unknown id succeeds.
func (c *Client) DeleteEventRule(ctx context.Context, networkID, ruleID string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+esc(networkID)+"/event-rules/"+esc(ruleID), nil, nil, nil)
}
