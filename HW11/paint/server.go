package main

import (
	"bufio"
	"flag"
	"net"
)

var port = flag.String("port", ":8081", "Port of localhost server")
var connections []net.Conn

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		line, _ := reader.ReadString('\n')
		for _, currConn := range connections {
			if currConn == conn {
				continue
			}
			currConn.Write([]byte(line))
		}
	}
}

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", *port)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		connections = append(connections, conn)
		go handleConnection(conn)
	}
}
