package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type TrafficCounter struct {
	mu      sync.Mutex
	dstPort sync.Map
	srcPort sync.Map
	dstIP   sync.Map
	srcIP   sync.Map
}

func main() {
	a := app.New()
	w := a.NewWindow("Packet Listener")
	w.Resize(fyne.NewSize(700, 700))

	deviceInput := widget.NewEntry()
	deviceInput.SetPlaceHolder("Input devices to listen. Default: all")

	ipInput := widget.NewEntry()
	ipInput.SetPlaceHolder("Input IP-address to collect info about")

	portInput := widget.NewEntry()
	portInput.SetPlaceHolder("Input port to collect info about")

	outputContent := widget.NewMultiLineEntry()
	//outputContent.Disable()
	outputContent.SetMinRowsVisible(20)

	startDefault := widget.Button{
		Text: "Start listener of all traffic",
	}
	startPort := widget.Button{
		Text: "Start listener of traffic by port",
	}
	stop := widget.Button{
		Text: "Stop listener",
	}

	counter := &TrafficCounter{}
	stopper := make(chan struct{}, 1)
	currentRun := ""

	startDefault.OnTapped = func() {
		outputContent.SetText("")
		currentRun = ""
		if len(stopper) > 0 {
			<-stopper
		}
		go startTrafficCapture(counter, outputContent, stopper, deviceInput.Text)
	}

	startPort.OnTapped = func() {
		outputContent.SetText("")
		currentRun = "port"
		if len(stopper) > 0 {
			<-stopper
		}
		go startTrafficCapture(counter, outputContent, stopper, deviceInput.Text)
	}

	stop.OnTapped = func() {
		if len(stopper) == 0 {
			stopper <- struct{}{}
		}
		printTrafficStats(counter, outputContent, currentRun, ipInput.Text, portInput.Text)
		counter = &TrafficCounter{}
	}

	w.SetContent(container.NewVBox(
		deviceInput,
		ipInput,
		portInput,
		&startDefault,
		&startPort,
		&stop,
		outputContent,
	))

	w.ShowAndRun()
}

func startTrafficCapture(counter *TrafficCounter, outputContent *widget.Entry, stopper chan struct{}, deviceName string) {
	if deviceName != "" {
		go captureTraffic(deviceName, counter, outputContent, stopper)
		return
	}

	devices, err := pcap.FindAllDevs()
	if err != nil {
		outputContent.SetText(err.Error())
		return
	}

	for _, device := range devices {
		go captureTraffic(device.Name, counter, outputContent, stopper)
	}
}

func captureTraffic(deviceName string, counter *TrafficCounter, outputContent *widget.Entry, stopper chan struct{}) {
	handle, err := pcap.OpenLive(deviceName, 65536, true, time.Second)
	if err != nil {
		outputContent.SetText(outputContent.Text + fmt.Sprintf("Failed to open device %s: %v\n", deviceName, err))
		return
	}
	outputContent.SetText(outputContent.Text + fmt.Sprintf("Opened device %s\n", deviceName))
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

catcher:
	for {
		select {
		case packet := <-packetSource.Packets():
			incrementCounters(packet, counter, outputContent)
		case <-stopper:
			stopper <- struct{}{}
			break catcher
		}
	}
}

func incrementCounters(packet gopacket.Packet, counter *TrafficCounter, outputContent *widget.Entry) {
	// Determine packet direction based on the source and destination addresses
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		ipLayer = packet.Layer(layers.LayerTypeIPv6)
	}

	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		if ip == nil {
			ip6, _ := ipLayer.(*layers.IPv6)
			if ip6 != nil {
				printPortInfo(packet, "IPv6", len(packet.Data()), counter, outputContent)
			}
		} else {
			printPortInfo(packet, "IPv4", len(packet.Data()), counter, outputContent)
		}
	}
}

func printPortInfo(packet gopacket.Packet, version string, dataSize int, counter *TrafficCounter, outputContent *widget.Entry) {
	// Extract transport layer (TCP or UDP) from packet
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	udpLayer := packet.Layer(layers.LayerTypeUDP)

	if packet.NetworkLayer() == nil {
		return
	}
	srcIP := packet.NetworkLayer().NetworkFlow().Src().String()
	dstIP := packet.NetworkLayer().NetworkFlow().Dst().String()

	counter.mu.Lock()
	val, _ := counter.srcIP.LoadOrStore(srcIP, 0)
	counter.srcIP.Store(srcIP, val.(int)+dataSize)

	val, _ = counter.dstIP.LoadOrStore(dstIP, 0)
	counter.dstIP.Store(dstIP, val.(int)+dataSize)
	counter.mu.Unlock()

	outInfo := fmt.Sprintf("%s --> %s\n", srcIP, dstIP)
	outInfo += fmt.Sprintf("IP Version: %s\n", version)
	outInfo += fmt.Sprintf("Transferred Data Size: %d bytes\n", dataSize)

	if tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)

		keySrc := srcIP + "::" + tcp.SrcPort.String()
		keyDst := dstIP + "::" + tcp.DstPort.String()

		counter.mu.Lock()
		val, _ := counter.srcPort.LoadOrStore(keySrc, 0)
		counter.srcPort.Store(keySrc, val.(int)+dataSize)

		val, _ = counter.dstPort.LoadOrStore(keyDst, 0)
		counter.dstPort.Store(keyDst, val.(int)+dataSize)
		counter.mu.Unlock()

		outInfo += fmt.Sprintf("Source Port (TCP): %d\n", tcp.SrcPort)
		outInfo += fmt.Sprintf("Destination Port (TCP): %d\n", tcp.DstPort)
	} else if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)

		keySrc := srcIP + "::" + udp.SrcPort.String()
		keyDst := dstIP + "::" + udp.DstPort.String()

		counter.mu.Lock()
		val, _ := counter.srcPort.LoadOrStore(keySrc, 0)
		counter.srcPort.Store(keySrc, val.(int)+dataSize)

		val, _ = counter.dstPort.LoadOrStore(keyDst, 0)
		counter.dstPort.Store(keyDst, val.(int)+dataSize)
		counter.mu.Unlock()

		outInfo += fmt.Sprintf("Source Port (UDP): %d\n", udp.SrcPort)
		outInfo += fmt.Sprintf("Destination Port (UDP): %d\n", udp.DstPort)
	}
	outputContent.SetText(outputContent.Text + outInfo + "\n")
}

func printTrafficStats(counter *TrafficCounter, outputContent *widget.Entry, typeOut, ipLtn, portLtn string) {
	outputContent.SetText(outputContent.Text + fmt.Sprintln("Traffic Statistics"))
	outputContent.SetText(outputContent.Text + fmt.Sprintln("------------------"))

	if typeOut == "" {
		val, _ := counter.dstIP.LoadOrStore(ipLtn, 0)
		outputContent.SetText(outputContent.Text + fmt.Sprintf("Incoming: %d bytes\n", val))
		val, _ = counter.srcIP.LoadOrStore(ipLtn, 0)
		outputContent.SetText(outputContent.Text + fmt.Sprintf("Outgoing: %d bytes\n", val))
	} else {
		val, _ := counter.dstPort.LoadOrStore(ipLtn+"::"+portLtn, 0)
		outputContent.SetText(outputContent.Text + fmt.Sprintf("Incoming: %d bytes\n", val))
		val, _ = counter.srcPort.LoadOrStore(ipLtn+"::"+portLtn, 0)
		outputContent.SetText(outputContent.Text + fmt.Sprintf("Outgoing: %d bytes\n", val))
	}
}
