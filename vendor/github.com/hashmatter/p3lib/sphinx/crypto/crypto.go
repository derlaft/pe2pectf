package crypto

import (
	"crypto/ecdsa"
	ec "crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"github.com/aead/chacha20"
)

// TODO: initially, this implementation is using SHA256 as hashing function.
// implement for other digests

type Hash256 [sha256.Size]byte

// generates shared secret using ECDH protocol in an arbitrary curve. The shared
// secret is the hash of the resulting x coordinate of point after scalar
// multiplication between the a ECDSA key pair.
func GenerateECDHSharedSecret(pub *ecdsa.PublicKey, priv *ecdsa.PrivateKey) Hash256 {
	curvep := pub.Curve.Params()
	x, _ := curvep.ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	sk := sha256.Sum256(x.Bytes())
	return sk
}

// computes blinding factor used for blinding the cyclic group element at each
// hop. The blinding factor is computed by hashing the concatenation of the the
// hop's public key and the secret key derived between the sender and the hop
// blinding_factor := sha256(hopPubKey || sharedSecret)
func ComputeBlindingFactor(pubKey *ecdsa.PublicKey, secret Hash256) Hash256 {
	mPubKey := serializePubKey(pubKey)
	sha := sha256.New()
	sha.Write(mPubKey)
	sha.Write(secret[:])

	var hash Hash256
	copy(hash[:], sha.Sum(nil))
	return hash
}

func serializePubKey(pub *ecdsa.PublicKey) []byte {
	return ec.Marshal(ec.P256(), pub.X, pub.Y)
}

func GetCurve(priv ecdsa.PrivateKey) ec.Curve {
	return priv.PublicKey.Curve
}

// computes HMAC-SHA-256
func ComputeMAC(key Hash256, message []byte) []byte {
	mac := hmac.New(sha256.New, key[:])
	mac.Write(message)
	return mac.Sum(nil)
}

// checks HMAC-SHA-256
func CheckMAC(message, messageMAC []byte, key Hash256) bool {
	mac := hmac.New(sha256.New, key[:])
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

// generates cipher stream of size numBytes from a given PRG. TODO: generalize
// to other ciphers
func GenerateCipherStream(key, nonce []byte, numBytes int) ([]byte, error) {
	c, err := chacha20.NewCipher(nonce, key)
	if err != nil {
		return []byte{}, err
	}
	out := make([]byte, numBytes)
	c.XORKeyStream(out, out)
	return out, nil
}
