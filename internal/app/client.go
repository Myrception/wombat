package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
	"google.golang.org/protobuf/proto"
)

var errNoConn = errors.New("app: no connection available")

type client struct {
	conn *grpc.ClientConn
}

// transportCreds wraps the TLS credentials so that a handshake error can be
// surfaced to the user instead of the generic "TRANSIENT_FAILURE" state.
//
// gRPC performs handshakes on its own goroutines for the whole lifetime of the
// connection (including every reconnect), so the error channel is never closed
// and sends are non-blocking: only the first error is kept, later ones are
// dropped. Previously the channel was closed after connecting, which caused a
// "send on closed channel" panic on the first failed reconnect.
type transportCreds struct {
	credentials.TransportCredentials
	errc chan error
}

func (t *transportCreds) ClientHandshake(ctx context.Context, addr string, in net.Conn) (net.Conn, credentials.AuthInfo, error) {
	out, auth, err := t.TransportCredentials.ClientHandshake(ctx, addr, in)
	if err != nil {
		select {
		case t.errc <- err:
		default:
		}
	}
	return out, auth, err
}

func (c *client) connect(o options, h stats.Handler) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	handshakeErr := make(chan error, 1)

	opts := []grpc.DialOption{
		grpc.WithStatsHandler(h),
		grpc.WithUserAgent(fmt.Sprintf("%s/%s", appName, semver)),
	}

	if !o.Plaintext {
		var tlsCfg tls.Config
		tlsCfg.InsecureSkipVerify = o.Insecure

		if o.Clientcert != "" {
			cert, err := tls.X509KeyPair([]byte(o.Clientcert), []byte(o.Clientkey))
			if err != nil {
				return err
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		var err error
		tlsCfg.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			tlsCfg.RootCAs = x509.NewCertPool()
		}

		if o.Rootca != "" {
			tlsCfg.RootCAs.AppendCertsFromPEM([]byte(o.Rootca))
		}

		opts = append(opts, grpc.WithTransportCredentials(&transportCreds{
			TransportCredentials: credentials.NewTLS(&tlsCfg),
			errc:                 handshakeErr,
		}))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(o.Addr, opts...)
	if err != nil {
		return err
	}
	conn.Connect()

	// Wait for the connection to become READY, fail fast on the first failure.
	for {
		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			c.conn = conn
			return nil
		case connectivity.TransientFailure, connectivity.Shutdown:
			_ = conn.Close() // giving up on it; the handshake error matters
			select {
			case err := <-handshakeErr:
				return err
			default:
				return fmt.Errorf("connection in state: %s", state)
			}
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close() // giving up on it; the timeout matters
			return fmt.Errorf("timed out while in state %s: %w", state, ctx.Err())
		}
	}
}

func (c *client) invoke(ctx context.Context, method string, req, resp proto.Message) error {
	if c.conn == nil {
		return errNoConn
	}

	return c.conn.Invoke(ctx, method, req, resp)
}

func (c *client) invokeServerStream(ctx context.Context, method string, req proto.Message) (grpc.ClientStream, error) {
	if c.conn == nil {
		return nil, errNoConn
	}
	sd := &grpc.StreamDesc{
		StreamName:    method,
		ClientStreams: false,
		ServerStreams: true,
	}
	// The caller owns ctx and cancels it when the call is done.
	s, err := c.conn.NewStream(ctx, sd, method)
	if err != nil {
		return nil, err
	}
	if err := s.SendMsg(req); err != nil {
		return nil, err
	}
	if err := s.CloseSend(); err != nil {
		return nil, err
	}
	return s, nil
}

func (c *client) invokeClientStream(ctx context.Context, method string) (grpc.ClientStream, error) {
	if c.conn == nil {
		return nil, errNoConn
	}
	sd := &grpc.StreamDesc{
		StreamName:    method,
		ClientStreams: true,
		ServerStreams: false,
	}
	return c.conn.NewStream(ctx, sd, method)
}

func (c *client) invokeBidiStream(ctx context.Context, method string) (grpc.ClientStream, error) {
	if c.conn == nil {
		return nil, errNoConn
	}
	sd := &grpc.StreamDesc{
		StreamName:    method,
		ClientStreams: true,
		ServerStreams: true,
	}
	return c.conn.NewStream(ctx, sd, method)
}

func (c *client) close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
