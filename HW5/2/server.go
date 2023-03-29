package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

var port = flag.String("port", ":8081", "Port of localhost server (starts with \":\")")

func main() {
	flag.Parse()

	fmt.Println("Starting server...")

	ln, err := net.Listen("tcp", *port)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error:", err.Error())
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command := scanner.Text()

		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		ans, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(conn, "Error: %s\n", err.Error())
		} else {
			fmt.Fprintf(conn, string(ans))
		}
	}
}
