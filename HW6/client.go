package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/jlaffaye/ftp"
)

var addr = flag.String("addr", "192.168.0.105:21", "FTP server address in format \"{host}:{port}\"")
var user = flag.String("user", "San", "Username")
var pwrd = flag.String("pwrd", "123", "User password")
var pathFTP = flag.String("pF", "/catch/pep.png", "Path to file on FTP")
var pathLocal = flag.String("pL", "pep.png", "Path to local file")

func main() {
	var getFiles, uploadFile, downloadFile, create bool
	flag.BoolVar(&getFiles, "get", false, "Get all files from FTP")
	flag.BoolVar(&create, "create", false, "Create directory on FTP")
	flag.BoolVar(&uploadFile, "upload", false, "Upload a file to FTP")
	flag.BoolVar(&downloadFile, "download", false, "Download file from FTP")

	flag.Parse()

	c, err := ftp.Dial(*addr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	err = c.Login(*user, *pwrd)
	if err != nil {
		log.Fatal(err)
	}

	if create {
		createDir(c, *pathFTP)
	}
	if uploadFile {
		upload(c, *pathLocal, *pathFTP)
	}
	if getFiles {
		getAllFiles(c)
	}
	if downloadFile {
		download(c, *pathLocal, *pathFTP)
	}

	if err := c.Quit(); err != nil {
		log.Fatal(err)
	}
}

func download(c *ftp.ServerConn, pathLocal, pathFTP string) {
	file, err := os.Create(pathLocal)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	reader, err := c.Retr(pathFTP)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer reader.Close()

	body, _ := ioutil.ReadAll(reader)
	file.Write(body)
}

func createDir(c *ftp.ServerConn, pathFTP string) {
	err := c.MakeDir(pathFTP)
	if err != nil {
		log.Fatal("Cant create: " + err.Error())
	}
}

func upload(c *ftp.ServerConn, pathLocal, pathFTP string) {
	file, err := os.Open(pathLocal)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	err = c.Stor(pathFTP, file)
	if err != nil {
		log.Fatal("Cant store: " + err.Error())
	}
}

func getAllFiles(c *ftp.ServerConn) {
	entries, err := c.List("/")
	if err != nil {
		log.Fatal(err)
	}

	padding := ""
	for _, entry := range entries {
		printDir(c, entry, padding, "/")
	}
}

func printDir(c *ftp.ServerConn, entry *ftp.Entry, padding string, path string) {
	fmt.Println(padding + " " + entry.Name)
	if entry.Type == ftp.EntryTypeFolder {
		entries, err := c.List(path + entry.Name)
		if err != nil {
			return
		}
		for _, subEntry := range entries {
			printDir(c, subEntry, "--"+padding, path+entry.Name+"/")
		}
	}
}
