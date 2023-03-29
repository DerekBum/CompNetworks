package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var port = flag.String("port", ":8081", "Port of localhost server (starts with \":\")")

func main() {
	flag.Parse()

	fmt.Println("Starting server...")

	port, ok := strings.CutPrefix(*port, ":")
	if !ok {
		panic("wrong port argument")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		panic("wrong port argument")
	}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: portNum,
	})
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	fmt.Println("Server is broadcasting on", conn.LocalAddr().String())

	for {
		currentTime := time.Now().Format(time.RFC3339)
		conn.Write([]byte(currentTime))
		time.Sleep(time.Second)
	}
}
