package middleware

import (
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"devtools/models"

	"github.com/gorilla/websocket"
)

func newMiddlewareTestDB(t *testing.T) *models.DB {
	t.Helper()
	db, err := models.NewDB(filepath.Join(t.TempDir(), "ws.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitMonitoring(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMonitoredTunnelConnCountsAndPersists(t *testing.T) {
	db := newMiddlewareTestDB(t)
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	monitored := NewMonitoredTunnelConn(db, a, "example.com:443", "1.1.1.1:1234", "ua/test")

	go func() {
		_, _ = b.Write([]byte("client->upstream"))
		_, _ = b.Write([]byte("ping"))
		_ = b.Close()
	}()
	go func() {
		buf := make([]byte, 64)
		for {
			_, err := b.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 64)
	n, _ := monitored.Read(buf)
	if string(buf[:n]) != "client->upstream" {
		t.Fatalf("first read mismatch: %q", string(buf[:n]))
	}
	_, _ = monitored.Read(buf)
	_, _ = monitored.Write([]byte("upstream->client"))
	monitored.Finish("test_done")

	logs, total, err := db.ListProxySessionLogs("connect_tunnel", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 logged session, got total=%d logs=%+v", total, logs)
	}
	if logs[0].Target != "example.com:443" || logs[0].BytesIn != int64(len("client->upstream")+len("ping")) || logs[0].BytesOut != int64(len("upstream->client")) {
		t.Fatalf("counts wrong: %+v", logs[0])
	}
	if logs[0].CloseReason != "test_done" || logs[0].DurationMS < 0 {
		t.Fatalf("summary wrong: %+v", logs[0])
	}
}

type countWSConn struct {
	*websocket.Conn
	readBumps  int64
	writeBumps int64
}

func (c *countWSConn) ReadMessage() (int, []byte, error) {
	messageType, data, err := c.Conn.ReadMessage()
	if err == nil {
		atomic.AddInt64(&c.readBumps, 1)
	}
	return messageType, data, err
}

func (c *countWSConn) WriteMessage(messageType int, data []byte) error {
	err := c.Conn.WriteMessage(messageType, data)
	if err == nil {
		atomic.AddInt64(&c.writeBumps, 1)
	}
	return err
}

func TestMonitoredWSConnEmbedding(t *testing.T) {
	// Verify that MonitoredWSConn satisfies gorilla.Conn's method set so handlers
	// can keep calling SetReadDeadline/SetPongHandler etc. through the wrapper.
	var _ interface {
		ReadMessage() (int, []byte, error)
		WriteMessage(int, []byte) error
		SetReadDeadline(time.Time) error
		SetWriteDeadline(time.Time) error
		SetReadLimit(int64)
		Close() error
	} = (*MonitoredWSConn)(nil)
}
