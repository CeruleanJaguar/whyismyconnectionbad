package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackpal/gateway"
	"github.com/nsf/termbox-go"
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

func printTb(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}

func printfTb(x, y int, fg, bg termbox.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	printTb(x, y, fg, bg, s)
}

func printStats(x, y int, name string, s *ping.Statistics) {
	printfTb(
		x, y+2, termbox.ColorDefault, termbox.ColorDefault,
		"Statistics for %s (%d packets sent):",
		name, s.PacketsSent)
	printfTb(
		x+2, y+3, termbox.ColorDefault, termbox.ColorDefault,
		"- Packet Loss: %v%%", s.PacketLoss)
	printfTb(
		x+2, y+4, termbox.ColorDefault, termbox.ColorDefault,
		"- Avg. RTT (ms): %v", s.AvgRtt)
	printfTb(
		x+2, y+5, termbox.ColorDefault, termbox.ColorDefault,
		"- Min. RTT (ms): %v", s.MinRtt)
	printfTb(
		x+2, y+6, termbox.ColorDefault, termbox.ColorDefault,
		"- Max. RTT (ms): %v", s.MaxRtt)
	printfTb(
		x+2, y+7, termbox.ColorDefault, termbox.ColorDefault,
		"- Std. Dev. RTT (ms): %v", s.StdDevRtt)
}

func keyCommand(key termbox.Key) (exit, stats bool) {
	exit = (key == termbox.KeyCtrlC)

	stats = (key == termbox.KeySpace)
	return
}

func end(ns []string, ps []*ping.Pinger) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.Close()

	fmt.Println("Test ended! Displaying final stats...")

	for i, p := range ps {
		name, s := ns[i], p.Statistics()
		fmt.Printf(
			"\n\nStatistics for %s (%d packets sent):\n\t- Packet Loss: %v%%\n\t- Avg. RTT (ms): %v\n\t- Min. RTT (ms): %v\n\t- Max. RTT (ms): %v\n\t- Std. Dev. RTT (ms): %v\n",
			name, s.PacketsSent, s.PacketLoss, s.AvgRtt, s.MinRtt, s.MaxRtt, s.StdDevRtt)
	}
}

func main() {
	tbErr := termbox.Init()
	if tbErr != nil {
		fmt.Println(tbErr)
		os.Exit(1)
	}
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	tbLine := 0
	printTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Attempting to resolve the default gateway...")
	tbLine++

	dg, gatewayErr := gateway.DiscoverGateway()

	if gatewayErr != nil {
		printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Could not resolve gateway: %s", gatewayErr)
		os.Exit(1)
	}

	printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Default gateway resolved to %s", dg.String())
	tbLine++

	sites := flag.Args()
	var valid []string

	for i, site := range sites {
		addrs, netErr := net.LookupHost(site)

		if netErr != nil {
			printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "%d. Could not resolve %s and therefore it will be ignored.", i+1, site)
			tbLine++
		} else {
			printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "%d. Resolved %s to %s", i+1, site, strings.Join(addrs, ", "))
			tbLine++
			valid = append(valid, site)
		}
	}

	if len(valid) > 0 {
		printTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Pinging default gateway and the following sites: ->")
		tbLine++
		for _, name := range valid {
			printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, " -> %s", name)
			tbLine++
		}
	} else {
		printTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "No valid sites specified, trying default external site (google.com)...")
		tbLine++
		_, netErr := net.LookupHost("google.com")

		if netErr != nil {
			printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Cannot resolve default external site (google.com), exiting...")
			tbLine++
			os.Exit(1)
		}

		fmt.Println("Default site resolved, pinging default gateway and the default site.")
		tbLine++
		valid = []string{"google.com"}
	}

	if dgPing, dgPErr := runPing(dg.String()); dgPErr != nil {
		printfTb(0, tbLine, termbox.ColorDefault, termbox.ColorDefault, "Couldn't set up pinger for the Default Gateway: `%s` - Exiting...", dgPErr)
		tbLine++
		os.Exit(1)
	} else {
		pingers := []*ping.Pinger{dgPing}
		for _, site := range valid {
			if p, err := runPing(site); err != nil {
				printfTb(0, 0, termbox.ColorDefault, termbox.ColorDefault, "Couldn't set up pinger for %s: `%s` - Exiting...", site, err)
				tbLine++
				os.Exit(1)
			} else {
				pingers = append(pingers, p)
			}
		}
		
		dgName := fmt.Sprintf("Default Gateway (%s)", dg.String())
		valid = append([]string{dgName}, valid...)
		defer end(valid, pingers)

		reportStats := func() {
			// termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
			printTb(0, 0, termbox.ColorDefault, termbox.ColorDefault, "Press space to freeze stats, Ctrl+C to end...")

			for i, pinger := range pingers {
				printStats(7*i+2, 0, valid[i], pinger.Statistics())
			}
		}

		frozen, exit := false, false

		go func() {
		evtLoop:
			for {
				switch ev := termbox.PollEvent(); ev.Type {
				case termbox.EventKey:
					exitEvt, freeze := keyCommand(ev.Key)
					if exitEvt {
						exit = true
						break evtLoop
					}
					frozen = (freeze != frozen)
				case termbox.EventError:
					defer fmt.Printf("Termbox encountered an error: %s", ev.Err)
					os.Exit(1)
					break evtLoop
				}
			}
		}()

		for !exit {
			if !frozen {
				reportStats()
			}
		}
	}
}
