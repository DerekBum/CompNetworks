package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jlaffaye/ftp"
)

var c *ftp.ServerConn

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("FTP Client")
	myWindow.Resize(fyne.NewSize(1000, 900))

	serverEntry := widget.NewEntry()
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	statusLabel := widget.NewLabel("Not connected")
	outputLabel := widget.NewLabel("")
	fileContent := widget.NewMultiLineEntry()
	fileContent.Disable()

	statusChan := make(chan string)

	// Create a label to display status updates
	serverStatusLabel := widget.NewLabel("")

	// Create a goroutine to update the status label with messages from the statusChan
	go func() {
		for message := range statusChan {
			serverStatusLabel.SetText(message)
		}
	}()

	getFilesButton := widget.NewButton("Get Files", func() {
		resp := getAllFiles(c)
		outputLabel.SetText(resp)
	})
	getFilesButton.Disable()

	createDirButton := widget.NewButton("Create Directory", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path to new directory", func(filePath string) {
			err := c.MakeDir(filePath)
			if err != nil {
				// Handle error
				statusChan <- fmt.Sprintf("Upload failed: %s", err)
				return
			}
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()

		statusChan <- "Creation of folder successful"
	})
	createDirButton.Disable()

	uploadButton := widget.NewButton("Upload", func() {
		fileSelect := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				statusChan <- fmt.Sprintln("File dialog cancelled or error:", err)
				return
			}

			// Get the chosen file
			fileBytes, err := ioutil.ReadAll(file)
			if err != nil {
				statusChan <- fmt.Sprintln("Error reading file:", err)
				return
			}

			// Close the reader
			file.Close()

			locationSelect := dialog.NewEntryDialog("Location", "Input path for uploading file to store", func(filePath string) {
				err = c.Stor(filePath+"/"+file.URI().Name(), bytes.NewReader(fileBytes))
				if err != nil {
					// Handle error
					statusChan <- fmt.Sprintf("Upload failed: %s", err)
					return
				}
			}, myWindow)
			locationSelect.Resize(fyne.NewSize(500, 100))
			locationSelect.Show()

			// Display success message
			statusChan <- "Upload successful"
		}, myWindow)
		fileSelect.Resize(fyne.NewSize(700, 700))
		fileSelect.Show()
	})
	uploadButton.Disable()

	downloadButton := widget.NewButton("Download", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path for downloading file to", func(filePath string) {
			splitted := strings.Split(filePath, "/")
			fileName := splitted[len(splitted)-1]

			reader, err := c.Retr(filePath)
			if err != nil {
				statusChan <- fmt.Sprintf("Download failed: %s", err)
				return
			}
			body, _ := ioutil.ReadAll(reader)

			reader.Close()

			fileContent.SetText(string(body))

			dirSelect := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {

				file, err := os.Create(dir.Path() + "/" + fileName)
				if err != nil {
					statusChan <- fmt.Sprintf("Download failed: %s", err)
					return
				}
				defer file.Close()
				_, err = file.Write(body)
				if err != nil {
					// Handle error
					statusChan <- fmt.Sprintf("Download failed: %s", err)
					return
				}
			}, myWindow)
			dirSelect.Resize(fyne.NewSize(700, 700))
			dirSelect.Show()
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()

		statusChan <- "Download successful"
	})
	downloadButton.Disable()

	createFileButton := widget.NewButton("Create File", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path for creating file to", func(filePath string) {
			err := c.Stor(filePath, strings.NewReader(fileContent.Text))
			if err != nil {
				statusChan <- fmt.Sprintf("Creation failed: %s", err)
				return
			}
			statusChan <- "Creation of file successful"
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()
	})
	createFileButton.Disable()

	readFileButton := widget.NewButton("Read File", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path for reading file", func(filePath string) {
			f, err := c.Retr(filePath)
			if err != nil {
				statusChan <- fmt.Sprintf("Reading failed: %s", err)
				return
			}
			defer f.Close()

			content, err := ioutil.ReadAll(f)
			if err != nil {
				statusChan <- fmt.Sprintf("Reading failed: %s", err)
				return
			}
			fileContent.SetText(string(content))
			statusChan <- "Reading of file successful"
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()
	})
	readFileButton.Disable()

	updateFileButton := widget.NewButton("Update File", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path for updating file", func(filePath string) {
			err := c.Stor(filePath, strings.NewReader(fileContent.Text))
			if err != nil {
				statusChan <- fmt.Sprintf("Updating failed: %s", err)
				return
			}
			statusChan <- "Updating of file successful"
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()
	})
	updateFileButton.Disable()

	deleteFileButton := widget.NewButton("Delete File", func() {
		locationSelect := dialog.NewEntryDialog("Location", "Input path for deleting file", func(filePath string) {
			err := c.Delete(filePath)
			if err != nil {
				statusChan <- fmt.Sprintf("Deleting failed: %s", err)
				return
			}
			statusChan <- "Deleting of file successful"
		}, myWindow)
		locationSelect.Resize(fyne.NewSize(500, 100))
		locationSelect.Show()
	})
	deleteFileButton.Disable()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Server", Widget: serverEntry},
			{Text: "Username", Widget: usernameEntry},
			{Text: "Password", Widget: passwordEntry},
		},
		OnSubmit: func() {
			server := serverEntry.Text + ":21"
			username := usernameEntry.Text
			password := passwordEntry.Text

			// Connect to the server
			var err error
			c, err = ftp.Dial(server, ftp.DialWithTimeout(5*time.Second))
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Connection failed: %s", err))
				return
			}

			err = c.Login(username, password)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Authorisation failed: %s", err))
				return
			}

			// Display success message
			statusLabel.SetText("Connected")

			// Print all server files
			resp := getAllFiles(c)
			outputLabel.SetText(resp)

			// Enable the action buttons
			getFilesButton.Enable()
			createDirButton.Enable()
			uploadButton.Enable()
			downloadButton.Enable()
			createFileButton.Enable()
			readFileButton.Enable()
			updateFileButton.Enable()
			deleteFileButton.Enable()
			fileContent.Enable()
		},
		SubmitText: "Connect",
	}

	container := fyne.NewContainerWithLayout(layout.NewGridLayout(1),
		form,
	)

	// Create a container to hold the action buttons
	buttonContainer := fyne.NewContainerWithLayout(layout.NewHBoxLayout(),
		getFilesButton, createDirButton, uploadButton, downloadButton, createFileButton, readFileButton, updateFileButton, deleteFileButton,
	)

	// Create a container to hold the status label
	statusContainer := fyne.NewContainerWithLayout(layout.NewVBoxLayout(),
		widget.NewLabel("Connection Status:"),
		statusLabel,
	)

	outputContainer := fyne.NewContainerWithLayout(layout.NewGridLayout(2),
		widget.NewLabel("Directory Output:"), widget.NewLabel("Server Messages:"),
		outputLabel, serverStatusLabel,
	)

	fileContainer := fyne.NewContainerWithLayout(layout.NewVBoxLayout(),
		widget.NewLabel("File Content:"),
		fileContent,
	)

	// Create a container to hold all the widgets
	content := fyne.NewContainerWithLayout(layout.NewVBoxLayout(),
		container, buttonContainer, statusContainer, outputContainer, fileContainer,
	)

	// Set the window content
	myWindow.SetContent(content)

	myWindow.ShowAndRun()
}
