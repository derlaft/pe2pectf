package main

import (
	"context"
	"flag"
	"os"

	"github.com/derlaft/pe2pectf/common"

	golog "github.com/ipfs/go-log"
	gologging "github.com/whyrusleeping/go-logging"
)

var log = golog.Logger("pe2pe_main") //nolint:gochecknoglobals

func main() {

	var settings = new(common.Settings)

	// service-related settings
	flag.StringVar(&settings.ListenAddr, "listen-relay", "0.0.0.0:4242", "Listen on (relay)")
	flag.StringVar(&settings.ProxyAddr, "listen-proxy", "0.0.0.0:9050", "Listen on (socks5 proxy")
	flag.StringVar(&settings.ExitNodeConfig, "exit-node-config", "", "Configuration file with service mappings")
	flag.StringVar(&settings.NetworkConfig, "network-config", "", "Configuration file with network map")
	flag.StringVar(&settings.CryptoConfig, "crypto-config", "", "Configuration file with client private crypto keys")

	// global debug mode
	debug := flag.Bool("debug", false, "enable verbose logging")

	// parse cli options
	flag.Parse()

	if settings.CryptoConfig == "" || settings.NetworkConfig == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *debug {
		golog.SetAllLoggers(gologging.DEBUG)
	} else {
		golog.SetAllLoggers(gologging.INFO)
	}

	ctx := context.Background()

	err := settings.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Loaded settings: %+v\n", settings)
	log.Debugf("Network settings: %+v\n", settings.Network)

	// Make a host that listens on the given multiaddress
	c, err := common.CreateHost(ctx, settings)
	if err != nil {
		log.Fatal(err)
	}

	err = c.ConnectDHT(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = c.StartRelay()
	if err != nil {
		log.Fatal(err)
	}

	// start proxy if requested
	if settings.ProxyAddr != "" {
		err = c.StartProxy()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Infof("listening for connections (addr is %v)", c.HostAddress())
	select {} // hang forever
}
