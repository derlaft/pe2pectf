package common

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/derlaft/pe2pectf/go-socks5"
	"github.com/pkg/errors"
)

// StartProxy binds a local port for proxy
func (c *Client) StartProxy() error {

	conf := &socks5.Config{
		Dial:     c.Dial,
		Resolver: c,
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	go func() {
		err := server.ListenAndServe("tcp", c.Settings.ProxyAddr)
		if err != nil {
			log.Fatal(err)
		}

	}()

	return nil
}

// Resolve virtual IP in the game network
func (c *Client) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {

	for _, client := range c.Settings.Network.Nodes {
		if client.ID == name || client.Address == name {
			return ctx, net.ParseIP(client.Address), nil
		}
	}

	log.Errorf("Could not resolve addr %v", name)

	// @TODO: actually implement some resolving
	return nil, nil, errors.New("not found")
}

// Dial some host in a virtual network
func (c *Client) Dial(ctx context.Context, network, addr string) (net.Conn, error) {

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("Error parsing network addr")
	}

	if addr == c.Settings.Network.Nodes[c.Settings.Crypto.ID].Address {
		dialer := net.Dialer{}

		if c.Settings.ExitNode == nil {
			return nil, errors.New("Exit node is disabled")
		}

		dialTo := (*c.Settings.ExitNode)[port]
		if dialTo == "" {
			return nil, errors.Errorf("Port %v is disabled", port)
		}

		localConn, err := dialer.DialContext(ctx, "tcp", dialTo)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to open local socket")
		}

		return localConn, nil
	}

	portValue, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("Error parsing network addr port")
	}

	peerID, found := c.Settings.Network.PeerIDForAddr(host)
	if !found {
		return nil, errors.Errorf("Addr %v is not in static routing table", addr)
	}

	// in case of this node
	if peerID == c.Host.ID() && c.Settings.ExitNode != nil {

		log.Debugf("A new local connection to port %v", portValue)

		// just dial addr locally - it's on this node
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%v", portValue))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to open local socket")
		}

		return conn, nil

	} else if peerID == c.Host.ID() {
		return nil, errors.Errorf("This relay does not host anything")
	}

	log.Debugf("A new remote connection to %v:%v", peerID, portValue)

	conn, err := c.OnionDial(ctx, network, peerID, portValue)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
