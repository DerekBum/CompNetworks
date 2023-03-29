package main

import (
	"flag"
	"fmt"

	"github.com/libp2p/go-reuseport"
)

var port = flag.String("port", ":8081", "Port of localhost server (starts with \":\")")

func main() {
	flag.Parse()

	fmt.Println("Starting client...")

	conn, _ := reuseport.ListenPacket("udp", *port)

	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		_, _, err := conn.ReadFrom(buffer)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		fmt.Println("Current Time: ", string(buffer))
	}
}
