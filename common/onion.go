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

	"github.com/hashmatter/p3lib/sphinx"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
)

type CryptoHop struct {
	HostID      core.PeerID
	ECDSAPublic ecdsa.PublicKey
}

const NumHops = 2 // @TODO debug only

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

func (c *Client) OnionDial(ctx context.Context, proto string, host core.PeerID, port int) (net.Conn, error) {

	// validate network
	if proto != "tcp" {
		return nil, errors.New("Only TCP is supported")
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
		}
		buf = &bytes.Buffer{}
	)

	// generate session key
	_, err = rand.Read(request.Key[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get some random bytes")
	}

	// write payload to buffer
	err = binary.Write(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	copy(payload[:], buf.Bytes())

	// create header packet
	netPacket, err := c.ConstructRelayHeader(chain, payload)
	if err != nil {
		return nil, err
	}

	log.Debugf("constructed packet with len %v", len(netPacket))

	var (
		connectionEstablished bool
		insideEnd, returnEnd  = net.Pipe()
	)

	// connect to the first peer
	stream, err := c.Host.NewStream(ctx, chain[0].HostID, ProxyRelayProtocol)
	if err != nil {
		return nil, err
	}

	// establish e2e encryption (no data sent yet)
	e2e, err := NewCryptoReadWriter(stream, request.Key[:])
	if err != nil {
		return nil, errors.Wrap(err, "while establishing e2e encryption")
	}

	defer func() {

		// connection is not fully open - error occurred
		if !connectionEstablished {
			err := stream.Close()
			if err != nil {
				log.Errorf("Failed closing the stream: %v", err)
			}
			return
		}

		// connection fully open - connect the pipe
		// @TODO
		go func() {

			log.Debugf("connecting dial stuff")
			err := connectStream(insideEnd, e2e)
			if err != nil {
				log.Errorf("Error with connected stream: %v", err)
			}
		}()
	}()

	// send header packet
	_, err = stream.Write(netPacket)
	if err != nil {
		return nil, err
	}

	log.Debugf("sent welcome")

	// read magic number
	var respBuf [1]byte
	_, err = e2e.Read(respBuf[:])
	if err != nil {
		return nil, err
	}

	log.Debugf("got magic")

	// check magic byte
	if respBuf[0] != MagicWelcomeByte {
		return nil, errors.Errorf("bad magic response number %v", respBuf[0])
	}

	// establish encrypted connection
	connectionEstablished = true
	return returnEnd, nil
}
