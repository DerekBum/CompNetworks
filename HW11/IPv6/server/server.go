package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
)

var port = flag.String("port", ":8081", "Port of localhost server")

func main() {
	flag.Parse()

	l, err := net.Listen("tcp6", *port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer l.Close()
	fmt.Printf("Listening on :%s\n", *port)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			continue
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	message := strings.TrimSpace(string(buf))
	fmt.Printf("Received message: %s\n", message)
	conn.Write([]byte(strings.ToUpper(message)))
	conn.Close()
}
