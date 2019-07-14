package main

import (
	"crypto/rand"
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
	return base64.StdEncoding.EncodeToString(b)
}

func main() {

	private, public, err := crypto.GenerateRSAKeyPair(2048, rand.Reader)
	if err != nil {
		fmt.Printf("Error while generating: %v\n", err)
		return
	}

	fmt.Printf("public: %v\nprivate: %v\n", toString(public), toString(private))
}
