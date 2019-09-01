package common

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/derlaft/connectstream"
	"github.com/google/uuid"
	"github.com/hashmatter/p3lib/sphinx"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
)

// CryptoHop represents one node in a chain of packet travelling path
type CryptoHop struct {
	HostID      core.PeerID
	ECDSAPublic ecdsa.PublicKey
}

// NumHops that each packet is required to travel
const NumHops = 2 // @TODO debug only

// GenPath creates a path of nodes that a packet could travel through
func (c *Client) GenPath(dest core.PeerID) ([]CryptoHop, error) {

	var (
		hops = NumHops - 1 // the last hop is dest
		ret  = make([]CryptoHop, 0, NumHops)
	)

	for addr, info := range c.Settings.Network.Nodes {

		// check return (here to allow 1 hops for testing purposes)
		if hops == 0 {
			break
		}

		// exclude own addr
		if addr == c.Host.ID() {
			continue
		}

		// exclude dest addr
		if dest == addr {
			continue
		}

		// decode && append next hop
		ret = append(ret, CryptoHop{
			ECDSAPublic: info.OnionKey,
			HostID:      addr,
		})

		// decrement counter
		hops--
	}

	destHopInfo, found := c.Settings.Network.Nodes[dest]
	if !found {
		return nil, errors.Errorf("dest hop %v not found in network map", dest)
	}

	// @TODO: append end addr
	ret = append(ret, CryptoHop{
		HostID:      dest,
		ECDSAPublic: destHopInfo.OnionKey,
	})

	if len(ret) != NumHops {
		return nil, fmt.Errorf("not enough hops: %v/%v", len(ret), NumHops)
	}

	return ret, nil
}

// ConstructRelayHeader returns an onion-wrapped welcome message with e2e encryption keys
func (c *Client) ConstructRelayHeader(hops []CryptoHop, payload [256]byte) ([]byte, error) {

	var (
		hopKeys  = make([]ecdsa.PublicKey, 0, len(hops))
		hopAddrs = make([][]byte, 0, len(hops))
	)

	// extract hop addresses
	for _, hop := range hops {
		hopKeys = append(hopKeys, hop.ECDSAPublic)

		dest, err := hop.HostID.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "HostID.MarshalBinary() failed")
		}

		hopAddrs = append(hopAddrs, dest)
	}

	// create a virtual identity
	vkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generating temporary key failed")
	}

	// craft a packet
	packet, err := sphinx.NewPacket(
		vkey,
		hopKeys,
		[]byte{}, // final addr not needed
		hopAddrs, // @TODO: all relay addrs
		payload,
	)
	if err != nil {
		return nil, err
	}

	var buf = &bytes.Buffer{}
	err = gob.NewEncoder(buf).Encode(packet)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// OnionDial establishs an encrypted onion e2e-connection
func (c *Client) OnionDial(ctx context.Context, proto string, host core.PeerID, port int) (net.Conn, error) {

	// validate network
	if proto != "tcp" {
		return nil, errors.Errorf("Protocol %v is not supported", proto)
	}

	// construct onion chain
	chain, err := c.GenPath(host)
	if err != nil {
		return nil, err
	}

	// create payload && connection request
	var (
		payload [256]byte
		request = connectionOpenRequest{
			Port:      uint32(port),
			Timestamp: time.Now().Unix(),
			StreamID:  uuid.New(),
		}
		buf = &bytes.Buffer{}
	)

	{ // log the packet path
		var debugPath []string
		for _, hop := range chain {
			debugPath = append(debugPath, hop.HostID.Pretty())
		}
		log.Debugf("Packet %v will travel the path: %v", request.StreamID, debugPath)
	}

	// generate session key
	_, err = rand.Read(request.Key[:])
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to get some random bytes (stream=%v)", request.StreamID)
	}

	// write payload to buffer
	err = binary.Write(buf, binary.BigEndian, request)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to send welcome message (stream=%v)", request.StreamID)
	}

	copy(payload[:], buf.Bytes())

	// create header packet
	netPacket, err := c.ConstructRelayHeader(chain, payload)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to construct relay header (stream=%v)", request.StreamID)
	}

	log.Debugf("constructed packet with len %v", len(netPacket))

	var (
		connectionEstablished bool
		insideEnd, returnEnd  = net.Pipe()
	)

	// connect to the first peer
	stream, err := c.Host.NewStream(ctx, chain[0].HostID, ProxyRelayProtocol)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to dial first host of the chain (host=%v,stream=%v", chain[0].HostID, request.StreamID)
	}

	// establish e2e encryption (no data sent yet)
	e2e, err := NewCryptoReadWriter(stream, request.Key[:])
	if err != nil {
		return nil, errors.Wrapf(err,
			"while establishing e2e encryption (stream=%v)", request.StreamID)
	}

	defer func() {

		// connection is not fully open - error occurred
		if !connectionEstablished {
			log.Debugf("Connection not established, closing (stream=%v)", request.StreamID)

			err := stream.Close()
			if err != nil {
				log.Errorf("Failed closing the stream (stream=%v): %v",
					request.StreamID, err)
			}
			return
		}

		// connection fully open - connect the pipe
		// @TODO
		go func() {

			log.Debugf("connecting dial stuff (steam=%v)", request.StreamID)
			err := connectstream.Connect(e2e, insideEnd)
			if err != nil {
				log.Errorf("Error with connected stream (stream=%v): %v",
					request.StreamID, err)
			}
			log.Debugf("connecting cleanup stuff (stream=%v)", request.StreamID)

			err = stream.Close()
			if err != nil {
				log.Errorf("Error while closing connected stream (stream=%v): %v",
					request.StreamID, err)
			}

			err = insideEnd.Close()
			if err != nil {
				log.Errorf("Error while closing insideEnd (stream=%v): %v",
					request.StreamID, err)
			}

		}()
	}()

	// send header packet (to non-encrypted stream)
	_, err = stream.Write(netPacket)
	if err != nil {
		return nil, err
	}

	log.Debugf("sent welcome (stream=%v)", request.StreamID)

	// read magic number
	var respBuf [1]byte
	_, err = e2e.Read(respBuf[:])
	if err != nil {
		return nil, errors.Errorf("failed to read magic byte (stream=%v): %v", request.StreamID, err)
	}

	log.Debugf("got magic (stream=%v)", request.StreamID)

	// check magic byte
	if respBuf[0] != MagicWelcomeByte {
		return nil, errors.Errorf("bad magic response number %v (stream=%v)", respBuf[0], request.StreamID)
	}

	// establish encrypted connection
	connectionEstablished = true
	return returnEnd, nil
}
