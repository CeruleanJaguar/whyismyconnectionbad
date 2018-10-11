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
		x, y+2, termbox.ColorWhite, termbox.ColorBlack,
		"Statistics for %s (%d packets sent):",
		name, s.PacketsSent)
	printfTb(
		x+2, y+3, termbox.ColorWhite, termbox.ColorBlack,
		"- Packet Loss: %v%%", s.PacketLoss)
	printfTb(
		x+2, y+4, termbox.ColorWhite, termbox.ColorBlack,
		"- Avg. RTT (ms): %v", s.AvgRtt)
	printfTb(
		x+2, y+5, termbox.ColorWhite, termbox.ColorBlack,
		"- Min. RTT (ms): %v", s.MinRtt)
	printfTb(
		x+2, y+6, termbox.ColorWhite, termbox.ColorBlack,
		"- Max. RTT (ms): %v", s.MaxRtt)
	printfTb(
		x+2, y+7, termbox.ColorWhite, termbox.ColorBlack,
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
			"\n\nStatistics for %s (%d packets sent):\n\t- Packet Loss: %v%% (%v packets)\n\t- Avg. RTT (ms): %v\n\t- Min. RTT (ms): %v\n\t- Max. RTT (ms): %v\n\t- Std. Dev. RTT (ms): %v\n",
			name, s.PacketsSent, s.PacketLoss, s.PacketsSent-s.PacketsRecv, s.AvgRtt, s.MinRtt, s.MaxRtt, s.StdDevRtt)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nPing test your default gateway and sites you specify. If none are specified, `google.com` is the default\nUsage:\n\t%s [-help] [addrs...]\n\tAddresses should not have a protocol prefix.\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	help := flag.Bool("help", false, "Shows the help dialogue")

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	tbErr := termbox.Init()
	if tbErr != nil {
		fmt.Println(tbErr)
		os.Exit(1)
	}
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	tbLine := 0
	printTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Attempting to resolve the default gateway...")
	termbox.Flush()
	tbLine++

	dg, gatewayErr := gateway.DiscoverGateway()

	if gatewayErr != nil {
		printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Could not resolve gateway: %s", gatewayErr)
		termbox.Flush()
		os.Exit(1)
	}

	printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Default gateway resolved to %s", dg.String())
	termbox.Flush()
	tbLine++

	sites := flag.Args()
	var valid []string

	for i, site := range sites {
		addrs, netErr := net.LookupHost(site)

		if netErr != nil {
			printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "%d. Could not resolve %s and therefore it will be ignored.", i+1, site)
			termbox.Flush()
			tbLine++
		} else {
			printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "%d. Resolved %s to %s", i+1, site, strings.Join(addrs, ", "))
			termbox.Flush()
			tbLine++
			valid = append(valid, site)
		}
	}

	if len(valid) > 0 {
		printTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Pinging default gateway and the following sites:")
		termbox.Flush()
		tbLine++
		for _, name := range valid {
			printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, " -> %s", name)
			termbox.Flush()
			tbLine++
		}
	} else {
		printTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "No valid sites specified, trying default external site (google.com)...")
		termbox.Flush()
		tbLine++
		_, netErr := net.LookupHost("google.com")

		if netErr != nil {
			printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Cannot resolve default external site (google.com), exiting...")
			termbox.Flush()
			tbLine++
			os.Exit(1)
		}

		printTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Default site resolved, pinging default gateway and the default site.")
		termbox.Flush()
		tbLine++
		valid = []string{"google.com"}
	}

	printTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Press Enter to start!")
	termbox.Flush()
	tbLine++

startupLoop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEnter {
				break startupLoop
			}
		case termbox.EventError:
			defer fmt.Printf("Termbox encountered an error: %s", ev.Err)
			os.Exit(1)
			break startupLoop
		}
	}

	if dgPing, dgPErr := runPing(dg.String()); dgPErr != nil {
		printfTb(0, tbLine, termbox.ColorWhite, termbox.ColorBlack, "Couldn't set up pinger for the Default Gateway: `%s` - Exiting...", dgPErr)
		tbLine++
		os.Exit(1)
	} else {
		pingers := []*ping.Pinger{dgPing}
		for _, site := range valid {
			if p, err := runPing(site); err != nil {
				printfTb(0, 0, termbox.ColorWhite, termbox.ColorBlack, "Couldn't set up pinger for %s: `%s` - Exiting...", site, err)
				termbox.Flush()
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
			termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
			printTb(0, 0, termbox.ColorWhite, termbox.ColorBlack, "Press space to freeze stats, Ctrl+C to end...")

			for i, pinger := range pingers {
				printStats(0, 7*i+2, valid[i], pinger.Statistics())
			}
			termbox.Flush()
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
