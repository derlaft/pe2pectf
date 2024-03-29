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

	"github.com/derlaft/connectstream"
	"github.com/google/uuid"
	"github.com/hashmatter/p3lib/sphinx"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
)

const (
	// ProxyRelayProtocol is the address of the service used for in-game network
	ProxyRelayProtocol = "/pe2pe/0.0.1"
	// ProxyRelayDialTimeout is a maximum amount of time waited until aborting.
	// ... probably should be less
	ProxyRelayDialTimeout = time.Second * 15
	// PacketLen is a relay packet message size. @TODO: calculate it (may change)
	PacketLen = 1330
	// MagicWelcomeByte is the first thing sent over the encrypted e2e connection
	MagicWelcomeByte = 0x42
)

type connectionOpenRequest struct {
	Timestamp int64
	Port      uint32
	Key       [chacha20poly1305.KeySize]byte
	StreamID  uuid.UUID
}

// StartRelay starts the relay service
func (c *Client) StartRelay() error {

	// create relay context
	c.RelayCtx = sphinx.NewRelayerCtx(&c.Settings.Crypto.OnionKey)

	// bind listener
	c.Host.SetStreamHandler(ProxyRelayProtocol, func(s network.Stream) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Debugf("Got a new stream!")

		if err := c.serveRelayPackets(ctx, s); err != nil {
			log.Error("resetting the stream", err)
			_ = s.Reset()
		} else {
			log.Error("resetting the stream (no error)")
			_ = s.Reset()
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

	return connectstream.Connect(stream, s)
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

	if c.Settings.ExitNode == nil {
		return errors.New("Exit node is disabled")
	}

	dialTo := (*c.Settings.ExitNode)[fmt.Sprintf("%v", header.Port)]

	if dialTo == "" {
		return errors.Errorf("Port %v is not allowed", header.Port)
	}

	var dialer = &net.Dialer{}

	// open the local socket
	// @TODO: configurable ip addr
	localConn, err := dialer.DialContext(ctx, "tcp", dialTo)
	if err != nil {
		return errors.Wrap(err, "Failed to open local socket")
	}

	defer func() {
		log.Debugf("COCC closing conn")
		err := localConn.Close()
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

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
	return connectstream.Connect(secureConn, localConn)
}
