package main

import (
	"encoding/gob"
	"fmt"
	"io"
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

	var ln net.Listener

	createButton := widget.NewButton("Create server", func() {
		ip := ipEntry.Text
		port := portEntry.Text

		var err error

		ln, err = net.Listen("tcp", ip+":"+port)
		if err != nil {
			statusLabel.SetText("Failed to listen")
		}

		statusLabel.SetText("Server is online")

		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					break
				}

				go func(conn net.Conn) {
					var received, total, sizeSum int
					var rtt time.Duration
					defer conn.Close()

					for {
						ans := Packet{}
						err := gob.NewDecoder(conn).Decode(&ans)
						if err == io.EOF {
							fmt.Println("All packets received, exiting the loop...")
							currText := statusLabel.Text
							statusLabel.SetText(fmt.Sprintf("End of transmission.\n"+
								"%s\n"+
								"Lost %d packets", currText, total-received))
							break
						}
						if err != nil {
							statusLabel.SetText(fmt.Sprintf("Failed to read data: %v", err))
							continue
						}
						received++
						total = ans.NumPackets
						rtt += time.Now().Sub(ans.Timestamp)
						sizeSum += int(unsafe.Sizeof(ans))
						statusLabel.SetText(fmt.Sprintf("Received %d packet(s) out of %d\n"+
							"Mean transmission speed is %0.5f gbps", received, ans.NumPackets, float64(1000*sizeSum)/float64(8*rtt.Microseconds())))
					}
				}(conn)
			}
		}()
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

	window := a.NewWindow("TCP Server")
	window.Resize(fyne.NewSize(500, 500))
	window.SetContent(content)
	window.ShowAndRun()
}
