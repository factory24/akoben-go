package connectivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL+"/v1", StaticToken("test-token")), srv
}

func writeEnvelope(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": status < 300, "timestamp": time.Now().Unix(),
		"message": "ok", "data": data,
	})
}

func TestAuthHeadersAndPath(t *testing.T) {
	var gotAuth, gotBusiness, gotPath string
	client, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBusiness = r.Header.Get("X-Business-Id")
		gotPath = r.URL.Path
		writeEnvelope(w, 200, map[string]any{"deviceId": "0102030405060708", "status": "active"})
	})
	client = New(srv.URL+"/v1", StaticToken("tok"), WithBusinessID("biz-1"))

	d, err := client.GetDevice(context.Background(), "0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBusiness != "biz-1" {
		t.Errorf("X-Business-Id = %q", gotBusiness)
	}
	if gotPath != "/v1/devices/0102030405060708" {
		t.Errorf("path = %q", gotPath)
	}
	if d.State != StateActive {
		t.Errorf("state = %q", d.State)
	}
}

func TestUnixTimeDecoding(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"deviceId": "a", "lastSeenAt": 1788310800, "createdAt": nil,
		})
	})
	d, err := client.GetDevice(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if d.LastSeenAt == nil || d.LastSeenAt.Unix() != 1788310800 {
		t.Errorf("lastSeenAt = %v", d.LastSeenAt)
	}
	if d.CreatedAt != nil {
		t.Errorf("createdAt should stay nil, got %v", d.CreatedAt)
	}
}

func TestErrorEnvelope(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false, "message": "Device not found",
			"errors": []string{"not_found: Device not found"},
		})
	})
	_, err := client.GetDevice(context.Background(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected not-found, got %v", err)
	}
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "not_found" {
		t.Errorf("code = %+v", ae)
	}
}

func TestSendCommandEncodesHex(t *testing.T) {
	var body map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeEnvelope(w, 201, map[string]any{"commandId": "cmd-9"})
	})
	id, err := client.SendCommand(context.Background(), "dev-1", []byte{0xFE, 0xFE, 0x68}, CommandOptions{Port: 10, Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if id != "cmd-9" {
		t.Errorf("commandId = %q", id)
	}
	if body["payloadHex"] != "fefe68" {
		t.Errorf("payloadHex = %v", body["payloadHex"])
	}
	if body["port"] != float64(10) || body["confirmed"] != true {
		t.Errorf("port/confirmed = %v/%v", body["port"], body["confirmed"])
	}
}

func TestSetDeviceSecretValidatesShape(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, nil)
	})
	if err := client.SetDeviceSecret(context.Background(), "d", "nothex"); err == nil {
		t.Error("expected refusal of a non-hex secret")
	}
	if err := client.SetDeviceSecret(context.Background(), "d", "00112233445566778899aabbccddeeff"); err != nil {
		t.Errorf("valid secret refused: %v", err)
	}
}

func TestListDevicesPagedEnvelope(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("bearer") != "lorwan" || r.URL.Query().Get("pageSize") != "50" {
			t.Errorf("query = %v", r.URL.Query())
		}
		writeEnvelope(w, 200, map[string]any{
			"content":       []map[string]any{{"deviceId": "a"}, {"deviceId": "b"}},
			"totalElements": 2, "page": 1, "pageSize": 50,
		})
	})
	res, err := client.ListDevices(context.Background(), ListDevicesRequest{Bearer: BearerLPWAN, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) != 2 || res.TotalElements != 2 {
		t.Errorf("paged = %+v", res)
	}
}

func TestClearCommands(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		writeEnvelope(w, 200, map[string]any{"retracted": 3})
	})
	n, err := client.ClearCommands(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("retracted = %d", n)
	}
}

func TestClientCredentialsTokenSourceCachesUntilExpiry(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "flow" {
			t.Errorf("form = %v", r.Form)
		}
		fmt.Fprintf(w, `{"access_token":"tok-%d","expires_in":3600}`, calls)
	}))
	defer srv.Close()

	ts := NewClientCredentialsTokenSource(srv.URL, "flow", "secret")
	for i := 0; i < 3; i++ {
		tok, err := ts.Token(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if tok != "tok-1" {
			t.Errorf("token = %q", tok)
		}
	}
	if calls != 1 {
		t.Errorf("token endpoint called %d times, want 1", calls)
	}
}
func decodeBody(t *testing.T, r *http.Request, out any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatalf("decoding request body: %v", err)
	}
}

func writeEnvelopeErrors(w http.ResponseWriter, status int, errs ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false, "timestamp": time.Now().Unix(),
		"message": "request failed", "errors": errs,
	})
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func newRecordingServer(t *testing.T, method, path, auth *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*method, *path, *auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		writeEnvelope(w, 200, nil)
	}))
	t.Cleanup(srv.Close)
	return srv
}
