package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"
)

type Packet struct {
	Index int
	Data  []byte
}

const PacketSize = 64
const WindowSize = 4

var port = flag.String("port", ":8081", "Port for the server to work")
var fileName = flag.String("fn", "example.txt", "Name of file to send")

func main() {
	// Connect to server
	conn, err := net.Dial("tcp", *port)
	if err != nil {
		log.Fatal("Error connecting to server:", err.Error())
	}
	defer conn.Close()

	// Get file name from command line arguments
	fmt.Println("File name:", *fileName)

	// Send file name to server
	err = gob.NewEncoder(conn).Encode(*fileName)
	if err != nil {
		log.Println("Error sending file name:", err.Error())
		return
	}

	// Read file data and split into packets
	data, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Fatal("Error reading file:", err.Error())
	}
	packets := splitIntoPackets(data)

	leftBound := 0
	rightBound := leftBound + WindowSize
	if rightBound > len(packets) {
		rightBound = len(packets)
	}

	for leftBound != rightBound {
		for i := leftBound; i < rightBound; i++ {
			var sendToServer bytes.Buffer
			err = gob.NewEncoder(conn).Encode(packets[i])
			if err != nil {
				fmt.Println("Error gob:", err.Error())
				continue
			}
			_, err = conn.Write(sendToServer.Bytes())
			if err != nil {
				fmt.Println("Error sending packet:", err.Error())
				continue
			}
			fmt.Printf("Packet %v sent to server\n", i)
		}
		logSender(leftBound, rightBound, len(packets))

	loop:
		for timeout := time.After(2 * time.Second); ; {
			select {
			case <-timeout:
				break loop
			default:
			}
			ackBuffer := make([]byte, PacketSize)
			conn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := conn.Read(ackBuffer)
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			if err != nil && err != io.EOF {
				fmt.Println("Error receiving ACK:", err.Error())
				time.Sleep(500 * time.Millisecond)
				continue
			}
			ack, err := strconv.Atoi(string(ackBuffer[:n]))
			if err != nil && err != io.EOF {
				fmt.Println("Error receiving ACK:", err.Error())
				time.Sleep(500 * time.Millisecond)
				continue
			}

			fmt.Printf("Received ACK: %d\n", ack)

			if ack == leftBound {
				leftBound++
				rightBound++

				if rightBound > len(packets) {
					rightBound--
					continue
				}

				logSender(leftBound, rightBound, len(packets))
			}
		}
	}

	fmt.Println("File transfer complete")
}

func splitIntoPackets(data []byte) []Packet {
	var packets []Packet
	dataSize := len(data)
	numPackets := (dataSize + PacketSize - 1) / PacketSize

	for i := 0; i < numPackets; i++ {
		start := i * PacketSize
		end := (i + 1) * PacketSize
		if end > dataSize {
			end = dataSize
		}
		packetData := data[start:end]
		packets = append(packets, Packet{Index: i, Data: packetData})
	}

	return packets
}

func logSender(leftBound, rightBound, size int) {
	fmt.Printf("Sender: ")
	for i := 0; i < leftBound; i++ {
		fmt.Printf("%d ", i)
	}
	fmt.Printf("[")
	for i := leftBound; i < rightBound; i++ {
		fmt.Printf("%d", i)
		if i+1 != rightBound {
			fmt.Printf(" ")
		}
	}
	fmt.Printf("]")
	for i := rightBound; i < size; i++ {
		fmt.Printf("%d ", i)
	}
	fmt.Printf("\n\n")
}
