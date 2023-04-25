package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	BroadcastInterval = 5 * time.Second
	MessageBufferSize = 1024
	BroadcastMessage  = "AppBroadcast"
	TerminateMessage  = "AppTerminate"
)

type AppInstance struct {
	IPAddr  net.IP
	Port    int
	Running bool
	lastAdd time.Time
}

var (
	instances = sync.Map{}
)

func main() {
	// get local IP address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var localAddr net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			localAddr = ipnet.IP
			break
		}
	}
	if localAddr == nil {
		fmt.Println("Failed to get local IP address.")
		os.Exit(1)
	}

	multicastAddr, err := net.ResolveUDPAddr("udp", "224.0.0.1:8081")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// get local port number
	sender, err := net.DialUDP("udp", nil, multicastAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer sender.Close()

	listener, err := net.ListenMulticastUDP("udp", nil, multicastAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer listener.Close()

	// start broadcast loop
	go func() {
		for {
			sendBroadcast(sender)
			time.Sleep(BroadcastInterval)
		}
	}()

	// wait for interrupt (disabling application)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sendTerminate(sender)
		os.Exit(1)
	}()

	// remove idle applications
	go func() {
		for {
			instances.Range(func(key, value interface{}) bool {
				instance := value.(AppInstance)
				if time.Now().Sub(instance.lastAdd) > BroadcastInterval*2 {
					instance.Running = false
					instances.Store(key, instance)
				}
				return true
			})
			time.Sleep(BroadcastInterval)
		}
	}()

	// start receive loop
	buf := make([]byte, MessageBufferSize)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err)
			continue
		}
		message := string(buf[:n])
		if message == BroadcastMessage {
			addInstance(remoteAddr.IP, remoteAddr.Port)
		} else if message == TerminateMessage {
			removeInstance(remoteAddr.IP, remoteAddr.Port)
		}
	}
}

func sendBroadcast(sender *net.UDPConn) {
	_, err := sender.Write([]byte(BroadcastMessage))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func sendTerminate(sender *net.UDPConn) {
	_, err := sender.Write([]byte(TerminateMessage))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func addInstance(ip net.IP, port int) {
	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	instance := AppInstance{}
	instance.IPAddr = ip
	instance.Port = port
	instance.Running = true
	instance.lastAdd = time.Now()
	instances.Store(addr, instance)
	printInstances()
}

func removeInstance(ip net.IP, port int) {
	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	instanceRaw, _ := instances.Load(addr)
	instance := instanceRaw.(AppInstance)
	instance.Running = false
	instances.Store(addr, instance)
	printInstances()
}

func printInstances() {
	fmt.Println("Running instances:")
	instances.Range(func(key, value interface{}) bool {
		instance := value.(AppInstance)
		if instance.Running {
			fmt.Printf("- %s\n", key)
		}
		return true
	})
	fmt.Println()
}
