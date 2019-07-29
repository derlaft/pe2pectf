package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func toString(k crypto.Key) string {

	b, err := k.Bytes()
	if err != nil {
		log.Fatal(err)
	}
	return bytesToString(b)
}

func bytesToString(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func main() {
	genRSA()
	genECDSA()
}

func genECDSA() {

	fmt.Println("ecdsa")

	ecp, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Printf("Error while generating: %v\n", err)
		return
	}

	ecpB, err := x509.MarshalECPrivateKey(ecp)
	if err != nil {
		fmt.Printf("Error while marshalling: %v\n", err)
		return
	}

	ecpBP, err := x509.MarshalPKIXPublicKey(ecp.Public())
	if err != nil {
		fmt.Printf("Error while marshalling: %v\n", err)
		return
	}

	fmt.Printf("public: %v\nprivate: %v\n", bytesToString(ecpBP), bytesToString(ecpB))

}

func genRSA() {

	fmt.Println("rsa")

	private, public, err := crypto.GenerateRSAKeyPair(2048, rand.Reader)
	if err != nil {
		fmt.Printf("Error while generating rsa: %v\n", err)
		return
	}

	fmt.Printf("public: %v\nprivate: %v\n", toString(public), toString(private))

}
