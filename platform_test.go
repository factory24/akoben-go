package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlatformClientSendsTheKeyAndChecksItsNetwork(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		_, _ = io.WriteString(w, `{"success":true,"data":{"id":"net-other","businessId":"biz-1","bearer":"cellular"}}`)
	}))
	defer srv.Close()

	t.Setenv("AKOBEN.BASE_URL", srv.URL+"/v1")
	t.Setenv("AKOBEN.CELLULAR.API_KEY", "k-123")
	t.Setenv("AKOBEN.CELLULAR.NETWORK_ID", "net-mine")
	p, err := NewPlatformFromEnv("cellular")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Network(context.Background())
	if gotAuth != "Bearer k-123" || gotPath != "/v1/platform/network" {
		t.Errorf("sent auth=%q path=%q", gotAuth, gotPath)
	}
	if err == nil || !strings.Contains(err.Error(), "net-mine") {
		t.Errorf("a key for another network was accepted: %v", err)
	}
}

func TestMessageHandlerAnswersCellularAndAcknowledgesLowPower(t *testing.T) {
	var got []Message
	h := MessageHandler("X-Message-Secret", "s3cret", func(_ context.Context, up Message) (*Reply, error) {
		got = append(got, up)
		return &Reply{Payload: []byte{0x68, 0x16}}, nil
	})

	post := func(secret, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString(body))
		req.Header.Set("X-Message-Secret", secret)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("wrong", `{}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong secret answered %d", rec.Code)
	}

	rec := post("s3cret", `{"deviceId":"68753600037252","networkId":"n1","bearer":"cellular","receivedAt":1789483989,"payloadHex":"6810aa16"}`)
	var answer struct {
		PayloadHex string `json:"payloadHex"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&answer)
	if rec.Code != http.StatusOK || answer.PayloadHex != "6816" {
		t.Errorf("cellular reading answered %d %+v, want the reply", rec.Code, answer)
	}

	rec = post("s3cret", `{"deviceId":"0004a30b001c0530","networkId":"n2","receivedAt":1789483989,"port":2,"payload":"AQIDBAU=","signalDbm":-57}`)
	_ = json.NewDecoder(rec.Body).Decode(&answer)
	if rec.Code != http.StatusOK || answer.PayloadHex != "" {
		t.Errorf("low-power message answered %d %+v, want an empty acknowledgement", rec.Code, answer)
	}

	if len(got) != 2 {
		t.Fatalf("handler saw %d messages, want 2", len(got))
	}
	if got[0].Bearer != BearerCellular || !bytes.Equal(got[0].Payload, []byte{0x68, 0x10, 0xaa, 0x16}) {
		t.Errorf("cellular message decoded as %+v", got[0])
	}
	if got[1].Bearer != BearerLPWAN || got[1].Port != 2 || !bytes.Equal(got[1].Payload, []byte{1, 2, 3, 4, 5}) {
		t.Errorf("low-power message decoded as %+v", got[1])
	}
}

func TestMessageHandlerRefusesEverythingWithoutASecret(t *testing.T) {
	h := MessageHandler("X-Message-Secret", "", func(context.Context, Message) (*Reply, error) {
		t.Fatal("an unconfigured handler accepted a message")
		return nil, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("answered %d, want 401", rec.Code)
	}
}

// A platform has to be able to name itself on a device it is creating: with
// more than one platform on a business, a device that says nothing takes the
// business default, which may not be the one provisioning it. The field was
// honoured by the API and absent from the SDK, so the one caller that most
// needs it — a platform's own client — could not send it.
func TestPlatformDeviceRequestsCarryTheRoutingPlatform(t *testing.T) {
	create, err := json.Marshal(CreatePlatformDeviceRequest{
		DeviceID: "68753699000001", Name: "meter", DeviceClassID: "c1", PlatformID: "p1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(create), `"platformId":"p1"`) {
		t.Fatalf("create request dropped the platform: %s", create)
	}
	update, err := json.Marshal(UpdatePlatformDeviceRequest{PlatformID: "p2"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(update), `"platformId":"p2"`) {
		t.Fatalf("update request dropped the platform: %s", update)
	}
	// Omitted must stay absent, or every rename would reroute the device to "".
	plain, err := json.Marshal(UpdatePlatformDeviceRequest{Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "platformId") {
		t.Fatalf("an omitted platform was sent anyway: %s", plain)
	}
}

func TestPlatformClientRegistersAnAccessPointOnItsKeysNetwork(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotMethod, gotPath = r.Header.Get("Authorization"), r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"success":true,"data":{"accessPointId":"54d0b4fffe45a496","networkId":"net-1"}}`)
	}))
	defer srv.Close()

	ap, err := NewPlatform(srv.URL+"/v1", "k-lpwan").RegisterAccessPoint(context.Background(), RegisterAccessPointRequest{
		NetworkID: "ignored", AccessPointID: "54D0B4FFFE45A496", Name: "Kibera North", ReportEverySeconds: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer k-lpwan" || gotMethod != http.MethodPost || gotPath != "/v1/platform/access-points" {
		t.Errorf("sent auth=%q %s %s", gotAuth, gotMethod, gotPath)
	}
	if _, named := gotBody["networkId"]; named || gotBody["accessPointId"] != "54D0B4FFFE45A496" {
		t.Errorf("body %v: the key decides the network, and the id goes as given", gotBody)
	}
	if ap.NetworkID != "net-1" {
		t.Errorf("reply not decoded: %+v", ap)
	}
}
