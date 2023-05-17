package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
	"unsafe"

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

	ipEntry := widget.NewEntry()
	portEntry := widget.NewEntry()

	statusLabel := widget.NewLabel(fmt.Sprintf("Server is offline"))

	var conn *net.UDPConn

	createButton := widget.NewButton("Create server", func() {
		ip := ipEntry.Text
		port := portEntry.Text

		var err error

		addr, err := net.ResolveUDPAddr("udp", ip+":"+port)
		if err != nil {
			fmt.Println(err)
			return
		}

		conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Failed to listen: %v", err))
			return
		}
		defer conn.Close()

		statusLabel.SetText("Server is online")

		updated := false

		for {
			var received, total, sizeSum int
			var rtt time.Duration

			for {
				ans := Packet{}
				conn.SetReadDeadline(time.Now().Add(time.Second))
				err = gob.NewDecoder(conn).Decode(&ans)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						currText := statusLabel.Text
						if !updated {
							fmt.Println("All packets received, exiting the loop...")
							statusLabel.SetText(fmt.Sprintf("End of transmission.\n"+
								"%s\n"+
								"Lost %d packets", currText, total-received))
						}
						updated = true
						break
					}
					statusLabel.SetText(fmt.Sprintf("Failed to read data: %v", err))
					continue
				}
				received++
				total = ans.NumPackets
				rtt += time.Now().Sub(ans.Timestamp)
				sizeSum += int(unsafe.Sizeof(ans))
				statusLabel.SetText(fmt.Sprintf("Received %d packet(s) out of %d\n"+
					"Mean transmission speed is %0.5f gbps", received, ans.NumPackets, float64(1000*sizeSum)/float64(8*rtt.Microseconds())))
				updated = false
			}
		}
	})

	ipEntry.SetPlaceHolder("IP Address")
	portEntry.SetPlaceHolder("Port")

	content := container.NewVBox(
		container.NewVBox(
			ipEntry,
			portEntry,
		),
		createButton,
		statusLabel,
	)

	window := a.NewWindow("UDP Server")
	window.Resize(fyne.NewSize(500, 500))
	window.SetContent(content)
	window.ShowAndRun()
}
