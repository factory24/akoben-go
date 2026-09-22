package connectivity

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PlatformClient manages devices on ONE network with that network's API key.
//
// This is the integration path for a customer's own backend: one key per
// network, created in the console and held in configuration. The key carries
// its business and its network, so nothing it calls can reach another. Devices
// are addressed by the identifier the customer already uses — the serial or
// the radio identifier — never by our internal id.
type PlatformClient struct {
	*Client
	networkID  string
	businessID string
}

// NewPlatform builds a client for the platform API. baseURL is the API root
// including /v1.
func NewPlatform(baseURL, apiKey string, opts ...Option) *PlatformClient {
	return &PlatformClient{Client: New(baseURL, StaticToken(apiKey), opts...)}
}

// NewPlatformFromEnv reads one network's integration from the environment.
// network names the key: "CELLULAR" reads AKOBEN.CELLULAR.API_KEY and
// AKOBEN.CELLULAR.NETWORK_ID.
//
//	AKOBEN.BASE_URL            API root, e.g. https://api.example.com/v1
//	AKOBEN.BUSINESS_ID         the tenant the key belongs to (checked, not sent)
//	AKOBEN.<NETWORK>.API_KEY   the network's API key
//	AKOBEN.<NETWORK>.NETWORK_ID the network the key is for (checked on first use)
//
// Underscored spellings (AKOBEN_CELLULAR_API_KEY, ...) are accepted too, and so
// is the old CONNECTIVITY prefix.
func NewPlatformFromEnv(network string) (*PlatformClient, error) {
	n := strings.ToUpper(network)
	baseURL := env("BASE_URL")
	key := env(n + ".API_KEY")
	if baseURL == "" || key == "" {
		return nil, fmt.Errorf("connectivity: set AKOBEN.BASE_URL and AKOBEN.%s.API_KEY", n)
	}
	p := NewPlatform(baseURL, key)
	p.networkID = env(n + ".NETWORK_ID")
	p.businessID = env("BUSINESS_ID")
	return p, nil
}

// PlatformNetwork is the network an API key is bound to.
type PlatformNetwork struct {
	ID             string `json:"id"`
	BusinessID     string `json:"businessId"`
	Name           string `json:"name"`
	Bearer         Bearer `json:"bearer"`
	DeviceEndpoint string `json:"deviceEndpoint,omitempty"`
	ListenPort     *int   `json:"listenPort,omitempty"`
}

// DeviceAccess is a cellular device's gate. On the low-power bearer it is
// always enabled with its key set, because that bearer has no gate of its own.
type DeviceAccess struct {
	Enabled    bool     `json:"enabled"`
	Discovered bool     `json:"discovered"`
	IMEI       string   `json:"imei,omitempty"`
	ICCID      string   `json:"iccid,omitempty"`
	Blockers   []string `json:"blockers"`
	Key        struct {
		Set         bool      `json:"set"`
		Fingerprint string    `json:"fingerprint,omitempty"`
		SetAt       *UnixTime `json:"setAt,omitempty"`
	} `json:"key"`
}

// PlatformDevice is a device as the platform API returns it: the device plus,
// on cellular, its gate.
type PlatformDevice struct {
	Device
	Access *DeviceAccess `json:"access,omitempty"`
}

// CreatePlatformDeviceRequest provisions a device. Key and Enabled are
// optional, so a device can be created, keyed and enabled in one call.
type CreatePlatformDeviceRequest struct {
	DeviceID      string `json:"deviceId"`
	Name          string `json:"name"`
	DeviceClassID string `json:"deviceClassId"`
	// Key is the device's AES-128 key, 32 hex characters. On cellular it is
	// checked against every frame; on the low-power bearer it is the device
	// secret it authenticates with. Stored as given, and returned by nothing.
	Key     string `json:"key,omitempty"`
	Enabled bool   `json:"enabled"`
	// Where the device is and what it is for — the same fields the operator
	// client's RegisterDevice takes.
	Description string    `json:"description,omitempty"`
	Location    *Location `json:"location,omitempty"`
}

// Frame is one frame on a cellular device's wire. Keepalives are sampled; every
// reading and every command is listed.
type Frame struct {
	At         int64  `json:"at"`
	Direction  string `json:"direction"`
	Kind       string `json:"kind"`
	PayloadHex string `json:"payloadHex"`
	SizeBytes  int    `json:"sizeBytes"`
	CommandID  string `json:"commandId,omitempty"`
	// Note says why a frame went no further, or that a reading we received was
	// never accepted by your platform.
	Note string `json:"note,omitempty"`
}

// Network returns the network this key is bound to, and fails if it is not the
// network the configuration says — a key pasted into the wrong variable is
// otherwise invisible until a device lands on the wrong network.
func (p *PlatformClient) Network(ctx context.Context) (*PlatformNetwork, error) {
	var out PlatformNetwork
	if err := p.do(ctx, http.MethodGet, "/platform/network", nil, nil, &out); err != nil {
		return nil, err
	}
	if p.networkID != "" && out.ID != p.networkID {
		return nil, fmt.Errorf("connectivity: this API key is for network %s, but the configuration names %s", out.ID, p.networkID)
	}
	if p.businessID != "" && out.BusinessID != p.businessID {
		return nil, fmt.Errorf("connectivity: this API key belongs to business %s, but the configuration names %s", out.BusinessID, p.businessID)
	}
	return &out, nil
}

func (p *PlatformClient) DeviceClasses(ctx context.Context) ([]DeviceClass, error) {
	var out []DeviceClass
	err := p.do(ctx, http.MethodGet, "/platform/device-classes", nil, nil, &out)
	return out, err
}

func (p *PlatformClient) CreateDevice(ctx context.Context, req CreatePlatformDeviceRequest) (*PlatformDevice, error) {
	var out PlatformDevice
	if err := p.do(ctx, http.MethodPost, "/platform/devices", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PlatformClient) GetDevice(ctx context.Context, deviceID string) (*PlatformDevice, error) {
	var out PlatformDevice
	if err := p.do(ctx, http.MethodGet, "/platform/devices/"+esc(deviceID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePlatformDeviceRequest changes a device. Empty fields are left
// unchanged.
type UpdatePlatformDeviceRequest struct {
	Name          string `json:"name,omitempty"`
	DeviceClassID string `json:"deviceClassId,omitempty"`
	// A nil Location leaves the stored one alone.
	Description string    `json:"description,omitempty"`
	Location    *Location `json:"location,omitempty"`
}

// UpdateDevice renames a device or moves it to another class on this network.
func (p *PlatformClient) UpdateDevice(ctx context.Context, deviceID string, req UpdatePlatformDeviceRequest) (*PlatformDevice, error) {
	var out PlatformDevice
	if err := p.do(ctx, http.MethodPut, "/platform/devices/"+esc(deviceID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PlatformClient) ListDevices(ctx context.Context, page, pageSize int) (*PagedResult[Device], error) {
	var out PagedResult[Device]
	if err := p.do(ctx, http.MethodGet, "/platform/devices", pageQuery(page, pageSize), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PlatformClient) DeleteDevice(ctx context.Context, deviceID string) error {
	return p.do(ctx, http.MethodDelete, "/platform/devices/"+esc(deviceID), nil, nil, nil)
}

// SetKey sets or replaces a device's key. There is no read-back.
func (p *PlatformClient) SetKey(ctx context.Context, deviceID, keyHex string) (*DeviceAccess, error) {
	var out DeviceAccess
	body := map[string]string{"key": keyHex}
	if err := p.do(ctx, http.MethodPut, "/platform/devices/"+esc(deviceID)+"/key", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PlatformClient) ClearKey(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := p.do(ctx, http.MethodDelete, "/platform/devices/"+esc(deviceID)+"/key", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Enable lets a cellular device's traffic through. A refusal is a conflict
// (IsConflict) whose message names what is missing.
func (p *PlatformClient) Enable(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := p.do(ctx, http.MethodPost, "/platform/devices/"+esc(deviceID)+"/enable", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PlatformClient) Disable(ctx context.Context, deviceID string) (*DeviceAccess, error) {
	var out DeviceAccess
	if err := p.do(ctx, http.MethodPost, "/platform/devices/"+esc(deviceID)+"/disable", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QueueCommand queues opaque bytes for a device. On cellular it overrides your
// platform's next answer, which is kept and sent on the reading after.
func (p *PlatformClient) QueueCommand(ctx context.Context, deviceID string, payload []byte, opts CommandOptions) (string, error) {
	body := map[string]any{"payloadHex": hex.EncodeToString(payload), "confirmed": opts.Confirmed}
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
		ID string `json:"id"`
	}
	if err := p.do(ctx, http.MethodPost, "/platform/devices/"+esc(deviceID)+"/commands", nil, body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (p *PlatformClient) PendingCommands(ctx context.Context, deviceID string) ([]Command, error) {
	var out []Command
	err := p.do(ctx, http.MethodGet, "/platform/devices/"+esc(deviceID)+"/commands", nil, nil, &out)
	return out, err
}

func (p *PlatformClient) ClearCommands(ctx context.Context, deviceID string) error {
	return p.do(ctx, http.MethodDelete, "/platform/devices/"+esc(deviceID)+"/commands", nil, nil, nil)
}

// Frames returns a page of a cellular device's wire traffic, newest first.
func (p *PlatformClient) Frames(ctx context.Context, deviceID string, q FrameQuery) (*FramePage, error) {
	var out FramePage
	if err := p.do(ctx, http.MethodGet, "/platform/devices/"+esc(deviceID)+"/frames", q.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
