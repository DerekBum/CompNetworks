package main

import (
	"fmt"
	"net"
)

func main() {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	// Iterate over each interface
	for _, iface := range ifaces {
		// Skip interfaces that are not up or not loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Get addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			panic(err)
		}

		// Iterate over each address
		for _, addr := range addrs {
			// Check if it's an IP address
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}

			// Print IP address and netmask
			fmt.Printf("IP address: %v\n", ipnet.IP)
			fmt.Printf("Netmask: %v\n", ipnet.Mask)
		}
	}
}
