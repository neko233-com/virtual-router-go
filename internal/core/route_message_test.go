package core

import "testing"

func TestRouteMessageEncodeDecodeRoundtrip(t *testing.T) {
	mt := RouteMessageTypeRpcRequest
	data := "{\"hello\":\"world\"}"
	msg := &RouteMessage{
		FromRouteId: "node-1",
		ToRouteId:   "node-2",
		MessageType: &mt,
		Data:        &data,
	}

	payload, err := msg.EncodePayload()
	if err != nil {
		t.Fatalf("EncodePayload error: %v", err)
	}

	decoded, err := DecodeRouteMessagePayload(payload)
	if err != nil {
		t.Fatalf("DecodeRouteMessagePayload error: %v", err)
	}

	if decoded.FromRouteId != msg.FromRouteId || decoded.ToRouteId != msg.ToRouteId {
		t.Fatalf("routeId mismatch: got %v -> %v", decoded.FromRouteId, decoded.ToRouteId)
	}
	if decoded.MessageType == nil || *decoded.MessageType != mt {
		t.Fatalf("messageType mismatch: got %v", decoded.MessageType)
	}
	if decoded.Data == nil || *decoded.Data != data {
		t.Fatalf("data mismatch: got %v", decoded.Data)
	}
}

func TestRouteMessageEncodeDecodeNullData(t *testing.T) {
	mt := RouteMessageTypeHeartBeat
	msg := &RouteMessage{
		FromRouteId: "node-a",
		ToRouteId:   "",
		MessageType: &mt,
		Data:        nil,
	}

	payload, err := msg.EncodePayload()
	if err != nil {
		t.Fatalf("EncodePayload error: %v", err)
	}

	decoded, err := DecodeRouteMessagePayload(payload)
	if err != nil {
		t.Fatalf("DecodeRouteMessagePayload error: %v", err)
	}
	if decoded.Data != nil {
		t.Fatalf("expected nil data, got %v", decoded.Data)
	}
}
