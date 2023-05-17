package main

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Packet struct {
	Timestamp  time.Time
	NumPackets int
	Data       []byte
}

func main() {
	a := app.New()

	buffer := make([]byte, 1024)

	ipEntry := widget.NewEntry()
	portEntry := widget.NewEntry()
	packets := widget.NewEntry()
	statusLabel := widget.NewLabel("")
	sendButton := widget.NewButton("Send", func() {
		numPacketsString := packets.Text
		ip := ipEntry.Text
		port := portEntry.Text

		numPackets := 1
		if num, err := strconv.Atoi(numPacketsString); err == nil {
			numPackets = num
		}

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
		if err != nil {
			statusLabel.SetText("Failed to connect")
			return
		}

		defer conn.Close()

		for i := 0; i < numPackets; i++ {
			message := Packet{NumPackets: numPackets}

			rand.Read(buffer)
			message.Data = buffer

			message.Timestamp = time.Now()

			err = gob.NewEncoder(conn).Encode(message)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Failed to send data: %v", err))
				return
			}
			time.Sleep(10 * time.Millisecond)
		}

		statusLabel.SetText(fmt.Sprintf("Sent %d packet(s)", numPackets))
	})

	ipEntry.SetPlaceHolder("IP Address")
	portEntry.SetPlaceHolder("Port")
	packets.SetPlaceHolder("Number of packets to send")

	content := container.NewVBox(
		container.NewVBox(
			ipEntry,
			portEntry,
		),
		packets,
		sendButton,
		statusLabel,
	)

	window := a.NewWindow("TCP Client")
	window.Resize(fyne.NewSize(500, 500))
	window.SetContent(content)
	window.ShowAndRun()
}
