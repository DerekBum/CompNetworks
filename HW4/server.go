package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var bList = flag.String("bl", "blacklist.txt", "Black list of sites and domains")
var addr = flag.String("addr", ":8081", "Addr of the localhost server. Example: \":8081\"")
var cachePath = flag.String("cache", "./cache", "Path to cache folder")

func main() {
	flag.Parse()

	// Create a log file
	logFile, err := os.OpenFile("proxy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	cache := map[string]string{}
	items, _ := ioutil.ReadDir(*cachePath)
	for _, item := range items {
		file, _ := os.Open(*cachePath + "/" + item.Name())
		scanner := bufio.NewScanner(file)
		scanner.Scan()
		isCurrURL := strings.Split(scanner.Text(), " ")
		if len(isCurrURL) <= 1 {
			file.Close()
			continue
		}
		currURL := isCurrURL[1]
		scanner.Scan()
		isAddingTime := strings.Split(scanner.Text(), "$")
		if len(isAddingTime) <= 1 {
			file.Close()
			continue
		}
		cache[currURL] = *cachePath + "/" + item.Name()
		file.Close()
	}

	file, _ := os.Open(*bList)
	scanner := bufio.NewScanner(file)
	var blackList []string
	for scanner.Scan() {
		blackList = append(blackList, scanner.Text())
	}
	file.Close()

	// Start the server and listen on port
	http.HandleFunc("/", handleRequest(logFile, cache, blackList, *cachePath))
	http.ListenAndServe(*addr, nil)
}

func handleRequest(logFile *os.File, cache map[string]string, blackList []string, cachePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a new request to the target server
		targetURL := r.URL.String()

		for _, blocked := range blackList {
			if strings.Contains(targetURL, blocked) {
				logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, http.StatusForbidden)
				logFile.WriteString(logMessage)
				http.Error(w, "Access denied! This site is blacklisted", http.StatusForbidden)
				return
			}
		}

		targetReq, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, http.StatusInternalServerError)
			logFile.WriteString(logMessage)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var fileName string

		if path, inCache := cache[targetURL]; inCache {

			var addingTime string
			var eTag string

			file, _ := os.Open(path)
			scanner := bufio.NewScanner(file)
			scanner.Scan()
			fileName = path
			scanner.Scan()
			addingTime = strings.Split(scanner.Text(), "$")[1]
			scanner.Scan()
			eTagIs := strings.Split(scanner.Text(), " ")
			if len(eTagIs) > 1 {
				eTag = eTagIs[1]
			}
			file.Close()

			targetReq.Header.Add("If-Modified-Since", addingTime)
			if eTag != "" && eTag != "\n" {
				targetReq.Header.Add("If-None-Match", eTag)
			}
		}

		// Copy headers from the incoming request to the target request
		for header, values := range r.Header {
			for _, value := range values {
				targetReq.Header.Add(header, value)
			}
		}

		// Send the target request and read the response
		client := &http.Client{}

		targetResp, err := client.Do(targetReq)
		if err != nil {
			logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, http.StatusBadGateway)
			logFile.WriteString(logMessage)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer targetResp.Body.Close()

		// Copy headers from the target response to the outgoing response
		for header, values := range targetResp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}

		var body []byte
		var statusCode int

		if fileName == "" {
			code := sha256.New()
			var fileNameInt []byte
			code.Write(fileNameInt)
			sha := base64.URLEncoding.EncodeToString(code.Sum(nil))
			fileName = cachePath + "/" + sha + ".txt"
		}

		// Copy the target response body to the outgoing response
		if targetResp.StatusCode == 304 {
			file, _ := os.Open(fileName)
			scanner := bufio.NewScanner(file)
			scanner.Scan()
			scanner.Scan()
			scanner.Scan()
			for scanner.Scan() {
				body = append(body, scanner.Bytes()...)
			}
			file.Close()
			statusCode = 304
		} else {
			body, err = ioutil.ReadAll(targetResp.Body)
			if err != nil {
				logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, http.StatusInternalServerError)
				logFile.WriteString(logMessage)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			file, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0745)

			file.Write([]byte(fmt.Sprintf("URL: %s\n", targetURL)))
			file.Write([]byte(fmt.Sprintf("LastModified:$%s\n", time.Now().Format(http.TimeFormat))))
			file.Write([]byte(fmt.Sprintf("eTag: %s\n", targetResp.Header.Get("Etag"))))
			file.Write(body)

			file.Close()

			cache[targetURL] = fileName
			statusCode = targetResp.StatusCode
		}

		w.Write(body)

		// Log the response
		logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, statusCode)
		if r.Method == "POST" {
			logMessage = fmt.Sprintf("%s %s %d\n%s\n", r.Method, targetURL, statusCode, string(body))
		}
		logFile.WriteString(logMessage)
	}
}
