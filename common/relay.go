package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/hashmatter/p3lib/sphinx"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
)

const (
	ProxyRelayProtocol    = "/pe2pe/0.0.1"
	ProxyRelayDialTimeout = time.Second * 15
	PacketLen             = 1330
	MagicWelcomeByte      = 0x42
)

type connectionOpenRequest struct {
	Timestamp int64
	Port      uint32
	Key       [chacha20poly1305.KeySize]byte
}

func (c *Client) StartRelay() error {

	// create relay context
	c.RelayCtx = sphinx.NewRelayerCtx(&c.Settings.Crypto.OnionKey)

	// bind listener
	c.Host.SetStreamHandler(ProxyRelayProtocol, func(s network.Stream) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Debugf("Got a new stream!")

		if err := c.serveRelayPackets(ctx, s); err != nil {
			log.Error(err)
			_ = s.Reset()
		} else {
			_ = s.Close()
		}
	})

	return nil
}

func (c *Client) serveRelayPackets(ctx context.Context, s network.Stream) error {

	var (
		decoder = gob.NewDecoder(s)
		packet  sphinx.Packet
	)

	err := decoder.Decode(&packet)
	if err != nil {
		return errors.Wrap(err, "decoding relay header")
	}

	// remove a layer of stuff
	nextAddr, nextPacket, err := c.RelayCtx.ProcessPacket(&packet)
	if err != nil {
		return errors.Wrap(err, "processing relay header")
	}

	if nextPacket.IsLast() && c.Settings.ExitNode != nil {
		// @TODO handle exit node
		return c.serveExitNode(ctx, nextPacket.Payload, s)

	} else if nextPacket.IsLast() {
		return fmt.Errorf("dead-end")
	}

	var dialAddr core.PeerID
	err = dialAddr.UnmarshalBinary(nextAddr[:34])
	if err != nil {
		return errors.Wrap(err, "parsing next hop addr")
	}

	stream, err := c.Host.NewStream(ctx, dialAddr, ProxyRelayProtocol)
	if err != nil {
		return errors.Wrap(err, "opening a stream to next hop")
	}

	var encoder = gob.NewEncoder(stream)

	err = encoder.Encode(nextPacket)
	if err != nil {
		return errors.Wrap(err, "encoding next header to stream")
	}

	return connectStream(stream, s)
}

// serveExitNode connects stream s to a local port
// @TODO: write errors back?
func (c *Client) serveExitNode(ctx context.Context, payload [256]byte, remoteConn network.Stream) error {

	log.Debugf("Payload is %v", payload)

	var (
		buf    = bytes.NewBuffer(payload[:])
		header connectionOpenRequest
	)

	// read the initial header from the encrypted payload
	err := binary.Read(buf, binary.BigEndian, &header)
	if err != nil {
		return errors.Wrap(err, "Failed to read request header")
	}

	if !c.Settings.ExitNode.IsPortAllowed(int(header.Port)) {
		return errors.Errorf("Port %v is not allowed", header.Port)
	}

	var dialer = &net.Dialer{}

	// open the local socket
	// @TODO: configurable ip addr
	localConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%v", header.Port))
	if err != nil {
		return errors.Wrap(err, "Failed to open local socket")
	}

	// create an encrypted readwriter
	secureConn, err := NewCryptoReadWriter(remoteConn, header.Key[:])
	if err != nil {
		return errors.Wrap(err, "Failed to open secure connection")
	}

	// dial is OK! fienally now we can send back the magic
	_, err = secureConn.Write([]byte{MagicWelcomeByte})
	if err != nil {
		return errors.Wrap(err, "Failed to write magic")
	}

	// connect secure stream with local pipe
	return connectStream(secureConn, localConn)
}
