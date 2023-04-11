package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

var port = flag.String("port", ":8081", "Port of the localhost server. Example: \":8081\"")
var tOut = flag.Int("time", 8, "Client considered stopped if no packets received for this duration (in seconds)")

type client struct {
	address    *net.UDPAddr // UDP address of client
	lastUpdate time.Time    // Time when last packet was received from client
	lastSeqNum int
}

func main() {
	flag.Parse()
	timeout := time.Duration(*tOut) * time.Second

	clients := make(map[string]*client)

	serverAddr, err := net.ResolveUDPAddr("udp", *port)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}
	defer conn.Close()

	rand.Seed(time.Now().UnixNano())

	fmt.Println("UDP server listening on", serverAddr)

	for {
		buf := make([]byte, 1024)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		if rand.Intn(5) == 0 { // simulate 20% packet loss
			fmt.Println("Dropped packet from", addr)
			continue
		}

		clientID := addr.String()
		c, ok := clients[clientID]
		if !ok {
			// New client - add to map
			c = &client{address: addr, lastUpdate: time.Now(), lastSeqNum: 0}
			clients[clientID] = c
			fmt.Println("New client connected:", clientID)

			go func() {
				for {
					if time.Since(c.lastUpdate) > timeout {
						delete(clients, clientID)
						fmt.Printf("Client %s timed out - disconnected\n", clientID)
						break
					}
				}
			}()
		}

		// Update last packet received time
		c.lastUpdate = time.Now()

		// Parse sequence number and timestamp from packet
		seqNum, timestamp, err := parseHeartbeatPacket(buf[:n])
		if err != nil {
			fmt.Println("Error parsing heartbeat packet:", err)
			continue
		}

		// Calculate RTT and report to client
		rtt := time.Since(timestamp)
		fmt.Printf("Received heartbeat from %s, sequence number %d, RTT: %v\n", clientID, seqNum, rtt)
		if seqNum-c.lastSeqNum > 1 {
			fmt.Printf("We lost %d packet(s) from %s\n", seqNum-c.lastSeqNum-1, clientID)
		}
		c.lastSeqNum = seqNum

		msg := strings.ToUpper(string(buf[:n]))

		_, err = conn.WriteToUDP([]byte(msg), addr)
		if err != nil {
			fmt.Println("Error writing to UDP:", err)
			continue
		}

		/*
			fmt.Println("Received message from", addr, ":", string(buf[:n]), " (", n, "bytes)")
			fmt.Println("Sent message to", addr, ":", msg, " (", len(msg), "bytes)")
		*/
	}
}

func parseHeartbeatPacket(packet []byte) (seqNum int, timestamp time.Time, err error) {
	str := string(packet)
	parse := strings.Split(str, " ")
	if len(parse) != 3 {
		return 0, time.Time{}, fmt.Errorf("invalid packet length")
	}
	seqNum, err = strconv.Atoi(parse[1])
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid sequence number")
	}
	ts, err := strconv.ParseInt(parse[2], 10, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid sequence number")
	}
	timestamp = time.Unix(0, ts)
	return seqNum, timestamp, nil
}
