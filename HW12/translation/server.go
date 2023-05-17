package main

import (
	"fmt"
	"io"
	"net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Rule struct {
	SourcePort      string
	DestinationIP   string
	DestinationPort string
}

type TransRule struct {
	Text      string
	DeleteBtn *widget.Button
}

type myListener struct {
	listener net.Listener
	deleted  chan interface{}
}

func deleteRow(options *[]*TransRule, index int, table *fyne.Container) func() {
	return func() {
		close(mapTextLn[(*options)[index].Text].deleted)
		mapTextLn[(*options)[index].Text].listener.Close()
		*options = append((*options)[:index], (*options)[index+1:]...)
		RefreshTable(options, table)
	}
}

func RefreshTable(options *[]*TransRule, table *fyne.Container) {
	// Clear previous rows
	table.Objects = []fyne.CanvasObject{}

	// Add header row
	headerRow := container.New(layout.NewGridLayout(2))
	headerRow.Add(widget.NewLabel("Option"))
	headerRow.Add(widget.NewLabel("")) // Empty cell for spacing
	table.Add(headerRow)

	// Add data rows
	for i, option := range *options {
		row := container.New(layout.NewGridLayout(2))
		row.Add(widget.NewLabel(option.Text))

		deleteBtn := widget.NewButton("Delete", deleteRow(options, i, table))
		option.DeleteBtn = deleteBtn
		row.Add(deleteBtn)

		table.Add(row)
	}

	// Refresh the view
	table.Refresh()
}

var mapTextLn = make(map[string]myListener)

func main() {
	a := app.New()

	dstIPEntry := widget.NewEntry()
	dstPortEntry := widget.NewEntry()
	srcPortEntry := widget.NewEntry()
	statusLabel := widget.NewLabel("")
	table := container.New(layout.NewVBoxLayout())
	options := make([]*TransRule, 0, 1e5)

	sendButton := widget.NewButton("Start", func() {
		dstIP := dstIPEntry.Text
		dstPort := dstPortEntry.Text
		srcPort := srcPortEntry.Text

		rule := Rule{
			SourcePort:      srcPort,
			DestinationIP:   dstIP,
			DestinationPort: dstPort,
		}

		go startListener(rule, statusLabel, table, &options)
	})

	dstIPEntry.SetPlaceHolder("IP Address of destination")
	dstPortEntry.SetPlaceHolder("Port of destination")
	srcPortEntry.SetPlaceHolder("Localhost port")

	content := container.NewVBox(
		container.NewVBox(
			srcPortEntry,
			dstIPEntry,
			dstPortEntry,
		),
		sendButton,
		statusLabel,
		table,
	)

	window := a.NewWindow("Translator")
	window.Resize(fyne.NewSize(500, 500))
	window.SetContent(content)
	window.ShowAndRun()
}

func startListener(rule Rule, statusLabel *widget.Label, table *fyne.Container, options *[]*TransRule) {
	// Listen on source port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", rule.SourcePort))
	if err != nil {
		statusLabel.SetText(fmt.Sprintf("Error starting listener: %v\n", err))
		return
	}
	smartListener := myListener{
		listener: listener,
		deleted:  make(chan interface{}, 1),
	}
	defer smartListener.listener.Close()

	fmt.Printf("Listening on port %s...\n", rule.SourcePort)

	statusLabel.SetText("Connected successfully")

	text := fmt.Sprintf("127.0.0.1\t::\t%s\tTRANSLATING\t%s\t::\t%s\n", rule.SourcePort, rule.DestinationIP, rule.DestinationPort)

	*options = append(*options, &TransRule{
		Text:      text,
		DeleteBtn: nil,
	})

	mapTextLn[text] = smartListener

	RefreshTable(options, table)

	// Handle incoming connections
endless:
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-smartListener.deleted:
				statusLabel.SetText(fmt.Sprintf("Connection closed succsessfuly\n"))
				break endless
			default:
				statusLabel.SetText(fmt.Sprintf("Error accepting connection: %v\n", err))
				continue
			}
		}

		fmt.Printf("Received connection from %s\n", conn.RemoteAddr())

		// Apply translation rule to determine destination address
		destAddr := fmt.Sprintf("%s:%s", rule.DestinationIP, rule.DestinationPort)

		// Open new connection to destination address
		destConn, err := net.Dial("tcp", destAddr)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Error connecting to destination: %v\n", err))
			continue
		}

		fmt.Printf("Forwarding data to %s\n", destAddr)

		// Start forwarding data between connections
		go forwardData(conn, destConn)
		go forwardData(destConn, conn)
	}
}

func forwardData(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()

	_, err := io.Copy(src, dest)
	if err != nil {
		fmt.Println("Error forwarding data:", err)
	}
}
