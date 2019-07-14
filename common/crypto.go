package common

import (
	"encoding/base64"
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
)

type encodeCryptoSettings struct {
	RSAPrivate string
}

// map rsa private keys to ecdsa private keys
type PublicKeys map[string]string

func (cs *CryptoSettings) MarshalJSON() ([]byte, error) {

	var ec encodeCryptoSettings

	rsa, err := cs.RSAPrivate.Bytes()
	if err != nil {
		return nil, err
	}

	ec.RSAPrivate = base64.StdEncoding.EncodeToString(rsa)

	return json.Marshal(ec)
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

	return nil
}
