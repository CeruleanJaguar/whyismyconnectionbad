package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackpal/gateway"
	"github.com/sparrc/go-ping"
)

func main() {
	fmt.Println("Attempting to resolve the default gateway...")

	dg, gatewayErr := gateway.DiscoverGateway()

	if gatewayErr != nil {
		fmt.Printf("Could not resolve gateway: %s\n", gatewayErr)
		os.Exit(1)
	}

	fmt.Printf("Default gateway resolved to %s\n", dg.String())

	sites := os.Args[1:]
	var valid []string

	for i, site := range sites {
		addrs, netErr := net.LookupHost(site)

		if netErr != nil {
			fmt.Printf("%d. Could not resolve %s and therefore it will be ignored.\n", i+1, site)
		} else {
			fmt.Printf("%d. Resolved %s to %s\n", i+1, site, strings.Join(addrs, ", "))
			valid = append(valid, site)
		}
	}

	if len(valid) > 0 {
		fmt.Println("\nPinging default gateway and the following sites:\n ->", strings.Join(valid, "\n -> "))
	} else {
		fmt.Println("\nNo valid sites specified, trying default external site (google.com)...")
		_, netErr := net.LookupHost("google.com")

		if netErr != nil {
			fmt.Println("Cannot resolve default external site (google.com), exiting...")
			os.Exit(1)
		}
	}

	fmt.Println("Default site resolved, pinging default gateway and the default site.")
	valid = []string{"google.com"}

	dgPing, dgErr := ping.NewPinger(dg.String())

	if dgErr != nil {
		fmt.Printf("Error creating DG ping: %s", dgErr)
		os.Exit(1)
	}

	dgPing.SetPrivileged(true)
	dgPing.Interval = time.Second
	dgPing.Count = -1
}
