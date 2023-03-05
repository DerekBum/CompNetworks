package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var serverHost = flag.String("host", "127.0.0.1", "Host of the server")
var serverPort = flag.String("port", "8081", "Port of the server")
var fileName = flag.String("file", "README.md", "Name of the file")

func main() {
	flag.Parse()
	correctURL := fmt.Sprintf("%s:%s", *serverHost, *serverPort)
	filenameURL := url.URL{Host: correctURL, Path: fmt.Sprintf("/%s", *fileName)}
	conn, err := net.Dial("tcp", correctURL)
	if err != nil {
		panic(err)
	}
	req := http.Request{Method: "GET", URL: &filenameURL}
	dumpRequest, err := httputil.DumpRequest(&req, false)
	if err != nil {
		panic(err)
	}
	_, err = conn.Write(dumpRequest)
	if err != nil {
		panic(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), &req)
	if err != nil {
		panic(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(string(body))
}
