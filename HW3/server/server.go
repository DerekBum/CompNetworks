package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

var concurrencyLevel = flag.Int("conlvl", 1, "Concurrency level")

var clientCounter int32 = 0

func main() {
	flag.Parse()

	guard := make(chan struct{}, *concurrencyLevel)

	fmt.Println("Launching server...")

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		clientCounter++
		fmt.Printf("Client №%d connected!\nHis address: %v\n", clientCounter, conn.RemoteAddr())
		guard <- struct{}{}
		fmt.Printf("Handling connection of client №%d\n", clientCounter)

		go func(conn net.Conn, clientID *int32, guard chan struct{}) {
			if err := handleConnection(conn); err != nil && err != io.EOF {
				fmt.Printf("Got error: %v\n", err)
			}
			fmt.Printf("Client №%d disconnected!\n", *clientID)
			atomic.AddInt32(clientID, -1)
			<-guard
		}(conn, &clientCounter, guard)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return err
	}
	data, err := os.ReadFile("./" + req.URL.String())
	var resp http.Response
	if err == nil {
		resp = http.Response{Status: "200 OK", Body: io.NopCloser(bytes.NewReader(data))}
	} else {
		resp = http.Response{Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("File not found"))}
	}
	err = resp.Write(conn)
	return err
}
