package main

import (
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func checkMask(mask string) bool {
	dec := strings.Split(mask, ".")
	if len(dec) != 4 {
		return false
	}
	for _, el := range dec {
		num, err := strconv.Atoi(el)
		if err != nil {
			return false
		}
		if num < 0 || num > 255 {
			return false
		}
	}
	return true
}

func initiateScan(progress *widget.ProgressBar, output *widget.Label, inputMask string, logOut *widget.Label) {
	if !checkMask(inputMask) {
		output.SetText(fmt.Sprintf("Incorrect mask"))
		return
	}

	// Get the local machine's IP address
	host, err := os.Hostname()
	if err != nil {
		output.SetText(fmt.Sprintln("Failed to get hostname:", err))
		return
	}
	ip := ""

	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	// Get the network mask
	mask := net.IPMask(net.ParseIP(inputMask).To4())

	// Get the network address
	ipAddr := net.ParseIP(ip).To4()
	network := net.IPNet{
		IP:   ipAddr.Mask(mask),
		Mask: mask,
	}

	selfMACAddr := getSelfMACAddress()

	// Display information about the local machine
	output.SetText(output.Text + fmt.Sprintf("Local Machine:\n"))
	output.SetText(output.Text + fmt.Sprintf("IP Address: %s\n", ip))
	output.SetText(output.Text + fmt.Sprintf("MAC Address: %s\n", selfMACAddr))
	output.SetText(output.Text + fmt.Sprintf("Name: %s\n\n", host))

	// Scan the network and display information about other computers
	output.SetText(output.Text + fmt.Sprintf("Network Computers:\n"))

	iface, err := pcap.FindAllDevs()
	if err != nil {
		output.SetText(fmt.Sprintln("Failed to find network interface:", err))
		return
	}

	// Start packet capture
	handle, err := pcap.OpenLive(iface[0].Name, 65536, true, pcap.BlockForever)
	if err != nil {
		output.SetText(fmt.Sprintln("Failed to open network interface:", err))
		return
	}
	defer handle.Close()

	filter := "icmp and icmp[0]=0"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		output.SetText(fmt.Sprintln("Failed to set BPF filter:", err))
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	count := 0
	maskSize, _ := mask.Size()
	netMaskSize := math.Pow(2, float64(32-maskSize))

	for ip := network.IP.Mask(network.Mask); network.Contains(ip); inc(ip) {
		count++
		progress.SetValue(float64(count) / netMaskSize)
		if ip.Equal(ipAddr) {
			continue // Skip the local machine
		}

		mac, err := getMACAddress(packetSource, ip.String())
		if err != nil {
			continue // Skip if MAC address retrieval fails
		}

		// Get the hostname
		names, _ := net.LookupAddr(ip.String())
		name := "--"
		if len(names) > 0 {
			name = names[0]
		}

		// Display information about the computer on the network
		output.SetText(output.Text + fmt.Sprintf("IP Address: %s\n", ip))
		output.SetText(output.Text + fmt.Sprintf("MAC Address: %s\n", mac))
		output.SetText(output.Text + fmt.Sprintf("Name: %s\n\n", name))
	}
	logOut.SetText("Scan is complete")
}

func main() {
	a := app.New()
	w := a.NewWindow("Network Scanner")
	w.Resize(fyne.NewSize(500, 500))

	maskInput := widget.NewEntry()
	maskInput.SetPlaceHolder("Input network mask here")

	outputLabel := widget.NewLabel("")

	startButton := widget.Button{
		Text: "Start scan",
	}

	progress := widget.NewProgressBar()

	logOut := widget.NewLabel("Ready to scan")

	startButton.OnTapped = func() {
		outputLabel.Text = ""
		logOut.SetText("Scanning Network...")
		go func() { initiateScan(progress, outputLabel, maskInput.Text, logOut) }()
	}

	w.SetContent(container.NewVBox(
		logOut,
		progress,
		maskInput,
		&startButton,
		outputLabel,
	))

	w.ShowAndRun()
}

func getMACAddress(packetSource *gopacket.PacketSource, destIP string) (net.HardwareAddr, error) {
	err := sendEchoRequest(destIP)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(100 * time.Millisecond)

	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("no response")
		case packet := <-packetSource.Packets():
			icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
			if icmpLayer != nil {
				icmpL, _ := icmpLayer.(*layers.ICMPv4)
				if icmpL.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
					// Extract the MAC address from the Ethernet layer
					ethLayer := packet.Layer(layers.LayerTypeEthernet)
					if ethLayer != nil {
						ethernet, _ := ethLayer.(*layers.Ethernet)
						return ethernet.SrcMAC, err
					}
				}
			}
		}
	}
}

func sendEchoRequest(destIP string) error {
	// Resolve the IP address
	ipAddr, err := net.ResolveIPAddr("ip4", destIP)
	if err != nil {
		return err
	}

	// Create a new ICMP connection
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create the ICMP message
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("HELLO"),
		},
	}

	// Serialize the ICMP message
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	// Send the ICMP message to the target IP
	_, err = conn.WriteTo(msgBytes, ipAddr)
	return err
}

// Helper function to increment an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Helper function to get the MAC address of the local machine
func getSelfMACAddress() string {
	ifas, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifa := range ifas {
		if ifa.Flags&net.FlagLoopback != 0 {
			continue
		}
		if ifa.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifa.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ifa.HardwareAddr.String()
			}
		}
	}
	return ""
}
