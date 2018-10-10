package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackpal/gateway"
	"github.com/sparrc/go-ping"
)

func runPing(addr string) (p *ping.Pinger, err error) {
	p, err = ping.NewPinger(addr)

	if err != nil {
		return
	}

	p.SetPrivileged(true)
	p.Interval = time.Second
	p.Count = -1

	go p.Run()

	return
}

func printStats(name string, s *ping.Statistics) {
	fmt.Printf(
		"\n\nStatistics for %s (%d packets sent):\n\t- Packet Loss: %v%%\n\t- Avg. RTT (ms): %v\n\t- Min. RTT (ms): %v\n\t- Max. RTT (ms): %v\n\t- Std. Dev. RTT (ms): %v\n",
		name, s.PacketsSent, s.PacketLoss, s.AvgRtt, s.MinRtt, s.MaxRtt, s.StdDevRtt)
}

func main() {
	fmt.Println("Attempting to resolve the default gateway...")

	dg, gatewayErr := gateway.DiscoverGateway()

	if gatewayErr != nil {
		fmt.Printf("Could not resolve gateway: %s\n", gatewayErr)
		os.Exit(1)
	}

	fmt.Printf("Default gateway resolved to %s\n", dg.String())

	sites := flag.Args()
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

		fmt.Println("Default site resolved, pinging default gateway and the default site.")
		valid = []string{"google.com"}
	}

	if dgPing, dgPErr := runPing(dg.String()); dgPErr != nil {
		fmt.Printf("Couldn't set up pinger for the Default Gateway:\n\t`%s`\nExiting...", dgPErr)
		os.Exit(1)
	} else {
		pingers := []*ping.Pinger{}
		for _, site := range valid {
			if p, err := runPing(site); err != nil {
				fmt.Printf("Couldn't set up pinger for %s:\n\t`%s`\nExiting...", site, err)
				os.Exit(1)
			} else {
				pingers = append(pingers, p)
			}
		}

		reportStats := func() {
			dgName := fmt.Sprintf("Default Gateway (%s)", dg.String())
			printStats(dgName, dgPing.Statistics())

			for i, pinger := range pingers {
				printStats(valid[i], pinger.Statistics())
			}
		}

		defer reportStats()
	}
}
