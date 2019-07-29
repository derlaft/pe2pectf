package common

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
)

type encodeCryptoSettings struct {
	RSAPrivate   string
	ECDSAPrivate string
	NetworkMap   map[string]string
}

func (cs *CryptoSettings) UnmarshalJSON(input []byte) error {

	var ec encodeCryptoSettings

	err := json.Unmarshal(input, &ec)
	if err != nil {
		return errors.Wrap(err, "decoding key object")
	}

	rsa, err := base64.StdEncoding.DecodeString(ec.RSAPrivate)
	if err != nil {
		return errors.Wrap(err, "decoding rsa key (base64)")
	}

	rsaDecoded, err := crypto.UnmarshalPrivateKey(rsa)
	if err != nil {
		return errors.Wrap(err, "decoding rsa key")
	}

	cs.RSAPrivate = rsaDecoded

	ecdsa, err := base64.StdEncoding.DecodeString(ec.ECDSAPrivate)
	if err != nil {
		return errors.Wrap(err, "decoding ecdsa private key")
	}

	ecdsaP, err := x509.ParseECPrivateKey(ecdsa)
	if err != nil {
		return errors.Wrap(err, "parsing ecdsa private key")
	}

	cs.ECDSAPrivate = *ecdsaP

	return nil
}

type encodeMembersMap map[string]encodeMember

type encodeMember struct {
	Address     string
	ECDSAPublic string
}

func (mm *MembersMap) UnmarshalJSON(input []byte) error {

	mm.Values = make(map[core.PeerID]Member)

	var dec encodeMembersMap
	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	for addr, data := range dec {

		peerID, err := peer.IDB58Decode(addr)
		if err != nil {
			return err
		}

		keyBytes, err := base64.StdEncoding.DecodeString(data.ECDSAPublic)
		if err != nil {
			return errors.Wrap(err, "decoding ecdsa public key")
		}

		ecdsaP, err := x509.ParsePKIXPublicKey(keyBytes)
		if err != nil {
			return errors.Wrap(err, "parsing ecdsa public key")
		}

		ecdsaPK, ok := ecdsaP.(*ecdsa.PublicKey)
		if !ok {
			return errors.Wrapf(err, "casting ecdsa public key (unknown type %T)", ecdsaP)

		}

		mm.Values[peerID] = Member{
			Address:     data.Address,
			ECDSAPublic: *ecdsaPK,
		}

	}

	return nil
}
