package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"time"
)

var port = flag.String("port", ":8081", "Port of the localhost server. Example: \":8081\"")
var interval = flag.Int("time", 3, "Heart rate of client (in seconds)")

func main() {
	flag.Parse()
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost"+*port)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error connecting to UDP server:", err)
		return
	}
	defer conn.Close()

	var rttTotal, rttMin, rttMax float64
	rttMin = 1e9
	var packetsSent, packetsReceived int

	sequenceNum := 0

	for {
		sequenceNum++

		timestamp := time.Now().UnixNano()
		msg := fmt.Sprintf("ping %d %d", sequenceNum, timestamp)
		buf := []byte(msg)

		_, err = conn.Write(buf)
		if err != nil {
			fmt.Println("Error sending UDP packet:", err)
			continue
		}
		packetsSent++

		time.Sleep(time.Duration(*interval) * time.Second)
		start := time.Now()

		buf = make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				fmt.Println("Request timed out")
				// break
			} else {
				fmt.Println("Error reading UDP packet:", err)
			}
			continue
		}

		end := time.Now()
		rtt := end.Sub(start).Seconds()

		response := string(buf[:n])

		rttTotal += rtt
		packetsReceived++
		if rtt < rttMin {
			rttMin = rtt
		}
		if rtt > rttMax {
			rttMax = rtt
		}

		fmt.Printf("Received response for Sequence %d after %.9f seconds: %s\n", sequenceNum, rtt, response)
	}

	packetLoss := float64(packetsSent-packetsReceived) / float64(packetsSent) * 100.0
	rttAvg := rttTotal / float64(packetsReceived)
	fmt.Printf("\n--- Ping statistics ---\n%d packets transmitted, %d received, %.2f%% packet loss, time %.9fs\nrtt min/avg/max = %.9f/%.9f/%.9f\n",
		packetsSent, packetsReceived, packetLoss, rttTotal, rttMin, rttAvg, rttMax)
}
