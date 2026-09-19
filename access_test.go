package connectivity

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// The console gate and the platform gate must address the same device through
// different paths. A method pointed at the wrong one still returns a plausible
// DeviceAccess, so the path is the assertion.
func TestConsoleGateRoutesAndMethods(t *testing.T) {
	access := map[string]any{
		"enabled": true, "discovered": false, "blockers": []string{},
		"key": map[string]any{"set": true, "fingerprint": "3a44b5fc", "setAt": 1789574292},
	}
	for _, tc := range []struct {
		name       string
		call       func(*Client) error
		wantMethod string
		wantPath   string
	}{
		{"access", func(c *Client) error {
			_, err := c.DeviceAccess(context.Background(), "68753600037252")
			return err
		}, http.MethodGet, "/v1/devices/68753600037252/access"},
		{"set key", func(c *Client) error {
			_, err := c.SetDeviceKey(context.Background(), "68753600037252", "00112233445566778899aabbccddeeff")
			return err
		}, http.MethodPut, "/v1/devices/68753600037252/key"},
		{"clear key", func(c *Client) error {
			_, err := c.ClearDeviceKey(context.Background(), "68753600037252")
			return err
		}, http.MethodDelete, "/v1/devices/68753600037252/key"},
		{"enable", func(c *Client) error {
			_, err := c.EnableDevice(context.Background(), "68753600037252")
			return err
		}, http.MethodPost, "/v1/devices/68753600037252/enable"},
		{"disable", func(c *Client) error {
			_, err := c.DisableDevice(context.Background(), "68753600037252")
			return err
		}, http.MethodPost, "/v1/devices/68753600037252/disable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				writeEnvelope(w, 200, access)
			})
			if err := tc.call(client); err != nil {
				t.Fatal(err)
			}
			if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
				t.Errorf("%s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
			}
		})
	}
}

func TestSetDeviceKeySendsTheKeyAndDecodesTheGate(t *testing.T) {
	var body map[string]string
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		decodeBody(t, r, &body)
		writeEnvelope(w, 200, map[string]any{
			"enabled": false, "blockers": []string{},
			"key": map[string]any{"set": true, "fingerprint": "3a44b5fc", "setAt": 1789574292},
		})
	})
	got, err := client.SetDeviceKey(context.Background(), "d", "00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	if body["key"] != "00112233445566778899aabbccddeeff" {
		t.Errorf("key sent as %q", body["key"])
	}
	if !got.Key.Set || got.Key.Fingerprint != "3a44b5fc" {
		t.Errorf("gate = %+v", got.Key)
	}
	if got.Key.SetAt == nil || got.Key.SetAt.Unix() != 1789574292 {
		t.Errorf("setAt = %v", got.Key.SetAt)
	}
}

// A refusal to enable has to arrive as a conflict carrying what is missing —
// a caller that cannot distinguish it from a transport failure retries forever.
func TestEnableRefusalIsAConflictNamingTheBlocker(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeErrors(w, http.StatusConflict, "device_not_ready: the device has no key")
	})
	_, err := client.EnableDevice(context.Background(), "d")
	if !IsConflict(err) {
		t.Fatalf("err = %v, want a conflict", err)
	}
	if !contains(err.Error(), "the device has no key") {
		t.Errorf("the blocker did not survive: %v", err)
	}
}

// hasMore is the transport's own answer: a page cut on a whole second can be
// shorter than the limit with history still behind it, so a caller counting
// rows against its limit stops early. Dropping the field loses that.
func TestFramePagesCarryHasMore(t *testing.T) {
	var gotQuery, gotPath string
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery, gotPath = r.URL.RawQuery, r.URL.Path
		writeEnvelope(w, 200, map[string]any{
			"entries": []map[string]any{{"at": 1789574292, "direction": "up", "kind": "data", "payloadHex": "c0ffee", "sizeBytes": 3}},
			"hasMore": true,
		})
	})
	page, err := client.DeviceFrames(context.Background(), "d", FrameQuery{Limit: 1, Before: time.Unix(1789574300, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore {
		t.Error("hasMore was dropped; a caller cannot tell a full page from the end of history")
	}
	if len(page.Entries) != 1 || page.Entries[0].PayloadHex != "c0ffee" {
		t.Errorf("entries = %+v", page.Entries)
	}
	if gotPath != "/v1/devices/d/frames" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "before=1789574300&limit=1" {
		t.Errorf("query = %q", gotQuery)
	}
}

// An empty sampled page means nothing arrived while we listened, not that the
// device has never spoken. A client that drops the flag cannot tell them apart.
func TestLogPagesKeepTheSampledFlag(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"entries": []any{}, "hasMore": false, "sampled": true, "windowSeconds": 2,
		})
	})
	page, err := client.DeviceLogs(context.Background(), "d", FrameQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !page.Sampled || page.WindowSeconds != 2 {
		t.Errorf("sampled=%v window=%d — an empty live sample reads as a silent device", page.Sampled, page.WindowSeconds)
	}
}
