package common

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/armon/go-socks5"
	"github.com/pkg/errors"
)

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
		err := server.ListenAndServe("tcp", c.Settings.Proxy.ListenAddr)
		if err != nil {
			log.Fatal(err)
		}

	}()

	return nil
}

func (c *Client) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// @TODO: actually implement some resolving
	return nil, nil, errors.New("not implemented yet")
}

func (c *Client) Dial(ctx context.Context, network, addr string) (net.Conn, error) {

	// @TODO: dial of addr of the same node

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("Error parsing network addr")
	}

	portValue, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("Error parsing network addr port")
	}

	peerID, found := c.Settings.Routing.PeerIDForAddr(host)
	if !found {
		return nil, errors.Errorf("Addr %v is not in static routing table", addr)
	}

	// in case of this node
	if peerID == c.Host.ID() && c.Settings.ExitNode != nil && c.Settings.ExitNode.Enabled {

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

	conn, err := c.OnionDial(ctx, network, peerID, portValue)
	if err != nil {
		return nil, err
	}

	return StreamWrapper{
		Conn:   conn,
		Local:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: portValue},
		Remote: &net.TCPAddr{IP: net.ParseIP(addr), Port: portValue},
	}, nil
}

type StreamWrapper struct {
	net.Conn
	Local  *net.TCPAddr
	Remote *net.TCPAddr
}

func (sw StreamWrapper) LocalAddr() net.Addr {
	return sw.Local
}

func (sw StreamWrapper) RemoteAddr() net.Addr {
	return sw.Remote
}
