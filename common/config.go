package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"crypto/ecdsa"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type Settings struct {
	// server will be listening on this addr (relay && connectivity)
	ListenAddr string
	// if not empty, socks5 proxy will be listening on this addr (entry point into the game network)
	ProxyAddr string

	// exit node config (hosted services)
	ExitNodeConfig string
	ExitNode       *ExitNodeSettings

	// network config (info about all nodes in the network)
	NetworkConfig string
	Network       *NetworkSettings

	// private keys
	CryptoConfig string
	Crypto       *CryptoSettings
}

type ExitNodeSettings map[string]string

type NetworkSettings struct {
	DHT   DHTSettings
	Nodes map[core.PeerID]Member
}

type DHTSettings struct {
	Bootstrap []string
	NetworkID string
}

type CryptoSettings struct {
	Key      crypto.PrivKey
	OnionKey ecdsa.PrivateKey
}

type Member struct {
	Address      string
	OnionKey     ecdsa.PublicKey
	TrustedRelay bool
}

func (ns *ExitNodeSettings) IsPortAllowed(id int) bool {

	if ns == nil {
		return false
	}

	var key = strconv.Itoa(id)
	_, allowed := (*ns)[key]
	return allowed
}

func (s *Settings) Load() error {

	// load network config
	{
		if s.NetworkConfig == "" {
			return fmt.Errorf("network map not provided")
		}

		cfg, err := LoadNetworkSettings(s.NetworkConfig)
		if err != nil {
			return err
		}

		s.Network = cfg
	}

	// load exit-node config
	if s.ExitNodeConfig > "" {

		nc := new(NetworkSettings)

		bytes, err := ioutil.ReadFile(s.ExitNodeConfig)
		if err != nil {
			return err
		}

		err = json.Unmarshal(bytes, nc)
		if err != nil {
			return err
		}

		s.Network = nc
	}

	// load private crypto keys
	{
		cs := new(CryptoSettings)
		if s.CryptoConfig == "" {
			return fmt.Errorf("crypto keys not provided")
		}

		bytes, err := ioutil.ReadFile(s.CryptoConfig)
		if err != nil {
			return err
		}

		err = json.Unmarshal(bytes, cs)
		if err != nil {
			return err
		}

		s.Crypto = cs
	}

	return nil
}

func (ns *NetworkSettings) PeerIDForAddr(addr string) (core.PeerID, bool) {

	if ns == nil || ns.Nodes == nil {
		return "", false
	}

	for peer, info := range ns.Nodes {
		if info.Address == addr {
			return peer, true
		}
	}

	return "", false
}
