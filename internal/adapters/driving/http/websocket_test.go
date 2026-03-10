package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSocketHub_RegisterAndUnregister(t *testing.T) {
	hub := NewWebSocketHub()

	conn1 := &wsConn{send: make(chan []byte, 16)}
	conn2 := &wsConn{send: make(chan []byte, 16)}

	hub.Register("wf-1", conn1)
	hub.Register("wf-1", conn2)
	hub.Register("wf-2", conn1)

	if len(hub.connections["wf-1"]) != 2 {
		t.Errorf("expected 2 connections for wf-1, got %d", len(hub.connections["wf-1"]))
	}
	if len(hub.connections["wf-2"]) != 1 {
		t.Errorf("expected 1 connection for wf-2, got %d", len(hub.connections["wf-2"]))
	}

	hub.Unregister("wf-1", conn1)
	if len(hub.connections["wf-1"]) != 1 {
		t.Errorf("after unregister, expected 1 connection for wf-1, got %d", len(hub.connections["wf-1"]))
	}
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	hub := NewWebSocketHub()

	conn1 := &wsConn{send: make(chan []byte, 16)}
	conn2 := &wsConn{send: make(chan []byte, 16)}
	conn3 := &wsConn{send: make(chan []byte, 16)}

	hub.Register("wf-1", conn1)
	hub.Register("wf-1", conn2)
	hub.Register("wf-2", conn3)

	msg := WebSocketMessage{
		Type:   "node_status",
		RunID:  "run-1",
		NodeID: "node-1",
		Status: "completed",
	}

	hub.BroadcastToWorkflow("wf-1", msg)

	// conn1 and conn2 should receive the message
	select {
	case data := <-conn1.send:
		var got WebSocketMessage
		json.Unmarshal(data, &got)
		if got.NodeID != "node-1" {
			t.Errorf("conn1 got NodeID=%q", got.NodeID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("conn1 didn't receive message")
	}

	select {
	case data := <-conn2.send:
		var got WebSocketMessage
		json.Unmarshal(data, &got)
		if got.Status != "completed" {
			t.Errorf("conn2 got Status=%q", got.Status)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("conn2 didn't receive message")
	}

	// conn3 should NOT receive (different workflow)
	select {
	case <-conn3.send:
		t.Error("conn3 should not have received message")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestWebSocketHub_BroadcastToEmpty(t *testing.T) {
	hub := NewWebSocketHub()
	// Should not panic
	hub.BroadcastToWorkflow("nonexistent", WebSocketMessage{})
}

func TestWebSocketUpgradeEndpoint_RejectsNonWebSocket(t *testing.T) {
	hub := NewWebSocketHub()
	handler := handleWebSocket(hub)

	req := httptest.NewRequest("GET", "/api/ws/workflows/wf-1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Without WebSocket upgrade headers, should fail
	if rr.Code == http.StatusOK {
		t.Error("should reject non-WebSocket requests")
	}
}

func TestWebSocketMessage_JSON(t *testing.T) {
	msg := WebSocketMessage{
		Type:   "node_status",
		RunID:  "run-123",
		NodeID: "node-abc",
		Status: "running",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	str := string(data)
	for _, want := range []string{"node_status", "run-123", "node-abc", "running"} {
		if !strings.Contains(str, want) {
			t.Errorf("JSON missing %q: %s", want, str)
		}
	}
}
