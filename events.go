package connectivity

import (
	"encoding/json"
	"strings"
	"time"
)

// EventKind is what a message read from a queue destination turned out to be.
type EventKind string

const (
	// KindMessage is a message from a device.
	KindMessage EventKind = "message"
	// KindCommand is a command's lifecycle: command.queued, command.sent,
	// command.failed or command.retracted.
	KindCommand EventKind = "command"
	// KindAlert is a watch firing or resolving: device.quiet, device.signal,
	// device.recovered.
	KindAlert EventKind = "alert"
)

// CommandEvent tells the platform that queued a command what became of it.
type CommandEvent struct {
	// Event is "command.queued", "command.sent", "command.failed" or
	// "command.retracted".
	Event     string
	CommandID string
	DeviceID  string
	NetworkID string
	// Reference is what you passed as CommandOptions.Reference when queuing
	// it — your own id for the command, echoed so you need no lookup table.
	Reference string
	At        time.Time
	// Reason says why, on command.failed.
	Reason string
}

// AlertEvent is a watch firing ("device.quiet", "device.signal") or resolving
// ("device.recovered").
type AlertEvent struct {
	Event      string
	DeviceID   string
	NetworkID  string
	Rule       string
	LastSeenAt time.Time
	At         time.Time
}

// Event is one decoded queue message: a device message, a command's lifecycle
// or an alert — and the envelope, when the destination publishes one.
type Event struct {
	Kind    EventKind
	Message *Message
	Command *CommandEvent
	Alert   *AlertEvent
	// Set when the destination was configured with the event envelope.
	Topic     string
	EventType string
	Timestamp time.Time
}

type commandWire struct {
	Event     string `json:"event"`
	CommandID string `json:"commandId"`
	DeviceID  string `json:"deviceId"`
	NetworkID string `json:"networkId"`
	Reference string `json:"reference"`
	At        int64  `json:"at"`
	Reason    string `json:"reason"`
}

type alertWire struct {
	Event      string `json:"event"`
	DeviceID   string `json:"deviceId"`
	NetworkID  string `json:"networkId"`
	Rule       string `json:"rule"`
	LastSeenAt *int64 `json:"lastSeenAt"`
	At         int64  `json:"at"`
}

// DecodeMessage decodes one message read from a queue destination, in
// whichever shape the destination publishes: the bare object, or the event
// envelope {topic, eventType, timestamp, payload}. It is to a queue consumer
// what MessageHandler is to an HTTP endpoint.
func DecodeMessage(body []byte) (*Event, error) {
	var env struct {
		EventType *string         `json:"eventType"`
		Payload   json.RawMessage `json:"payload"`
		Topic     string          `json:"topic"`
		Timestamp int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	out := &Event{}
	inner := body
	// The envelope carries eventType and an OBJECT payload. A bare message has
	// a payload too — the device's bytes, as a string — and never eventType.
	if env.EventType != nil && len(env.Payload) > 0 && env.Payload[0] == '{' {
		out.Topic, out.EventType = env.Topic, *env.EventType
		if env.Timestamp != 0 {
			out.Timestamp = time.UnixMilli(env.Timestamp).UTC()
		}
		inner = env.Payload
	}

	var kind struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(inner, &kind); err != nil {
		return nil, err
	}
	switch {
	case strings.HasPrefix(kind.Event, "command."):
		var w commandWire
		if err := json.Unmarshal(inner, &w); err != nil {
			return nil, err
		}
		out.Kind = KindCommand
		out.Command = &CommandEvent{
			Event: w.Event, CommandID: w.CommandID, DeviceID: w.DeviceID, NetworkID: w.NetworkID,
			Reference: w.Reference, At: time.Unix(w.At, 0).UTC(), Reason: w.Reason,
		}
	case kind.Event != "":
		var w alertWire
		if err := json.Unmarshal(inner, &w); err != nil {
			return nil, err
		}
		out.Kind = KindAlert
		out.Alert = &AlertEvent{Event: w.Event, DeviceID: w.DeviceID, NetworkID: w.NetworkID, Rule: w.Rule, At: time.Unix(w.At, 0).UTC()}
		if w.LastSeenAt != nil {
			out.Alert.LastSeenAt = time.Unix(*w.LastSeenAt, 0).UTC()
		}
	default:
		var w messageWire
		if err := json.Unmarshal(inner, &w); err != nil {
			return nil, err
		}
		m, err := w.message()
		if err != nil {
			return nil, err
		}
		out.Kind = KindMessage
		out.Message = &m
	}
	return out, nil
}
