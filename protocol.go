package highrisebot

import (
	"encoding/json"
	"errors"
	"fmt"
)

// SupportedProtocolVersion records the current Highrise Bot API protocol version.
const SupportedProtocolVersion = "1.0.0"

// MaxMessageSize sets maximum inbound payload size (512 KB) to prevent OOM/DoS.
const MaxMessageSize = 512 * 1024

var (
	// ErrInvalidPayload is returned when a payload cannot be decoded or is empty.
	ErrInvalidPayload = errors.New("protocol: invalid message payload")

	// ErrAuthRejected is returned when authentication is refused by Highrise.
	ErrAuthRejected = errors.New("protocol: authentication rejected by server")

	// ErrRoomNotFound is returned when the target room does not exist or bot has no access.
	ErrRoomNotFound = errors.New("protocol: room not found or access denied")
)

// Envelope represents the top-level Highrise protocol frame.
type Envelope struct {
	Type  string          `json:"_type"`
	RID   string          `json:"rid,omitempty"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// ErrorPayload represents server error frames.
type ErrorPayload struct {
	Type           string `json:"_type"`
	Message        string `json:"message"`
	Code           int    `json:"code,omitempty"`
	DoNotReconnect bool   `json:"do_not_reconnect,omitempty"`
}

// ProtocolEncoder encodes structs/maps to protocol JSON bytes.
type ProtocolEncoder struct{}

// Encode converts any payload to raw JSON bytes.
func (e *ProtocolEncoder) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, ErrInvalidPayload
	}
	return json.Marshal(v)
}

// ProtocolDecoder parses incoming raw bytes into envelopes and typed messages.
type ProtocolDecoder struct{}

// DecodeEnvelope unmarshals raw data into a basic protocol Envelope.
func (d *ProtocolDecoder) DecodeEnvelope(data []byte) (*Envelope, error) {
	if len(data) == 0 {
		return nil, ErrInvalidPayload
	}
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if env.Type == "" {
		return nil, fmt.Errorf("%w: missing _type field", ErrInvalidPayload)
	}
	return &env, nil
}

// DecodeTyped decodes raw payload into target struct.
func (d *ProtocolDecoder) DecodeTyped(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
