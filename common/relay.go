package common

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
)

const (
	ProxyRelayProtocol    = "/pe2pe/0.0.1"
	ProxyRelayDialTimeout = time.Second * 15
)

type connectionOpenRequest struct {
	Port uint32
}

func (c *Client) StartRelay() error {

	c.Host.SetStreamHandler(ProxyRelayProtocol, func(s network.Stream) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Debugf("Got a new stream!")

		if err := c.serveExitNode(ctx, s); err != nil {
			log.Error(err)
			_ = s.Reset()
		} else {
			_ = s.Close()
		}
	})

	return nil
}

// doEcho reads a line of data a stream and writes it back
func (c *Client) serveExitNode(ctx context.Context, s io.ReadWriter) error {

	// read the initial header
	var header connectionOpenRequest
	err := binary.Read(s, binary.BigEndian, &header)
	if err != nil {
		return errors.Wrap(err, "Failed to read request header")
	}

	var dialer = &net.Dialer{}

	// open the local socket
	// @TODO: configurable ip addr
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%v", header.Port))
	if err != nil {
		return errors.Wrap(err, "Failed to open local socket")
	}

	go func() {
		// copy data one side in async mode %)
		_, err := io.Copy(conn, s)
		if err != nil {
			log.Errorf("Error while copying data: %v", err)
		}
	}()

	// copy data back side in sync mode
	_, err = io.Copy(s, conn)
	if err != nil {
		return err
	}

	return nil
}
