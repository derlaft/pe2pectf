module github.com/derlaft/pe2pectf

go 1.12

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/derlaft/p3lib v0.0.0-20190726232235-64eddbdc601b
	github.com/hashmatter/p3lib v0.0.0-20190725202400-f1d0c92114a3
	github.com/ipfs/go-log v0.0.1
	github.com/libp2p/go-libp2p v0.2.0
	github.com/libp2p/go-libp2p-core v0.0.6
	github.com/libp2p/go-libp2p-kad-dht v0.1.1
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc
	golang.org/x/crypto v0.0.0-20190618222545-ea8f1a30c443
)

replace github.com/hashmatter/p3lib => github.com/derlaft/p3lib v0.0.0-20190727000155-af54243638b5
