package connectivity

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// A queue consumer reads whatever shape the destination was configured to
// publish. Every shape decodes to the same Event, so a consumer branches on
// Kind and never on the wire.
func TestDecodeMessageHandlesEveryShape(t *testing.T) {
	lpwan := `{"deviceId":"0102030405060708","networkId":"n1","bearer":"lorwan","receivedAt":1789500000,"port":10,"payload":"AQID","signalDbm":-88,"sequence":412,"accessPoints":[{"id":"aa555a0000000a01","rssi":-88,"snr":7.5},{"id":"aa555a0000000a02","rssi":-101,"snr":-2}]}`
	ev, err := DecodeMessage([]byte(lpwan))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindMessage || ev.Message == nil {
		t.Fatalf("kind = %q", ev.Kind)
	}
	m := ev.Message
	if m.Bearer != BearerLPWAN || m.DeviceID != "0102030405060708" || string(m.Payload) != "\x01\x02\x03" || m.Port != 10 {
		t.Errorf("message = %+v", m)
	}
	if m.Sequence == nil || *m.Sequence != 412 || len(m.AccessPoints) != 2 || m.AccessPoints[0].ID != "aa555a0000000a01" || m.AccessPoints[1].SNR != -2 {
		t.Errorf("sequence/access points = %v %+v", m.Sequence, m.AccessPoints)
	}
	if ev.EventType != "" || ev.Topic != "" {
		t.Error("a bare message has no envelope")
	}

	cellular := `{"deviceId":"68753600037252","networkId":"n2","bearer":"cellular","receivedAt":1789500001,"payload":"aBA="}`
	ev, err = DecodeMessage([]byte(cellular))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindMessage || ev.Message.Bearer != BearerCellular || ev.Message.Sequence != nil || ev.Message.AccessPoints != nil {
		t.Errorf("cellular = %+v", ev.Message)
	}

	wrapped := `{"topic":"persistent://public/default/uplinks","eventType":"uplink.cellular","timestamp":1789500001234,"payload":` + cellular + `}`
	ev, err = DecodeMessage([]byte(wrapped))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindMessage || ev.EventType != "uplink.cellular" || ev.Topic != "persistent://public/default/uplinks" || ev.Message.DeviceID != "68753600037252" {
		t.Errorf("wrapped = %+v", ev)
	}
	if ev.Timestamp != time.UnixMilli(1789500001234).UTC() {
		t.Errorf("timestamp = %s, want milliseconds decoded", ev.Timestamp)
	}

	command := `{"event":"command.sent","commandId":"c1","deviceId":"68753600037252","networkId":"n2","reference":"queue-item-77","at":1789500002}`
	ev, err = DecodeMessage([]byte(command))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindCommand || ev.Command.Event != "command.sent" || ev.Command.Reference != "queue-item-77" || ev.Command.CommandID != "c1" {
		t.Errorf("command = %+v", ev.Command)
	}
	wrappedCommand := `{"topic":"t","eventType":"command.failed","timestamp":1,"payload":{"event":"command.failed","commandId":"c2","deviceId":"d","networkId":"n","at":1789500003,"reason":"expired"}}`
	ev, err = DecodeMessage([]byte(wrappedCommand))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindCommand || ev.Command.Reason != "expired" || ev.EventType != "command.failed" {
		t.Errorf("wrapped command = %+v %+v", ev, ev.Command)
	}

	alert := `{"event":"device.quiet","deviceId":"d","networkId":"n","rule":"Tell me when a meter stops","lastSeenAt":1789431886,"at":1789435486}`
	ev, err = DecodeMessage([]byte(alert))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != KindAlert || ev.Alert.Rule != "Tell me when a meter stops" || ev.Alert.LastSeenAt.Unix() != 1789431886 {
		t.Errorf("alert = %+v", ev.Alert)
	}

	if _, err := DecodeMessage([]byte(`not json`)); err == nil {
		t.Error("malformed input decoded")
	}
}

// The reference travels with the command and comes back on its events, so a
// platform needs no table from our command id to its own.
func TestSendCommandCarriesTheReference(t *testing.T) {
	var got map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeEnvelope(w, 201, map[string]any{"commandId": "c1"})
	})
	if _, err := client.SendCommand(context.Background(), "d1", []byte{1}, CommandOptions{Reference: "queue-item-77", ExpiresIn: 2 * time.Minute}); err != nil {
		t.Fatal(err)
	}
	if got["reference"] != "queue-item-77" || got["expiresInSeconds"] != float64(120) {
		t.Errorf("body = %v, want the reference and the deadline in seconds", got)
	}
	got = nil
	if _, err := client.SendCommand(context.Background(), "d1", []byte{1}, CommandOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, present := got["reference"]; present {
		t.Error("an empty reference was sent")
	}
}

// Optional keys are sent only when set, so an unchanged destination reads
// back as it was written and a token is never sent as an empty string that
// would rotate the stored one to nothing.
func TestQueueConfigRendersOnlyWhatIsSet(t *testing.T) {
	plain := QueueConfig{BrokerURL: "pulsar://b:6650", Topic: "t"}.Config()
	if len(plain) != 2 {
		t.Errorf("plain config = %v, want brokerUrl and topic only", plain)
	}
	full := QueueConfig{
		BrokerURL: "pulsar+ssl://b:6651", Topic: "t", Token: "tok", TLSTrustPEM: "-----BEGIN CERTIFICATE-----",
		TLSAllowInsecure: true, Envelope: EnvelopeEvent, EventType: "uplink.lpwan",
	}.Config()
	if full["token"] != "tok" || full["tlsAllowInsecure"] != true || full["envelope"] != "event" || full["eventType"] != "uplink.lpwan" || full["tlsTrustPem"] == nil {
		t.Errorf("full config = %v", full)
	}
}
