package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

type Packet struct {
	Index int
	Data  []byte
}

const WindowSize = 4
const INF = 1e6

var port = flag.String("port", ":8081", "Port for the server to work")

func main() {
	// Listen for incoming connections
	listener, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatal("Error starting listener:", err.Error())
	}
	defer listener.Close()
	fmt.Printf("Server started, listening on port %s...\n", *port)

	for {
		// Accept incoming connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}

		// Handle connection in a new goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Receive file name from client
	var fileName string
	err := gob.NewDecoder(conn).Decode(&fileName)
	if err != nil && err != io.EOF {
		fmt.Println("Error receiving file name:", err.Error())
		return
	}
	fmt.Println("Received file name:", fileName)

	// Create file for writing
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("Error creating file for writing:", err.Error())
		return
	}
	defer file.Close()

	// Initialize variables for GBN protocol
	packets := make([]Packet, INF)
	leftBound := 0
	rightBound := WindowSize

	// Receive file data from client
	for {
		// Receive packet from client
		var packet Packet
		err = gob.NewDecoder(conn).Decode(&packet)

		// Check if all packets have been received
		if err == io.EOF {
			fmt.Println("All packets received, exiting GBN loop...")
			break
		}

		if err != nil {
			fmt.Println("Error gob:", err.Error())
			continue
		}

		// Create new packet
		packets[packet.Index] = packet

		fmt.Printf("Packet %v received\n", packet.Index)

		if packet.Index == leftBound {
			file.Write(packet.Data)

			// Send ACK for packet
			_, err = conn.Write([]byte(strconv.Itoa(packet.Index)))
			if err != nil {
				log.Println("Error sending ACK for packet:", err.Error())
				return
			}

			fmt.Printf("ACK %v sent for packet %v\n", packet.Index, packet.Index)

			leftBound++
			rightBound++

			logReceiver(leftBound, rightBound)
		}
	}

	fmt.Println("File transfer complete")
}

func logReceiver(leftBound, rightBound int) {
	fmt.Printf("Receiver: ")
	for i := 0; i < leftBound; i++ {
		fmt.Printf("%d ", i)
	}
	fmt.Printf("[")
	for i := leftBound; i < rightBound; i++ {
		fmt.Printf("?")
		if i+1 != rightBound {
			fmt.Printf(" ")
		}
	}
	fmt.Printf("]")
	fmt.Printf("\n\n")
}
