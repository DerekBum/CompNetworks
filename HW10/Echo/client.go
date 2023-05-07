package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

var host = flag.String("host", "akamai.com", "Host for ICMP requesting")

const (
	icmpEchoRequest = 8
	icmpEchoReply   = 0
	ipHeaderSize    = 20
)

type icmpHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	Id       uint16
	Seq      uint16
}

func main() {
	flag.Parse()

	ipAddr, err := net.ResolveIPAddr("ip", *host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("PING %s (%s):\n", *host, ipAddr)

	// create a raw socket to send ICMP packets
	conn, err := net.DialIP("ip4:icmp", nil, ipAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	id := uint16(os.Getpid() & 0xffff)
	seq := uint16(0)
	payload := []byte("PingData")
	received := 0
	minRtt := time.Minute
	maxRtt := time.Duration(0)
	totalRtt := time.Duration(0)
	lostPackets := 0

	for {
		// send an ICMP echo request
		sendTime := time.Now()
		seq++
		header := icmpHeader{
			Type: icmpEchoRequest, Code: 0,
			Checksum: 0, Id: id, Seq: seq,
		}
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.BigEndian, header)
		binary.Write(&buffer, binary.BigEndian, payload)
		checksum := checksum(buffer.Bytes())
		header.Checksum = checksum
		buffer.Reset()
		binary.Write(&buffer, binary.BigEndian, header)
		binary.Write(&buffer, binary.BigEndian, payload)
		if _, err := conn.Write(buffer.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		// wait for an ICMP echo reply
		reply := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := conn.Read(reply)

		if err != nil {
			lostPackets++
			if err, ok := err.(net.Error); ok && err.Timeout() {
				fmt.Printf("Request timeout for icmp_seq %d\n", seq)
			} else {
				fmt.Printf("Error receiving packet icmp_seq %d: %s\n", seq, err)
				time.Sleep(time.Second)
			}
			continue
		}
		rtt := time.Since(sendTime)
		if rtt < minRtt {
			minRtt = rtt
		}
		if rtt > maxRtt {
			maxRtt = rtt
		}
		totalRtt += rtt
		headerReply := icmpHeader{}
		buffer = bytes.Buffer{}
		binary.Read(bytes.NewReader(reply[ipHeaderSize:ipHeaderSize+8]), binary.BigEndian, &headerReply)
		fmt.Printf("--------------------------------------------------------------------------------\n")

		if !checkChecksum(headerReply, reply[ipHeaderSize+8:n]) {
			fmt.Printf("ERROR: bad checksum\ttime=%v\n", rtt)
			lostPackets++
		} else {

			if headerReply.Type == icmpEchoReply {
				received++
				if headerReply.Id != header.Id {
					continue
				}

				fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n", n, ipAddr, headerReply.Seq, rtt)
				fmt.Printf("\n--- %s ping statistics ---\n", *host)
			} else {
				switch headerReply.Code {
				case 0:
					fmt.Println("ICMP ERROR: Destination network unreachable")
				case 1:
					fmt.Println("ICMP ERROR: Destination host unreachable")
				case 2:
					fmt.Println("ICMP ERROR: Protocol unreachable")
				case 3:
					fmt.Println("ICMP ERROR: Destination port unreachable")
				default:
					fmt.Printf("ICMP ERROR: Code %d\n", headerReply.Code)
				}
				lostPackets++
			}
		}

		lossPercent := float64(lostPackets) / float64(received+lostPackets) * 100.0
		avgRtt := totalRtt / time.Duration(received+lostPackets)
		fmt.Printf("%d packets transmitted, %d packets received, %.3f%% packet loss\n", received+lostPackets, received, lossPercent)
		fmt.Printf("round-trip min/avg/max = %v/%v/%v\n", minRtt, avgRtt, maxRtt)

		fmt.Printf("--------------------------------------------------------------------------------\n")
		time.Sleep(time.Second)
	}
}

func checksum(data []byte) uint16 {
	var sum uint32
	length := len(data)
	for i := 0; i < length-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if length%2 == 1 {
		sum += uint32(data[length-1]) << 8
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return uint16(^sum)
}

func checkChecksum(header icmpHeader, payload []byte) bool {
	got := header.Checksum
	header.Checksum = 0

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, header)
	binary.Write(&buff, binary.BigEndian, payload)

	control := checksum(buff.Bytes())

	return control == got
}
