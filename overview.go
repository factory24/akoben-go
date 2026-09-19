package connectivity

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// ActivityPoint is one bucket of a traffic series.
type ActivityPoint struct {
	At       int64 `json:"at"`
	Messages int64 `json:"messages"`
	Commands int64 `json:"commands"`
	Failures int64 `json:"failures"`
}

// NetworkActivity is one network's line in the business overview.
//
// MedianLatencyMs is declared but not measured — nothing on the ingest path
// records a per-message latency yet, so it is always 0. Do not render it as a
// figure; a confident 0 ms is worse than an absent number.
type NetworkActivity struct {
	NetworkID       string `json:"networkId"`
	Name            string `json:"name"`
	Bearer          Bearer `json:"bearer"`
	Devices         int64  `json:"devices"`
	DevicesOnline   int64  `json:"devicesOnline"`
	MessagesIn24h   int64  `json:"messagesIn24h"`
	CommandsIn24h   int64  `json:"commandsIn24h"`
	MedianLatencyMs int    `json:"medianLatencyMs"`
}

// Overview is the business at a glance, across every network.
type Overview struct {
	Networks            int64             `json:"networks"`
	Devices             int64             `json:"devices"`
	DevicesOnline       int64             `json:"devicesOnline"`
	AccessPoints        int64             `json:"accessPoints"`
	AccessPointsOnline  int64             `json:"accessPointsOnline"`
	MessagesIn24h       int64             `json:"messagesIn24h"`
	CommandsIn24h       int64             `json:"commandsIn24h"`
	CommandFailuresIn24 int64             `json:"commandFailuresIn24h"`
	MedianLatencyMs     int               `json:"medianLatencyMs"`
	ByNetwork           []NetworkActivity `json:"byNetwork"`
	Series              []ActivityPoint   `json:"series"`
}

// NetworkClassShare is one device class's share of a network, with the
// interval its devices are judged against so the number is explicable.
type NetworkClassShare struct {
	ClassID                string `json:"classId,omitempty"`
	Name                   string `json:"name"`
	MessageIntervalSeconds int    `json:"messageIntervalSeconds"`
	SilentAfterSeconds     int    `json:"silentAfterSeconds"`
	Devices                int64  `json:"devices"`
	Online                 int64  `json:"online"`
	Silent                 int64  `json:"silent"`
	NeverSeen              int64  `json:"neverSeen"`
}

// NetworkOverview is one network at a glance.
//
// ConnectedNow and Unidentified are absent — not zero — when the transport
// could not be reached: "we could not ask" and "nothing is connected" are
// different answers and only one of them is an incident.
type NetworkOverview struct {
	NetworkID      string              `json:"networkId"`
	Bearer         Bearer              `json:"bearer"`
	Devices        int64               `json:"devices"`
	Online         int64               `json:"online"`
	Silent         int64               `json:"silent"`
	NeverSeen      int64               `json:"neverSeen"`
	ConnectedNow   *int64              `json:"connectedNow,omitempty"`
	Unidentified   *int64              `json:"unidentified,omitempty"`
	WindowHours    int                 `json:"windowHours"`
	Messages       int64               `json:"messages"`
	BytesIn        int64               `json:"bytesIn"`
	Commands       int64               `json:"commands"`
	CommandsFailed int64               `json:"commandsFailed"`
	LastMessageAt  *int64              `json:"lastMessageAt,omitempty"`
	ByClass        []NetworkClassShare `json:"byClass"`
	Series         []ActivityPoint     `json:"series"`
}

// AuditEntry is one line of the audit trail: who did what, and when. The trail
// is read-only by design — there is no write route.
type AuditEntry struct {
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	ActorEmail string `json:"actorEmail"`
	Action     string `json:"action"`
	Target     string `json:"target"`
	Detail     string `json:"detail,omitempty"`
	OccurredAt int64  `json:"occurredAt"`
}

// GetOverview returns the business at a glance, across every network.
func (c *Client) GetOverview(ctx context.Context) (*Overview, error) {
	var out Overview
	if err := c.do(ctx, http.MethodGet, "/overview", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetNetworkOverview returns one network at a glance. The window is the
// server's; WindowHours on the result says which one was applied.
func (c *Client) GetNetworkOverview(ctx context.Context, networkID string) (*NetworkOverview, error) {
	var out NetworkOverview
	if err := c.do(ctx, http.MethodGet, "/networks/"+esc(networkID)+"/overview", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AuditQuery filters and pages the audit trail. From and To are inclusive;
// a zero time is unbounded on that side.
type AuditQuery struct {
	Action   string
	Actor    string
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

// GetAuditLog returns one page of the business's audit trail, newest first.
func (c *Client) GetAuditLog(ctx context.Context, req AuditQuery) (*PagedResult[AuditEntry], error) {
	q := pageQuery(req.Page, req.PageSize)
	if req.Action != "" {
		q.Set("action", req.Action)
	}
	if req.Actor != "" {
		q.Set("actor", req.Actor)
	}
	if !req.From.IsZero() {
		q.Set("from", strconv.FormatInt(req.From.Unix(), 10))
	}
	if !req.To.IsZero() {
		q.Set("to", strconv.FormatInt(req.To.Unix(), 10))
	}
	var out PagedResult[AuditEntry]
	if err := c.do(ctx, http.MethodGet, "/audit", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
