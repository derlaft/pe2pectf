package common

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	"gopkg.in/ini.v1"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

type encodeCryptoSettings struct {
	Key      string
	OnionKey string
}

func (cs *CryptoSettings) UnmarshalJSON(input []byte) error {

	var ec encodeCryptoSettings

	err := json.Unmarshal(input, &ec)
	if err != nil {
		return errors.Wrap(err, "decoding key object")
	}

	rsa, err := base64.StdEncoding.DecodeString(ec.Key)
	if err != nil {
		return errors.Wrap(err, "decoding rsa key (base64)")
	}

	rsaDecoded, err := crypto.UnmarshalPrivateKey(rsa)
	if err != nil {
		return errors.Wrap(err, "decoding rsa key")
	}

	cs.Key = rsaDecoded

	ecdsa, err := base64.StdEncoding.DecodeString(ec.OnionKey)
	if err != nil {
		return errors.Wrap(err, "decoding ecdsa private key")
	}

	ecdsaP, err := x509.ParseECPrivateKey(ecdsa)
	if err != nil {
		return errors.Wrap(err, "parsing ecdsa private key")
	}

	cs.OnionKey = *ecdsaP

	return nil
}

type encodeMembers struct {
	DHT  DHTSettings
	Node map[string]encodeMember
}

type encodeMember struct {
	Address  string
	Key      string
	OnionKey string
	Trusted  bool
}

func LoadNetworkSettings(fname string) (*NetworkSettings, error) {

	cfg, err := ini.Load(fname)
	if err != nil {
		return nil, err
	}

	var dec encodeMembers
	err = cfg.MapTo(&dec)
	if err != nil {
		return nil, err
	}

	var output = NetworkSettings{
		DHT: dec.DHT,
	}

	for addr, data := range dec.Node {

		peerID, err := peer.IDB58Decode(addr)
		if err != nil {
			return nil, err
		}

		keyBytes, err := base64.StdEncoding.DecodeString(data.OnionKey)
		if err != nil {
			return nil, errors.Wrap(err, "decoding ecdsa public key")
		}

		ecdsaP, err := x509.ParsePKIXPublicKey(keyBytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing ecdsa public key")
		}

		ecdsaPK, ok := ecdsaP.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.Wrapf(err, "casting ecdsa public key (unknown type %T)", ecdsaP)

		}

		output.Nodes[peerID] = Member{
			Address:      data.Address,
			OnionKey:     *ecdsaPK,
			TrustedRelay: data.Trusted,
		}
	}

	return &output, nil
}
