package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashmatter/p3lib/sphinx"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
)

const (
	DiscoveryTimeout = time.Second * 60
	ConnectTimeout   = time.Second * 16
)

var log = golog.Logger("pe2pe_common") //nolint:gochecknoglobals

type Client struct {
	Host      host.Host
	Settings  *Settings
	Discovery routing.ContentRouting
	RelayCtx  *sphinx.RelayerCtx
}

func (c *Client) HostAddress() string {
	var id = c.Host.ID().Pretty()
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", id))
	addr := c.Host.Addrs()[0]
	return fmt.Sprintf("%v", addr.Encapsulate(hostAddr))
}

// createHost creates this new CTF host
func CreateHost(context context.Context, settings *Settings) (*Client, error) {

	listenAddr := fmt.Sprintf("/ip4/%s/tcp/%d",
		settings.ListenHost,
		settings.ListenPort,
	)

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Identity(settings.Crypto.RSAPrivate),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	if !settings.Relay {
		opts = append(opts, libp2p.DisableRelay())
	}

	basicHost, err := libp2p.New(context, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		Host:     basicHost,
		Settings: settings,
	}, nil
}

func (c *Client) ConnectDHT(ctx context.Context) error {

	// create DHT
	kademliaDHT, err := dht.New(ctx, c.Host)
	if err != nil {
		return err
	}

	// turn our host into a routed host
	c.Host = routedhost.Wrap(c.Host, kademliaDHT)

	// initialize
	err = kademliaDHT.Bootstrap(ctx)
	if err != nil {
		return err
	}

	c.Discovery = kademliaDHT

	// bootstrap (connect to initial nodes)
	err = c.bootstrapDHT(ctx)
	if err != nil {
		return err
	}

	return nil

}

func (c *Client) bootstrapDHT(ctx context.Context) error {

	// connect to all bootstrap nodes
	var wg sync.WaitGroup
	for _, peerAddr := range c.Settings.BootstrapNodes {

		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			return err
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}

		wg.Add(1)

		go func() {
			defer wg.Done()

			childContext, cancel := context.WithTimeout(ctx, ConnectTimeout)
			defer cancel()

			err := c.Host.Connect(childContext, *peerInfo)
			if err != nil {
				log.Warningf("Unable to connect to %v: %v", addr, err)
			} else {
				log.Infof("Connected to %v", addr)
			}

		}()
	}

	wg.Wait()

	return nil
}
