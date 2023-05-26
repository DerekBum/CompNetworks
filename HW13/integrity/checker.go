package main

import (
	"fmt"
	"hash/crc32"
)

const (
	PacketSize = 5
)

// GenerateChecksum calculates the CRC checksum for the given data.
func generateChecksum(data []byte) uint32 {
	crc32q := crc32.NewIEEE()
	crc32q.Write(data)
	return crc32q.Sum32()
}

// EncodePacket encodes the packet by appending the checksum to the data.
func EncodePacket(data []byte) []byte {
	checksum := generateChecksum(data)
	encodedPacket := append(data, byte(checksum), byte(checksum>>8), byte(checksum>>16), byte(checksum>>24))
	return encodedPacket
}

// HasError checks if the packet has an error by verifying the checksum.
func HasError(packet []byte) bool {
	data := packet[:PacketSize]
	checksum := uint32(packet[PacketSize]) | uint32(packet[PacketSize+1])<<8 | uint32(packet[PacketSize+2])<<16 | uint32(packet[PacketSize+3])<<24
	return generateChecksum(data) != checksum
}

func main() {
	packetData := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // Example packet data
	encodedPacket := EncodePacket(packetData)

	// Manipulate data to simulate data corruption
	encodedPacket[2] = 0xFF

	// Verify checksum
	err := HasError(encodedPacket)
	if !err {
		panic("Expected: invalid. Got: valid")
	} else {
		fmt.Println("Yup, checksum is invalid.")
	}
}
