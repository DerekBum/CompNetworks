package main

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestChecker(t *testing.T) {
	text := "This is a test message. Blah blah blah, can I get an A without an exam?"

	numPackets := len(text) / PacketSize

	packets := make([][]byte, numPackets)
	encodedPackets := make([][]byte, numPackets)

	for i := 0; i < numPackets; i++ {
		startIndex := i * PacketSize
		endIndex := startIndex + PacketSize
		if endIndex > len(text) {
			endIndex = len(text)
		}
		packets[i] = []byte(text[startIndex:endIndex])
		encodedPackets[i] = EncodePacket(packets[i])
	}

	errored := make([]bool, numPackets)

	// Introduce errors in some packets
	for i := 0; i < numPackets; i++ {
		if rand.Intn(4) == 0 { // Introduce errors in 25% of the packets
			errored[i] = true
			bitPosition := rand.Intn(PacketSize * 8)
			packets[i][bitPosition/8] ^= 1 << (bitPosition % 8)
			encodedPackets[i][bitPosition/8] ^= 1 << (bitPosition % 8)
		}
	}

	// Process each packet
	for i, encodedPacket := range encodedPackets {
		hasError := HasError(encodedPacket)

		fmt.Printf("Packet %d:\n", i+1)
		fmt.Printf("Payload: %s\n", packets[i])
		fmt.Printf("Encoded Packet: %v\n", encodedPacket)
		fmt.Printf("Control Code: %v\n", encodedPacket[PacketSize:])

		if hasError {
			fmt.Println("Error detected in the packet.")
		} else {
			fmt.Println("Packet is error-free.")
		}

		if hasError != errored[i] {
			t.Errorf("Got isError: %v, Expected: %v", hasError, !hasError)
		}

		fmt.Println()
	}
}
