package connectivity

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// A device's position is optional and must stay optional on the wire: a
// register that names no location must not send 0, 0, which is a real
// coordinate a few hundred kilometres off the coast of the fleet this was
// built for. Every device on a customer's map would otherwise be there.
func TestRegisterDeviceSendsLocationOnlyWhenGiven(t *testing.T) {
	var got map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeEnvelope(w, 201, map[string]any{"deviceId": "0102030405060708"})
	})

	if _, err := client.RegisterDevice(context.Background(), RegisterDeviceRequest{
		NetworkID: "n-1", DeviceID: "0102030405060708", Name: "Meter 4",
	}); err != nil {
		t.Fatal(err)
	}
	if _, sent := got["location"]; sent {
		t.Errorf("a register with no location sent one: %v", got)
	}
	if _, sent := got["description"]; sent {
		t.Errorf("a register with no description sent one: %v", got)
	}

	got = nil
	if _, err := client.RegisterDevice(context.Background(), RegisterDeviceRequest{
		NetworkID: "n-1", DeviceID: "0102030405060708", Name: "Meter 4",
		Description: "Kumasi zone 3, standpipe", Location: At(6.6666, -1.6163),
	}); err != nil {
		t.Fatal(err)
	}
	location, ok := got["location"].(map[string]any)
	if !ok {
		t.Fatalf("location was not sent: %v", got)
	}
	if location["latitude"] != 6.6666 || location["longitude"] != -1.6163 {
		t.Errorf("location = %v", location)
	}
	if got["description"] != "Kumasi zone 3, standpipe" {
		t.Errorf("description = %v", got["description"])
	}
}

// Zero is a legitimate degree — Ghana sits on the prime meridian — so a
// caller that means it must be able to send it.
func TestZeroDegreesIsSentAndNotTreatedAsAbsent(t *testing.T) {
	var got map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeEnvelope(w, 200, map[string]any{"deviceId": "d1"})
	})
	if _, err := client.UpdateDevice(context.Background(), UpdateDeviceRequest{
		DeviceID: "d1", Location: At(5.6037, 0),
	}); err != nil {
		t.Fatal(err)
	}
	location, ok := got["location"].(map[string]any)
	if !ok {
		t.Fatalf("location was not sent: %v", got)
	}
	if location["longitude"] != float64(0) {
		t.Errorf("longitude = %v, want 0 on the wire", location["longitude"])
	}
}

// A device that has not been placed reads back with no location at all,
// never as a pair of zeroes.
func TestAnUnplacedDeviceReadsBackWithoutALocation(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeEnvelope(w, 200, map[string]any{"deviceId": "d1", "name": "Meter 4"})
	})
	d, err := client.GetDevice(context.Background(), "d1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Location != nil {
		t.Errorf("location = %+v, want none", d.Location)
	}

	client, _ = newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"deviceId": "d1", "description": "Kumasi zone 3",
			"location": map[string]any{"latitude": 6.6666, "longitude": -1.6163},
		})
	})
	if d, err = client.GetDevice(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	if d.Location == nil || d.Location.Latitude != 6.6666 || d.Location.Longitude != -1.6163 {
		t.Errorf("location = %+v", d.Location)
	}
	if d.Description != "Kumasi zone 3" {
		t.Errorf("description = %q", d.Description)
	}
}

// The platform client (a customer's own backend, one network's API key) takes
// the same two fields, or a 4G meter installed through it would be placeless
// where a LoRa one is not.
func TestPlatformCreateCarriesLocation(t *testing.T) {
	var got map[string]any
	srv := newPlatformTestServer(t, &got)
	if _, err := srv.CreateDevice(context.Background(), CreatePlatformDeviceRequest{
		DeviceID: "68753600037252", Name: "Meter 4", DeviceClassID: "c-1",
		Description: "Tamale zone 1", Location: At(9.4008, -0.8393),
	}); err != nil {
		t.Fatal(err)
	}
	location, ok := got["location"].(map[string]any)
	if !ok || location["latitude"] != 9.4008 {
		t.Fatalf("location = %v", got["location"])
	}
	if got["description"] != "Tamale zone 1" {
		t.Errorf("description = %v", got["description"])
	}
}

func newPlatformTestServer(t *testing.T, got *map[string]any) *PlatformClient {
	t.Helper()
	client, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(got)
		writeEnvelope(w, 201, map[string]any{"deviceId": "68753600037252"})
	})
	_ = client
	return NewPlatform(srv.URL+"/v1", "k-1")
}
