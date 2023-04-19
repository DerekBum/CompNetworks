package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var port = flag.String("port", ":8081", "Port of the localhost server. Example: \":8081\"")
var tOut = flag.Int("time", 2, "Timeout for ACK response (in seconds)")

var packetSize = 128

func main() {
	flag.Parse()

	// Open UDP listener
	addr, err := net.ResolveUDPAddr("udp", *port)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}
	defer conn.Close()

	// Receive file name and size
	buffer := make([]byte, 128)
	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error receiving file header:", err)
		return
	}

	gotSum := binary.BigEndian.Uint16(buffer[0:2])
	if !validate(buffer[2:], gotSum) {
		fmt.Println("Incorrect file header control sum")
		return
	}

	// Parse file name and size
	needToSend := false
	if buffer[2] == '1' {
		needToSend = true
	}
	if needToSend {
		sendToClient(conn, addr, string(buffer[3:n]))
		return
	}
	fileHeader := string(buffer[3:n])
	parts := strings.Split(fileHeader, ":")
	if len(parts) != 2 {
		fmt.Println("Invalid file header:", fileHeader)
		return
	}
	fileSize, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		fmt.Println("Error parsing file size:", err)
		return
	}

	// Open file for writing
	file, err := os.Create(parts[0])
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Receive file data in packets
	var seqNum byte
	var numReceived, numLost int
	var prevSeq byte
	prevSeq ^= 1
	for {
		packet := make([]byte, packetSize+2)
		// Receive packet
		n, addr, err := conn.ReadFromUDP(packet)
		if err != nil {
			fmt.Println("Error receiving packet:", err)
			return
		}

		gotSum := binary.BigEndian.Uint16(packet[0:2])
		if !validate(packet[2:], gotSum) {
			fmt.Println(fmt.Sprintf("Incorrect packet control sum. Expected %d, got %d", getSum(packet[2:]), gotSum))
			return
		}

		// Simulate packet loss
		if rand.Float32() < 0.3 {
			numLost++
			fmt.Println("Lost packet from client to server: ", numReceived)
			continue
		}

		fmt.Println("Received packet from client to server: ", numReceived)

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
		if _, err := conn.WriteToUDP([]byte{seqNum}, addr); err != nil {
			fmt.Println("Error sending ACK:", err)
			return
		}
		if seqNum != prevSeq {
			numReceived++
			prevSeq = seqNum
		}

		// Check if all packets have been received
		if numReceived == int(math.Ceil(float64(fileSize)/float64(128-1))) {
			fmt.Printf("Received %d packets, lost %d packets\n", numReceived, numLost)
		}
	}
}

func sendToClient(conn *net.UDPConn, addr *net.UDPAddr, fileName string) {
	timeout := time.Duration(*tOut) * time.Second

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

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
			fmt.Printf("Sending packet to client: %d\n", numSent)
			sumFileHeader := getSum(packet[:n+1])
			sumByte := make([]byte, 2)
			binary.BigEndian.PutUint16(sumByte, sumFileHeader)

			if _, err := conn.WriteToUDP(append(sumByte, packet[:n+1]...), addr); err != nil {
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
