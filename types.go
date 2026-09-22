package connectivity

import (
	"encoding/json"
	"strconv"
	"time"
)

// UnixTime is a timestamp carried on the wire as Unix seconds. It embeds
// time.Time, so all time.Time methods are available on it.
type UnixTime struct{ time.Time }

func (u *UnixTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	secs, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	u.Time = time.Unix(secs, 0).UTC()
	return nil
}

func (u UnixTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Unix())
}

// PagedResult is one page of a listing.
type PagedResult[T any] struct {
	Content       []T   `json:"content"`
	TotalElements int64 `json:"totalElements"`
	Page          int   `json:"page"`
	PageSize      int   `json:"pageSize"`
}

// Location is where a device is installed, in plain degrees.
//
// A device that has not been placed has no location at all, which is why it
// is a pointer everywhere it appears: 0, 0 is a real coordinate, and sending
// it for "unknown" puts the device on the map in the Gulf of Guinea.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// At returns a Location for the given degrees, so a caller with two floats
// does not need a named variable to take the address of.
func At(latitude, longitude float64) *Location {
	return &Location{Latitude: latitude, Longitude: longitude}
}

// Device is a connected hardware unit.
type Device struct {
	ID              string      `json:"id"`
	DeviceID        string      `json:"deviceId"`
	Name            string      `json:"name"`
	NetworkID       string      `json:"networkId"`
	ApplicationID   string      `json:"applicationId"`
	Bearer          Bearer      `json:"bearer"`
	DeviceClassID   string      `json:"deviceClassId"`
	DeviceClassName string      `json:"deviceClassName"`
	State           DeviceState `json:"status"`
	LastSeenAt      *UnixTime   `json:"lastSeenAt"`
	SignalDBm       *int        `json:"signalDbm"`
	AccessPointID   string      `json:"accessPointId"`
	CreatedAt       *UnixTime   `json:"createdAt"`
	// Description is what the device is for, as you set it.
	Description string `json:"description,omitempty"`
	// Location is where it is installed, absent until it is placed.
	Location *Location `json:"location,omitempty"`
}

// RegisterDeviceRequest creates a device on a network.
type RegisterDeviceRequest struct {
	NetworkID     string `json:"-"`
	DeviceID      string `json:"deviceId"`
	Name          string `json:"name"`
	DeviceClassID string `json:"deviceClassId,omitempty"`
	// Description is what the device is for; Location is where it is
	// installed. Both optional — a device registered from a warehouse has
	// neither until it is installed.
	Description string    `json:"description,omitempty"`
	Location    *Location `json:"location,omitempty"`
}

// UpdateDeviceRequest changes a device. Zero-valued fields are left unchanged.
// NetworkID moves the device to another network of the same bearer; a move
// needs a DeviceClassID that lives on the destination network.
type UpdateDeviceRequest struct {
	DeviceID      string `json:"-"`
	Name          string `json:"name,omitempty"`
	DeviceClassID string `json:"deviceClassId,omitempty"`
	NetworkID     string `json:"networkId,omitempty"`
	// A nil Location leaves the stored one alone, as an empty Name does: a
	// device is placed once and updated for many other reasons.
	Description string    `json:"description,omitempty"`
	Location    *Location `json:"location,omitempty"`
}

// ListDevicesRequest filters and pages a device listing. A zero value lists
// the first page of every device in the business; set NetworkID to scope to
// one network.
type ListDevicesRequest struct {
	NetworkID string
	Search    string
	Bearer    Bearer
	State     DeviceState
	Page      int
	PageSize  int
}

// AccessPoint is a unit of radio infrastructure on a low-power network.
type AccessPoint struct {
	ID                   string    `json:"id"`
	AccessPointID        string    `json:"accessPointId"`
	Name                 string    `json:"name"`
	NetworkID            string    `json:"networkId"`
	Online               bool      `json:"online"`
	LastSeenAt           *UnixTime `json:"lastSeenAt"`
	Latitude             float64   `json:"latitude"`
	Longitude            float64   `json:"longitude"`
	Devices              int64     `json:"devices"`
	DutyCycleHeadroomPct *float64  `json:"dutyCycleHeadroomPct"`
}

// AccessPointDetail is the full view of one access point.
type AccessPointDetail struct {
	AccessPointID      string            `json:"accessPointId"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Latitude           float64           `json:"latitude"`
	Longitude          float64           `json:"longitude"`
	Altitude           float64           `json:"altitude"`
	ReportEverySeconds int64             `json:"reportEverySeconds"`
	LastSeenAt         *UnixTime         `json:"lastSeenAt"`
	Online             bool              `json:"online"`
	Properties         map[string]string `json:"properties"`
}

// RegisterAccessPointRequest registers an access point on a low-power network.
type RegisterAccessPointRequest struct {
	NetworkID          string  `json:"-"`
	AccessPointID      string  `json:"accessPointId"`
	Name               string  `json:"name"`
	Description        string  `json:"description,omitempty"`
	Latitude           float64 `json:"latitude,omitempty"`
	Longitude          float64 `json:"longitude,omitempty"`
	ReportEverySeconds int64   `json:"reportEverySeconds,omitempty"`
}

// UpdateAccessPointRequest changes an access point's registration.
type UpdateAccessPointRequest struct {
	AccessPointID      string  `json:"-"`
	Name               string  `json:"name,omitempty"`
	Description        string  `json:"description,omitempty"`
	Latitude           float64 `json:"latitude,omitempty"`
	Longitude          float64 `json:"longitude,omitempty"`
	ReportEverySeconds int64   `json:"reportEverySeconds,omitempty"`
}

// Network is one deployment of connectivity for a business.
type Network struct {
	ID            string    `json:"id"`
	BusinessID    string    `json:"businessId"`
	Name          string    `json:"name"`
	Bearer        Bearer    `json:"bearer"`
	Region        string    `json:"region"`
	Status        string    `json:"status"`
	ApplicationID string    `json:"applicationId"`
	Devices       int64     `json:"devices"`
	DevicesOnline int64     `json:"devicesOnline"`
	AccessPoints  *int64    `json:"accessPoints,omitempty"`
	CreatedAt     *UnixTime `json:"createdAt"`
}

// CreateNetworkRequest creates a network.
type CreateNetworkRequest struct {
	Name   string `json:"name"`
	Bearer Bearer `json:"bearer"`
	Region string `json:"region,omitempty"`
}

// UpdateNetworkRequest renames a network.
type UpdateNetworkRequest struct {
	NetworkID string `json:"-"`
	Name      string `json:"name,omitempty"`
}

// CommandDeliveryMode is when a class's devices can receive a command.
type CommandDeliveryMode string

const (
	// DeliveryOnSend delivers when the device next transmits.
	DeliveryOnSend CommandDeliveryMode = "on-send"
	// DeliveryScheduled delivers in the device's scheduled receive slots.
	DeliveryScheduled CommandDeliveryMode = "scheduled"
	// DeliveryAlways delivers immediately; the device always listens.
	DeliveryAlways CommandDeliveryMode = "always"
)

// DeviceClass is a device type template.
type DeviceClass struct {
	ID                     string              `json:"id"`
	Name                   string              `json:"name"`
	Region                 string              `json:"region"`
	MessageIntervalSeconds int64               `json:"messageIntervalSeconds"`
	Devices                int64               `json:"devices"`
	CommandDelivery        CommandDeliveryMode `json:"commandDelivery"`
	CreatedAt              *UnixTime           `json:"createdAt"`
}

// CreateDeviceClassRequest creates a device class on a network.
type CreateDeviceClassRequest struct {
	NetworkID              string              `json:"-"`
	Name                   string              `json:"name"`
	Region                 string              `json:"region,omitempty"`
	MessageIntervalSeconds int64               `json:"messageIntervalSeconds,omitempty"`
	CommandDelivery        CommandDeliveryMode `json:"commandDelivery,omitempty"`
}

// UpdateDeviceClassRequest changes a device class. Region is fixed at
// creation: it is the regulatory domain the devices already operate in.
type UpdateDeviceClassRequest struct {
	NetworkID              string              `json:"-"`
	ClassID                string              `json:"-"`
	Name                   string              `json:"name"`
	MessageIntervalSeconds int64               `json:"messageIntervalSeconds,omitempty"`
	CommandDelivery        CommandDeliveryMode `json:"commandDelivery,omitempty"`
}

// Application is a fleet viewed by purpose. It is 1:1 with its network.
type Application struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	NetworkID    string `json:"networkId"`
	NetworkName  string `json:"networkName"`
	Bearer       Bearer `json:"bearer"`
	Devices      int64  `json:"devices"`
	Integrations int64  `json:"integrations"`
}

// CommandDelivery explains when a command will reach its device.
type CommandDelivery struct {
	Immediate             bool   `json:"immediate"`
	DeliversOnNextContact bool   `json:"deliversOnNextContact"`
	Explanation           string `json:"explanation"`
}

// Command is a payload queued for delivery to a device. Payloads are opaque:
// the platform never interprets them.
type Command struct {
	ID        string          `json:"id"`
	DeviceID  string          `json:"deviceId"`
	State     string          `json:"state"`
	Confirmed bool            `json:"confirmed"`
	QueuedAt  *UnixTime       `json:"queuedAt"`
	ExpiresAt *UnixTime       `json:"expiresAt"`
	Reason    string          `json:"reason"`
	Reference string          `json:"reference,omitempty"`
	Delivery  CommandDelivery `json:"delivery"`
}

// CommandOptions tunes a SendCommand call.
type CommandOptions struct {
	// Port routes the command on devices that multiplex several functions.
	// Zero uses the platform default.
	Port int
	// Confirmed asks the device to acknowledge receipt.
	Confirmed bool
	// Reference is your own id for the command, at most 120 characters. It
	// is echoed on the pending list and on every command.* event, so a
	// consumer can match command.sent to its own record without a lookup.
	Reference string
	// ExpiresIn retires a cellular command that has not gone out by then,
	// with a command.failed event (60s to 30 days). Zero waits for the
	// device however long it takes.
	ExpiresIn time.Duration
}

// DeviceKey is a named credential reference. The platform stores a reference
// and a fingerprint, never key material.
type DeviceKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Algorithm   string    `json:"algorithm"`
	Fingerprint string    `json:"fingerprint"`
	Revoked     bool      `json:"revoked"`
	CreatedAt   *UnixTime `json:"createdAt"`
}

// ActivityEvent is one entry of a device's connection history.
type ActivityEvent struct {
	At     *UnixTime `json:"at"`
	Event  string    `json:"event"`
	Detail string    `json:"detail"`
}

// InspectRequest asks the customer's own key service to decode a payload.
type InspectRequest struct {
	DeviceID   string `json:"-"`
	PayloadHex string `json:"payloadHex"`
	KeyID      string `json:"keyId,omitempty"`
}

// InspectResult is a decoded payload. IdentityMatches false means the payload
// decodes to a different device than the one it was attributed to — every
// reading attributed to that device is then suspect.
type InspectResult struct {
	DeviceID         string          `json:"deviceId"`
	ReportedDeviceID string          `json:"reportedDeviceId"`
	IdentityMatches  bool            `json:"identityMatches"`
	KeyName          string          `json:"keyName"`
	Decoded          json.RawMessage `json:"decoded"`
}

// UsageMetric is one billed quantity in a usage summary. Both values are
// decimal strings, never floats — a quantity can be fractional (megabytes) and
// money must not round.
type UsageMetric struct {
	Quantity string `json:"quantity"`
	Amount   string `json:"amount,omitempty"`
}

// UsageSummary is the current billing period to date. Money values are
// strings, never floats.
type UsageSummary struct {
	PeriodStart    *UnixTime              `json:"periodStart"`
	PeriodEnd      *UnixTime              `json:"periodEnd"`
	ByMetric       map[string]UsageMetric `json:"byMetric"`
	BaseFee        string                 `json:"baseFee"`
	EstimatedTotal string                 `json:"estimatedTotal"`
	Currency       string                 `json:"currency"`
	Unrated        []string               `json:"unrated"`
	Capped         []string               `json:"capped"`
}

// LogEntry is one line of a device's transport log.
type LogEntry struct {
	ID         string            `json:"id,omitempty"`
	At         int64             `json:"at"`
	Kind       string            `json:"kind"`
	Summary    string            `json:"summary"`
	Body       string            `json:"body,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// LogPage is one page of a device's transport log.
//
// Sampled says the page is a live sample rather than stored history — the
// low-power bearer keeps none. An empty sampled page means nothing arrived
// while we listened, NOT that the device has never spoken; a reader that
// cannot tell those apart shows a talking device as silent.
type LogPage struct {
	Entries       []LogEntry `json:"entries"`
	HasMore       bool       `json:"hasMore"`
	Sampled       bool       `json:"sampled,omitempty"`
	WindowSeconds int        `json:"windowSeconds,omitempty"`
}
