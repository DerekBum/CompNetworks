package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

var port = flag.String("port", ":8081", "Port of localhost server")

func main() {
	flag.Parse()

	conn, err := net.Dial("tcp6", *port)
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	message, _ := reader.ReadString('\n')

	fmt.Printf("Sending message: %s\n", message)
	conn.Write([]byte(message))
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Println("Error reading:", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Received message: %s\n", string(buf))
}
