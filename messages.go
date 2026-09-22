package connectivity

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Message is one message from a device, whichever bearer carried it.
type Message struct {
	DeviceID   string
	NetworkID  string
	Bearer     Bearer
	ReceivedAt time.Time
	// Payload is exactly what the device sent. It is not decoded or decrypted
	// on the way to you.
	Payload []byte
	// Port and SignalDBm are set on the low-power bearer only.
	Port      int
	SignalDBm *int
	// Sequence is the device's own message counter, on the low-power bearer,
	// for a receiver that keeps per-message history. Nil where there is none.
	Sequence *int
	// AccessPoints lists every access point that heard the message, strongest
	// first, on the low-power bearer. SignalDBm is the first one's.
	AccessPoints []AccessPointReport
}

// AccessPointReport is one access point's reception of a message.
type AccessPointReport struct {
	// ID is the accessPointId the access point was registered with.
	ID   string  `json:"id"`
	RSSI int     `json:"rssi"`
	SNR  float64 `json:"snr"`
}

// Reply is what a cellular device is sent back on the socket it is holding
// open. Return nil for "nothing to send". It is ignored on the low-power
// bearer, whose devices are reached with QueueCommand.
type Reply struct {
	Payload []byte
	// Close ends the device's connection after the reply is sent.
	Close bool
}

// MessageFunc handles one message. An error answers 500, and on cellular the
// reading is then recorded on our side as received but not accepted by you.
type MessageFunc func(ctx context.Context, up Message) (*Reply, error)

type messageWire struct {
	DeviceID     string              `json:"deviceId"`
	NetworkID    string              `json:"networkId"`
	Bearer       Bearer              `json:"bearer"`
	ReceivedAt   int64               `json:"receivedAt"`
	PayloadHex   string              `json:"payloadHex"`
	PayloadB64   string              `json:"payload"`
	Port         int                 `json:"port"`
	SignalDBm    *int                `json:"signalDbm"`
	Sequence     *int                `json:"sequence"`
	AccessPoints []AccessPointReport `json:"accessPoints"`
}

// MessageHandler is the endpoint you register as your platform's URL. It
// authenticates every request by authHeader carrying secret, and refuses
// everything when secret is empty rather than accepting anything.
//
// The same handler serves both bearers: a cellular reading is answered
// synchronously with your Reply; a low-power message is acknowledged.
func MessageHandler(authHeader, secret string, fn MessageFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		got := r.Header.Get(authHeader)
		if secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var in messageWire
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "malformed message: "+err.Error(), http.StatusBadRequest)
			return
		}
		up, err := in.message()
		if err != nil {
			http.Error(w, "malformed message: "+err.Error(), http.StatusBadRequest)
			return
		}
		reply, err := fn(r.Context(), up)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		out := map[string]any{"payloadHex": "", "close": false}
		if reply != nil && up.Bearer == BearerCellular {
			out["payloadHex"], out["close"] = hex.EncodeToString(reply.Payload), reply.Close
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}

func (in messageWire) message() (Message, error) {
	up := Message{
		DeviceID: in.DeviceID, NetworkID: in.NetworkID, Bearer: in.Bearer,
		ReceivedAt: time.Unix(in.ReceivedAt, 0).UTC(), Port: in.Port, SignalDBm: in.SignalDBm,
		Sequence: in.Sequence, AccessPoints: in.AccessPoints,
	}
	var err error
	switch {
	case in.PayloadHex != "":
		up.Payload, err = hex.DecodeString(in.PayloadHex)
		if up.Bearer == "" {
			up.Bearer = BearerCellular
		}
	case in.PayloadB64 != "":
		up.Payload, err = base64.StdEncoding.DecodeString(in.PayloadB64)
		if up.Bearer == "" {
			up.Bearer = BearerLPWAN
		}
	}
	return up, err
}
