package transport

import (
	"context"
	"time"

	"github.com/goplugin/wsrpc/credentials"
	"github.com/goplugin/wsrpc/logger"
)

const (
	defaultWriteTimeout = 10 * time.Second

	defaultReadLimit = int64(100_000_000) // 100 MB

	// Time allowed to read the next pong message from the peer.
	pongWait = 20 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// Abstracts websocket.Conn
type WebSocketConn interface {
	SetReadLimit(limit int64)
	SetReadDeadline(time time.Time) error
	SetPongHandler(handler func(string) error)
	SetWriteDeadline(time.Time) error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	Close() error
}

// ConnectOptions covers all relevant options for communicating with the server.
type ConnectOptions struct {
	// Time allowed to write a message to the connection.
	WriteTimeout time.Duration

	// Size of request allowed
	ReadLimit int64

	// TransportCredentials stores the Authenticator required to setup a client
	// connection.
	TransportCredentials credentials.TransportCredentials
}

// ClientTransport is the common interface for wsrpc client-side transport
// implementations.
type ClientTransport interface {
	// Read reads a message from the stream
	Read() <-chan []byte

	// Write sends a message to the stream.
	Write(ctx context.Context, msg []byte) error

	// Close tears down this transport. Once it returns, the transport
	// should not be accessed any more.
	Close()

	// Start starts this transport.
	Start()
}

// NewClientTransport establishes the transport with the required ConnectOptions
// and returns it to the caller.
func NewClientTransport(ctx context.Context, lggr logger.Logger, addr string, opts ConnectOptions, afterWritePump func()) (ClientTransport, error) {
	return newWebsocketClient(ctx, lggr, addr, opts, afterWritePump)
}

// state of transport.
type transportState int

const (
	// The default transport state.
	//
	// nolint is required because we don't actually use the var anywhere,
	// but it does represent a reachable transport.
	reachable transportState = iota //nolint:deadcode,varcheck
	closing
)

// ServerConfig consists of all the configurations to establish a server transport.
type ServerConfig struct {
	ReadLimit    int64
	WriteTimeout time.Duration
}

// ServerTransport is the common interface for wsrpc server-side transport
// implementations.
type ServerTransport interface {
	// Read reads a message from the stream.
	Read() <-chan []byte

	// Write sends a message to the stream.
	Write(ctx context.Context, msg []byte) error

	// Close tears down the transport. Once it is called, the transport
	// should not be accessed any more.
	Close() error
}

// NewServerTransport creates a ServerTransport with conn or non-nil error
// if it fails.
func NewServerTransport(c WebSocketConn, config *ServerConfig, afterWritePump func()) ServerTransport {
	return newWebsocketServer(c, config, afterWritePump)
}

func handlePong(conn WebSocketConn) func(string) error {
	return func(msg string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}
