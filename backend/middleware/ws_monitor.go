package middleware

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"devtools/models"

	"github.com/gorilla/websocket"
)

// MonitoredWSConn wraps gorilla/websocket.Conn to count frames/bytes and
// persist a single summary row when the connection closes. The wrapper
// embeds the original conn so existing handler call sites (NextReader,
// SetReadDeadline, SetPongHandler, etc.) keep working without changes.
type MonitoredWSConn struct {
	*websocket.Conn
	summary models.ProxySessionLog
	db      *models.DB

	framesIn  int64
	bytesIn   int64
	framesOut int64
	bytesOut  int64

	startedAt time.Time
	closed    atomic.Bool
	closeMu   sync.Mutex
}

func NewMonitoredWSConn(db *models.DB, conn *websocket.Conn, route, target, clientIP, userAgent, sessionType string) *MonitoredWSConn {
	return &MonitoredWSConn{
		Conn:      conn,
		db:        db,
		startedAt: time.Now(),
		summary: models.ProxySessionLog{
			SessionType: sessionType,
			Method:      "WS",
			Route:       truncateValue(route, 512),
			Target:      truncateValue(target, 512),
			ClientIP:    truncateValue(clientIP, 128),
			UserAgent:   truncateValue(userAgent, 512),
			StatusCode:  101,
			CreatedAt:   time.Now().UTC(),
		},
	}
}

func (c *MonitoredWSConn) ReadMessage() (int, []byte, error) {
	messageType, data, err := c.Conn.ReadMessage()
	if err == nil {
		atomic.AddInt64(&c.framesIn, 1)
		atomic.AddInt64(&c.bytesIn, int64(len(data)))
	}
	return messageType, data, err
}

func (c *MonitoredWSConn) WriteMessage(messageType int, data []byte) error {
	err := c.Conn.WriteMessage(messageType, data)
	if err == nil {
		atomic.AddInt64(&c.framesOut, 1)
		atomic.AddInt64(&c.bytesOut, int64(len(data)))
	}
	return err
}

func (c *MonitoredWSConn) Close() error {
	c.markClosed("")
	return c.Conn.Close()
}

// MarkClosed records a session summary row with a given close reason before
// closing the underlying conn. Safe to call multiple times.
func (c *MonitoredWSConn) MarkClosed(reason string) {
	c.markClosed(reason)
	_ = c.Conn.Close()
}

func (c *MonitoredWSConn) markClosed(reason string) {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	summary := c.summary
	summary.FramesIn = atomic.LoadInt64(&c.framesIn)
	summary.BytesIn = atomic.LoadInt64(&c.bytesIn)
	summary.FramesOut = atomic.LoadInt64(&c.framesOut)
	summary.BytesOut = atomic.LoadInt64(&c.bytesOut)
	summary.DurationMS = time.Since(c.startedAt).Milliseconds()
	if reason == "" {
		reason = "closed"
	}
	summary.CloseReason = truncateValue(reason, 255)
	summary.CreatedAt = time.Now().UTC()
	if c.db == nil {
		return
	}
	_ = c.db.CreateProxySessionLog(&summary)
}

// MonitoredTunnelConn counts read+write bytes passing through a CONNECT
// tunnel. Used for the upstream proxy where the connection is hijacked and
// streams raw TCP. Returns a finalizer that callers invoke when the tunnel
// ends.
type MonitoredTunnelConn struct {
	net.Conn
	summary models.ProxySessionLog
	db      *models.DB

	bytesIn   int64
	bytesOut  int64
	startedAt time.Time
	closed    atomic.Bool
	closeMu   sync.Mutex
}

// NewMonitoredTunnelConn wraps a net.Conn. The caller passes the metadata
// about the CONNECT request (target, client IP, etc.) and must invoke
// Finish(reason) when the tunnel ends.
func NewMonitoredTunnelConn(db *models.DB, conn net.Conn, target, clientIP, userAgent string) *MonitoredTunnelConn {
	return &MonitoredTunnelConn{
		Conn:      conn,
		db:        db,
		startedAt: time.Now(),
		summary: models.ProxySessionLog{
			SessionType: "connect_tunnel",
			Method:      "CONNECT",
			Target:      truncateValue(target, 512),
			ClientIP:    truncateValue(clientIP, 128),
			UserAgent:   truncateValue(userAgent, 512),
			StatusCode:  200,
			CreatedAt:   time.Now().UTC(),
		},
	}
}

func (c *MonitoredTunnelConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		atomic.AddInt64(&c.bytesIn, int64(n))
	}
	return n, err
}

func (c *MonitoredTunnelConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		atomic.AddInt64(&c.bytesOut, int64(n))
	}
	return n, err
}

func (c *MonitoredTunnelConn) Finish(reason string) {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	summary := c.summary
	summary.BytesIn = atomic.LoadInt64(&c.bytesIn)
	summary.BytesOut = atomic.LoadInt64(&c.bytesOut)
	summary.DurationMS = time.Since(c.startedAt).Milliseconds()
	if reason == "" {
		reason = "tunnel_closed"
	}
	summary.CloseReason = truncateValue(reason, 255)
	summary.CreatedAt = time.Now().UTC()
	_ = c.Conn.Close()
	if c.db == nil {
		return
	}
	_ = c.db.CreateProxySessionLog(&summary)
}

func truncateValue(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
