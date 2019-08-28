package common

import (
	"encoding/json"
	"io/ioutil"

	"crypto/ecdsa"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
)

type Settings struct {
	NodeCluster NodeClusterSettings // always required
	Crypto      CryptoSettings      // always required
	Routing     *RoutingSettings    // required for teams
	Proxy       *ProxySettings      // required for teams
	ExitNode    *ExitNodeSettings   // required for teams
}

type NodeClusterSettings struct {
	ListenHost     string
	ListenPort     int
	BootstrapNodes []string
	NetworkID      string
}

type CryptoSettings struct {
	RSAPrivate   crypto.PrivKey
	ECDSAPrivate ecdsa.PrivateKey
}

type RoutingSettings struct {
	// Networks is a map of public keys -> allowed subnets
	Networks MembersMap
}

type MembersMap struct {
	Values map[core.PeerID]Member
}

type Member struct {
	Address     string
	ECDSAPublic ecdsa.PublicKey
}

type ProxySettings struct {
	Enabled    bool
	ListenAddr string
}

type ExitNodeSettings struct {
	Enabled bool
	// AllowedPorts defines which services are proxy-accessible
	AllowedPorts []int
}

func (ns *ExitNodeSettings) IsPortAllowed(id int) bool {

	if ns == nil {
		return false
	}

	for _, ap := range ns.AllowedPorts {
		if ap == id {
			return true
		}
	}

	return false
}

func SettingsFromFile(fname string) (*Settings, error) {

	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	var settings Settings

	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to decode config")
	}

	return &settings, nil
}

func (r *RoutingSettings) PeerIDForAddr(addr string) (core.PeerID, bool) {

	if r == nil || r.Networks.Values == nil {
		return "", false
	}

	for peer, info := range r.Networks.Values {
		if info.Address == addr {
			return peer, true
		}
	}

	return "", false
}
