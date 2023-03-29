package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/smtp"
)

var addrTo = flag.String("addrTo", "", "Email address of receiver")
var addrFrom = flag.String("addrFrom", "", "Email address of sender")
var pwrd = flag.String("pwrd", "", "Email password of sender")
var filePath = flag.String("fp", "hello.txt", "File to send (.txt or .html)")
var smtpHost = flag.String("host", "mail.sibnet.ru", "SMTP host")
var smtpPort = flag.String("port", "25", "SMTP port")

func main() {
	flag.Parse()

	from := *addrFrom
	password := *pwrd

	to := []string{
		*addrTo,
	}

	subject := "Subject: Test email from Go!\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body, err := ioutil.ReadFile(*filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	message := []byte(subject + mime)
	message = append(message, body...)

	auth := smtp.PlainAuth("", from, password, *smtpHost)

	err = smtp.SendMail(*smtpHost+":"+*smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email Sent Successfully!")
}
