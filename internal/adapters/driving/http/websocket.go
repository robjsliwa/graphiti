package http

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
)

// WebSocketMessage is the JSON structure sent to connected browsers.
type WebSocketMessage struct {
	Type        string `json:"type"`
	RunID       string `json:"runID"`
	NodeID      string `json:"nodeID"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
	Duration    string `json:"duration,omitempty"`
}

// wsConn represents a WebSocket connection with a send channel.
type wsConn struct {
	conn net.Conn
	rw   *bufio.ReadWriter
	send chan []byte
	mu   sync.Mutex
}

func (c *wsConn) writeMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.rw == nil {
		// Test mode - just push to channel
		select {
		case c.send <- data:
		default:
		}
		return nil
	}

	// Write WebSocket frame (text, unmasked from server)
	frame := encodeWSFrame(1, data) // opcode 1 = text
	_, err := c.rw.Write(frame)
	if err != nil {
		return err
	}
	return c.rw.Flush()
}

func (c *wsConn) close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// WebSocketHub manages WebSocket connections grouped by workflow ID.
type WebSocketHub struct {
	mu          sync.RWMutex
	connections map[string]map[*wsConn]bool // workflowID -> connections
}

// NewWebSocketHub creates a new WebSocket hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		connections: make(map[string]map[*wsConn]bool),
	}
}

// Register adds a connection to a workflow's connection set.
func (h *WebSocketHub) Register(workflowID string, conn *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections[workflowID] == nil {
		h.connections[workflowID] = make(map[*wsConn]bool)
	}
	h.connections[workflowID][conn] = true
}

// Unregister removes a connection from a workflow's connection set.
func (h *WebSocketHub) Unregister(workflowID string, conn *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.connections[workflowID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, workflowID)
		}
	}
}

// BroadcastToWorkflow sends a message to all connections viewing a workflow.
func (h *WebSocketHub) BroadcastToWorkflow(workflowID string, msg WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("ws: marshal error", "error", err)
		return
	}

	h.mu.RLock()
	conns := h.connections[workflowID]
	h.mu.RUnlock()

	for conn := range conns {
		if err := conn.writeMessage(data); err != nil {
			slog.Error("ws: write error", "error", err)
			h.Unregister(workflowID, conn)
			conn.close()
		}
	}
}

// handleWebSocket handles WebSocket upgrade requests for workflow execution updates.
func handleWebSocket(hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		if workflowID == "" {
			http.Error(w, "missing workflow ID", http.StatusBadRequest)
			return
		}

		// Check for WebSocket upgrade
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "WebSocket upgrade required", http.StatusBadRequest)
			return
		}

		// Compute accept key
		key := r.Header.Get("Sec-WebSocket-Key")
		if key == "" {
			http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
			return
		}

		acceptKey := computeAcceptKey(key)

		// Hijack the connection
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		netConn, rw, err := hj.Hijack()
		if err != nil {
			http.Error(w, "hijack failed", http.StatusInternalServerError)
			return
		}

		// Send upgrade response
		upgradeResp := fmt.Sprintf(
			"HTTP/1.1 101 Switching Protocols\r\n"+
				"Upgrade: websocket\r\n"+
				"Connection: Upgrade\r\n"+
				"Sec-WebSocket-Accept: %s\r\n\r\n",
			acceptKey,
		)
		rw.WriteString(upgradeResp)
		rw.Flush()

		conn := &wsConn{
			conn: netConn,
			rw:   rw,
			send: make(chan []byte, 64),
		}

		hub.Register(workflowID, conn)
		slog.Info("ws: client connected", "workflow", workflowID)

		// Read loop (handles ping/pong and close)
		go func() {
			defer func() {
				hub.Unregister(workflowID, conn)
				conn.close()
				slog.Info("ws: client disconnected", "workflow", workflowID)
			}()

			for {
				_, err := readWSFrame(rw.Reader)
				if err != nil {
					return
				}
			}
		}()
	}
}

// computeAcceptKey computes the Sec-WebSocket-Accept value.
func computeAcceptKey(key string) string {
	const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// encodeWSFrame creates a WebSocket frame.
func encodeWSFrame(opcode byte, payload []byte) []byte {
	length := len(payload)
	var frame []byte

	frame = append(frame, 0x80|opcode) // FIN + opcode
	if length < 126 {
		frame = append(frame, byte(length))
	} else if length < 65536 {
		frame = append(frame, 126)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		frame = append(frame, buf...)
	} else {
		frame = append(frame, 127)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		frame = append(frame, buf...)
	}

	frame = append(frame, payload...)
	return frame
}

// readWSFrame reads a single WebSocket frame (simplified for server-side).
func readWSFrame(r *bufio.Reader) ([]byte, error) {
	// Read first 2 bytes
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Check for close frame
	opcode := header[0] & 0x0F
	if opcode == 8 {
		return nil, fmt.Errorf("close frame received")
	}

	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7F)

	if length == 126 {
		buf := make([]byte, 2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		length = int(binary.BigEndian.Uint16(buf))
	} else if length == 127 {
		buf := make([]byte, 8)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		length = int(binary.BigEndian.Uint64(buf))
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(r, mask); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return payload, nil
}
