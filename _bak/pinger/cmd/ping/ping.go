package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
	"unsafe"

	probing "github.com/prometheus-community/pro-bing"
)

func main() {
	addrs, err := readAddrs(os.Stdin)
	if err != nil {
		log.Fatalf("can't read address list: %v", err)
	}

	pingers := make([]*probing.Pinger, 0, len(addrs))

	for _, addr := range addrs {
		pinger, err := probing.NewPinger(addr)
		if err != nil {
			log.Printf("can't NewPinger %s: %v", addr, err)
			continue
		}

		log.Printf("pinger: %+v", pinger)
		pinger.RecordRtts = false
		pinger.RecordTTLs = false

		pinger.OnRecv = func(pkt *probing.Packet) {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v now=%v\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, time.Now())
		}

		pinger.OnDuplicateRecv = func(pkt *probing.Packet) {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v (DUP!) now=%v\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.TTL, time.Now())
		}

		pinger.OnFinish = func(stats *probing.Statistics) {
			fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
			fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
				stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
			fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
				stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
		}

		pingers = append(pingers, pinger)
	}

	// Listen for Ctrl-C.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		for _, pinger := range pingers {
			pinger.Stop()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(pingers))

	for _, pinger := range pingers {
		pinger := pinger
		go func() {
			defer wg.Done()
			fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
			if err := pinger.Run(); err != nil {
				log.Printf("PING %s ERROR: %v", pinger.Addr(), err)
			}
		}()
	}

	wg.Wait()
}

func readAddrs(r io.Reader) ([]string, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	s := unsafe.String(unsafe.SliceData(buf), len(buf))
	list := strings.Split(s, "\n")

	for i := range list {
		list[i] = strings.TrimSpace(list[i])
	}

	if n := len(list); n > 0 && list[n-1] == "" {
		list = list[:n-1]
	}

	return list, nil
}
