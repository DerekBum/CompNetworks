package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"golang.org/x/net/ipv4"
)

var dest = flag.String("dst", "akamai.com", "Destination host name")
var retries = flag.Int("ret", 3, "Number of retries")
var tOut = flag.Int("time", 1, "Timeout in seconds")
var localIP = flag.String("ip", "no-ip", "Local IP to do traceroute")

type Packet struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
}

func main() {
	flag.Parse()

	timeout := time.Duration(*tOut) * time.Second

	addr, err := net.ResolveIPAddr("ip4", *dest)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		os.Exit(1)
	}

	var local string
	if *localIP == "no-ip" {
		var ip net.IP
		ifaces, _ := net.Interfaces()
		for _, i := range ifaces {
			addrs, _ := i.Addrs()
			for _, address := range addrs {
				switch v := address.(type) {
				case *net.IPNet:
					ip = v.IP
				}
			}
		}
		local = ip.String()
	} else {
		local = *localIP
	}

	conn, err := net.ListenPacket("ip4:icmp", local)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	defer conn.Close()
	ipv4Conn := ipv4.NewPacketConn(conn)
	defer ipv4Conn.Close()

	if err := ipv4Conn.SetControlMessage(ipv4.FlagTTL|ipv4.FlagDst|ipv4.FlagInterface|ipv4.FlagSrc, true); err != nil {
		fmt.Fprintf(os.Stderr, "Could not set options on the ipv4 socket: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Traceroute to %s (%s)\n", *dest, addr.String())

	for ttl := 1; ; ttl++ {
		msg := Packet{Type: uint8(ipv4.ICMPTypeEcho), Code: 0, ID: uint16(rand.Int()), Seq: uint16(ttl)}

		var buff bytes.Buffer
		binary.Write(&buff, binary.BigEndian, msg)
		msg.Checksum = checksum(buff.Bytes())

		gotToDst := false

		for i := 1; i <= *retries; i++ {
			ipv4Conn.SetTTL(ttl)
			start := time.Now()

			buff.Reset()
			binary.Write(&buff, binary.BigEndian, msg)

			if _, err := ipv4Conn.WriteTo(buff.Bytes(), nil, addr); err != nil {
				fmt.Printf("%d\t*\t%s\n", ttl, err)
				continue
			}

			ipv4Conn.SetReadDeadline(time.Now().Add(timeout))

			data := make([]byte, 1500)

			n, _, node, err := ipv4Conn.ReadFrom(data)

			if err != nil {
				if node != nil {
					nodeName, _ := net.LookupAddr(node.String())
					fmt.Printf("%d\t*\t%s\t(%s)\t%s\n", ttl, node.String(), nodeName, err)
				} else {
					fmt.Printf("%d\t*\t%s\n", ttl, err)
				}
				continue
			}

			nodeName, _ := net.LookupAddr(node.String())

			ans := Packet{}
			buff = bytes.Buffer{}
			payload := make([]byte, n-8)
			binary.Read(bytes.NewReader(data[:8]), binary.BigEndian, &ans)
			binary.Read(bytes.NewReader(data[8:n]), binary.BigEndian, &payload)

			if !checkChecksum(ans, payload) {
				fmt.Printf("%d\t*\t%s\t(%s)\tERROR: bad checksum\n", ttl, node.String(), nodeName)
				continue
			}

			if err != nil {
				fmt.Printf("%d\t*\t%s\t(%s)\t%s\n", ttl, node.String(), nodeName, err)
				continue
			}

			rtt := time.Since(start)

			if ans.Type == uint8(ipv4.ICMPTypeTimeExceeded) {
				fmt.Printf("%d\t*\t%s\t(%s)\t*\tRTT: %v\n", ttl, node.String(), nodeName, rtt)
			} else if ans.Type == uint8(ipv4.ICMPTypeEchoReply) {
				fmt.Printf("%d\t*\t%s\t(%s)\t*\tRTT: %v\n", ttl, node.String(), nodeName, rtt)
				fmt.Printf("End of trace: Got to destination\n")
				gotToDst = true
				break
			} else {
				fmt.Printf("%d\t*\t%s\t(%s)\t*\tOther echo response\n", ttl, node.String(), nodeName)
			}
		}
		fmt.Printf("\n")
		if gotToDst {
			break
		}
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

func checkChecksum(packet Packet, payload []byte) bool {
	got := packet.Checksum
	packet.Checksum = 0

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, packet)
	binary.Write(&buff, binary.BigEndian, payload)

	control := checksum(buff.Bytes())

	return control == got
}
