package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

var port = flag.String("port", ":8081", "Port of the localhost server. Example: \":8081\"")
var fileName = flag.String("fn", "exampleClient.txt", "Name of the file")
var tOut = flag.Int("time", 2, "Timeout for ACK response (in seconds)")

var packetSize = 128

func main() {
	var receiveFromServer bool
	flag.BoolVar(&receiveFromServer, "recv", false, "Get file from server")

	flag.Parse()

	timeout := time.Duration(*tOut) * time.Second

	if receiveFromServer {
		receive(timeout)
		return
	}

	// Open file for reading
	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Determine file name and size
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file information:", err)
		return
	}
	fileSize := fileInfo.Size()
	fileHeader := "0" + fileInfo.Name() + ":" + strconv.FormatInt(fileSize, 10)

	// Create UDP connection to server
	addr, err := net.ResolveUDPAddr("udp", *port)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	sumFileHeader := getSum([]byte(fileHeader))
	sumByte := make([]byte, 2)
	binary.BigEndian.PutUint16(sumByte, sumFileHeader)

	// Send file header to server
	if _, err := conn.Write(append(sumByte, []byte(fileHeader)...)); err != nil {
		fmt.Println("Error sending file header:", err)
		return
	}

	// Send file data in packets
	packet := make([]byte, packetSize)
	var seqNum byte
	var numSent, ackLost int
	for {
		// Read packet data from file
		n, err := file.Read(packet[1:])
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error reading from file:", err)
			return
		}

		// Send packet
		packet[0] = seqNum
		attempt := 1
	SendPacket:
		for {
			fmt.Printf("Sending packet to server: %d\n", numSent)
			sumFileHeader := getSum(packet[:n+1])
			sumByte := make([]byte, 2)
			binary.BigEndian.PutUint16(sumByte, sumFileHeader)

			if _, err := conn.Write(append(sumByte, packet[:n+1]...)); err != nil {
				fmt.Printf("Error sending packet (seq=%d, attempt=%d): %v\n", seqNum, attempt, err)
				time.Sleep(timeout)
			} else {
				// Wait for ACK
				for {
					conn.SetReadDeadline(time.Now().Add(timeout))
					ack := make([]byte, 1)
					_, err = conn.Read(ack)
					if err != nil {
						fmt.Printf("Error receiving ACK (seq=%d): %v\n", seqNum, err)
						continue SendPacket
					}
					if ack[0] != seqNum {
						fmt.Printf("Error: incorrect ACK received (seq=%d, expected=%d)\n", ack[0], seqNum)
						continue SendPacket
					}

					// Simulate packet loss
					if rand.Float32() < 0.3 {
						ackLost++
						fmt.Println("Lost ACK packet from server: ", numSent)
						continue
					}

					fmt.Println("Received ACK packet from server: ", numSent)

					break
				}
				numSent++
				seqNum ^= 1
				break
			}
			attempt++
		}
	}

	fmt.Printf("Sent %d packets, %d ACKs lost\n", numSent, ackLost)
}

func receive(timeout time.Duration) {
	// Open file for writing
	file, err := os.Create(*fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Determine file name and size
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file information:", err)
		return
	}
	fileHeader := "1" + fileInfo.Name()

	// Create UDP connection to server
	addr, err := net.ResolveUDPAddr("udp", *port)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	sumFileHeader := getSum([]byte(fileHeader))
	sumByte := make([]byte, 2)
	binary.BigEndian.PutUint16(sumByte, sumFileHeader)

	// Send receive command to server
	if _, err := conn.Write(append(sumByte, []byte(fileHeader)...)); err != nil {
		fmt.Println("Error sending receive command:", err)
		return
	}

	// Receive file data in packets
	var seqNum byte
	var numReceived, numLost int
	var prevSeq byte
	prevSeq ^= 1
	for {
		packet := make([]byte, packetSize+2)
		// Receive packet
		n, _, err := conn.ReadFromUDP(packet)
		if err != nil {
			fmt.Println("Error receiving packet:", err)
			return
		}

		gotSum := binary.BigEndian.Uint16(packet[0:2])
		if !validate(packet[2:], gotSum) {
			fmt.Println("Incorrect packet control sum")
			return
		}

		// Simulate packet loss
		if rand.Float32() < 0.3 {
			numLost++
			fmt.Println("Lost packet from server to client: ", numReceived)
			continue
		}

		fmt.Println("Received packet from server to client: ", numReceived)

		// Check sequence number
		seqNum = packet[2]

		// Write packet data to file
		if seqNum != prevSeq {
			if _, err := file.Write(packet[3:n]); err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}
		}

		// Send ACK
		if _, err := conn.Write([]byte{seqNum}); err != nil {
			fmt.Println("Error sending ACK:", err)
			return
		}
		if seqNum != prevSeq {
			numReceived++
			prevSeq = seqNum
		}
	}
}

func getSum(input []byte) uint16 {
	var res uint16
	if len(input)%2 != 0 {
		input = append([]byte{0}, input...)
	}
	for i := 0; i < len(input); i += 2 {
		res += binary.BigEndian.Uint16(input[i : i+2])
	}
	return ^uint16(0) - res
}

func validate(input []byte, sum uint16) bool {
	correct := getSum(input)
	return correct == sum
}
