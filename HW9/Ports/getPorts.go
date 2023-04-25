package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
)

var addr = flag.String("addr", "127.0.0.1", "Address to search available ports")
var lowerBound = flag.Int("lb", 0, "Lower bound of search interval")
var upperBound = flag.Int("ub", 65535, "Upper bound of search interval")

func main() {
	flag.Parse()

	ipAddress := *addr
	startPort := *lowerBound // starting port of range
	endPort := *upperBound   // ending port of range

	fmt.Printf("Available ports:\n")
	cnt := 0

	for port := startPort; port <= endPort; port++ {
		addr := net.JoinHostPort(ipAddress, strconv.Itoa(port))
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			cnt++
			fmt.Printf("%d, ", port)
		} else {
			conn.Close()
		}
	}
}
