package protocol_test

import (
	"errors"
	"testing"

	"github.com/rezadesky/highrisebot/protocol"
)

func TestProtocolEncoderDecoder(t *testing.T) {
	enc := protocol.NewEncoder()
	dec := protocol.NewDecoder()

	// 1. Encode valid struct
	payload := map[string]any{
		"_type":   "ChatRequest",
		"message": "Hello Highrise",
	}
	encoded, err := enc.Encode(payload)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	// 2. Decode Envelope
	env, err := dec.DecodeEnvelope(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if env.Type != "ChatRequest" {
		t.Errorf("expected _type 'ChatRequest', got '%s'", env.Type)
	}

	// 3. Decode invalid / empty data
	_, errEmpty := dec.DecodeEnvelope([]byte(""))
	if !errors.Is(errEmpty, protocol.ErrInvalidPayload) {
		t.Errorf("expected ErrInvalidPayload on empty bytes, got %v", errEmpty)
	}

	// 4. Missing _type field validation
	_, errNoType := dec.DecodeEnvelope([]byte(`{"foo":"bar"}`))
	if errNoType == nil {
		t.Errorf("expected error for missing _type, got nil")
	}
}

func TestProtocolVersion(t *testing.T) {
	if protocol.SupportedProtocolVersion == "" {
		t.Error("expected supported protocol version string")
	}
}
