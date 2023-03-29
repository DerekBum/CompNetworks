package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

var addrTo = flag.String("addrTo", "", "Email address of receiver")
var addrFrom = flag.String("addrFrom", "", "Email address of sender")
var pwrd = flag.String("pwrd", "", "Email password of sender")
var filePath = flag.String("fp", "hello.txt", "File to send (.txt or .html)")
var smtpHost = flag.String("host", "mail.sibnet.ru", "SMTP host")
var smtpPort = flag.String("port", "25", "SMTP port")
var images = flag.String("imgs", "", "Images for attachments")

func main() {
	flag.Parse()

	// Specify the mail server host and port number
	serverAddr := *smtpHost + ":" + *smtpPort

	// Connect to the mail server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to the mail server:", err)
		return
	}

	// Send the greeting message to the mail server
	err = sendCommand(conn, "HELO "+*smtpHost)
	if err != nil {
		fmt.Println("Error sending HELO command:", err)
		return
	}

	// Send the AUTH LOGIN command to the mail server
	err = sendCommand(conn, "AUTH LOGIN")
	if err != nil {
		fmt.Println("Error sending AUTH LOGIN command:", err)
		return
	}
	sender := *addrFrom
	err = sendCommand(conn, base64.StdEncoding.EncodeToString([]byte(sender)))
	if err != nil {
		fmt.Println("Error sending base64-encoded username:", err)
		return
	}
	err = sendCommand(conn, base64.StdEncoding.EncodeToString([]byte(*pwrd)))
	if err != nil {
		fmt.Println("Error sending base64-encoded password:", err)
		return
	}

	// Send the "MAIL FROM" command
	err = sendCommand(conn, "MAIL FROM:<"+sender+">")
	if err != nil {
		fmt.Println("Error sending MAIL FROM command:", err)
		return
	}

	// Send the "RCPT TO" command
	recipient := *addrTo
	err = sendCommand(conn, "RCPT TO:<"+recipient+">")
	if err != nil {
		fmt.Println("Error sending RCPT TO command:", err)
		return
	}

	// Send the email content
	err = sendCommand(conn, "DATA")
	if err != nil {
		fmt.Println("Error sending DATA command:", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// Get the email header and body from the user
	fmt.Print("Enter the email subject: ")
	subject, _ := reader.ReadString('\n')
	subject = strings.TrimSpace(subject)

	body, err := ioutil.ReadFile(*filePath)
	if err != nil {
		fmt.Println("Error while reading a body file: ", err)
		return
	}

	var message []byte

	// Construct the email message and send it to the mail server
	if *images == "" {
		message = []byte("Subject: " + subject + "\r\n\r\n" + string(body) + "\r\n.")
	} else {
		message = createMessageWithAttachments(sender, recipient, subject, *images, string(body))
	}

	err = sendCommand(conn, string(message))
	if err != nil {
		fmt.Println("Error sending email message:", err)
		return
	}

	// Send the "QUIT" command
	err = sendCommand(conn, "QUIT")
	if err != nil {
		fmt.Println("Error sending QUIT command:", err)
		return
	}

	// Close the connection to the mail server
	err = conn.Close()
	if err != nil {
		fmt.Println("Error closing connection to mail server:", err)
		return
	}
}

// Sends a command to the mail server and returns any error that occurs
func sendCommand(conn net.Conn, command string) error {
	fmt.Println(command)

	_, err := conn.Write([]byte(command + "\r\n"))
	if err != nil {
		return err
	}

	response := make([]byte, 512)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	fmt.Println(string(response))
	return nil
}

func createMessageWithAttachments(sender, recipient, subject, images, body string) []byte {
	// Create the email message with MIME parts
	boundary := "$$$/|\\$$$"
	message := []byte("From: " + sender + "\r\n")
	message = append(message, []byte("To: "+recipient+"\r\n")...)
	message = append(message, []byte("Subject: "+subject+"\r\n")...)
	message = append(message, []byte("MIME-Version: 1.0\r\n")...)
	message = append(message, []byte("Content-Type: multipart/mixed; boundary="+boundary+"\r\n\r\n")...)
	message = append(message, []byte("--"+boundary+"\r\n")...)
	message = append(message, []byte("Content-Type: text/plain; charset=UTF-8\r\n\r\n")...)
	message = append(message, []byte(string(body)+"\r\n")...)
	message = append(message, []byte("--"+boundary+"\r\n")...)
	for _, image := range strings.Split(images, ",") {
		message = append(message, []byte("Content-Type: image/jpeg\r\n")...)
		message = append(message, []byte("Content-Transfer-Encoding: base64\r\n")...)
		message = append(message, []byte("Content-Disposition: attachment; filename=\""+image+"\"\r\n\r\n")...)
		imageContent, err := ioutil.ReadFile(image)
		if err != nil {
			fmt.Println("Error reading image "+image+" content:", err)
			continue
		}
		message = append(message, []byte(base64.StdEncoding.EncodeToString(imageContent)+"\r\n")...)
		message = append(message, []byte("--"+boundary+"--\r\n")...)
	}
	message = append(message, []byte(".")...)
	return message
}
