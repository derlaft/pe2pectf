package common

import (
	"encoding/json"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
)

type Settings struct {
	ConnectSettings
	Crypto  *CryptoSettings
	Routing *RoutingSettings
	Proxy   *ProxySettings
	Node    *NodeSettings
}

type ConnectSettings struct {
	ListenHost     string
	ListenPort     int
	BootstrapNodes []string
	NetworkID      string
	DoDiscovery    bool
	Relay          bool
}

type CryptoSettings struct {
	RSAPrivate crypto.PrivKey
}

type RoutingSettings struct {
	// Only these nodes are allowed to be routed
	TrustedRelays []string
	// Networks is a map of public keys -> allowed subnets
	Networks map[string]string
}

type ProxySettings struct {
	Enabled    bool
	ListenAddr string
}

type NodeSettings struct {
	Enabled bool
	// AllowedPorts defines which services are proxy-accessible
	AllowedPorts []int
}

func (ns *NodeSettings) IsPortAllowed(id int) bool {

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

func (r *RoutingSettings) PublicKeyForAddr(addr string) (string, bool) {

	if r == nil || r.Networks == nil {
		return "", false
	}

	addr, found := r.Networks[addr]
	return addr, found
}
