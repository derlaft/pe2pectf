package main

import (
	"context"
	"flag"

	"github.com/derlaft/pe2pectf/common"

	golog "github.com/ipfs/go-log"
	gologging "github.com/whyrusleeping/go-logging"
)

var log = golog.Logger("pe2pe_main") //nolint:gochecknoglobals

func main() {

	// parse cli options
	settingsPath := flag.String("config", "server.json", "config file location")
	debug := flag.Bool("debug", false, "enable verbose logging")
	flag.Parse()

	if *debug {
		golog.SetAllLoggers(gologging.DEBUG)
	} else {
		golog.SetAllLoggers(gologging.INFO)
	}

	ctx := context.Background()

	settings, err := common.SettingsFromFile(*settingsPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Loaded settings: %+v\n", settings)

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
	if settings.Proxy != nil && settings.Proxy.Enabled {
		err = c.StartProxy()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Infof("listening for connections (addr is %v)", c.HostAddress())
	select {} // hang forever
}
