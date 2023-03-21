package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type CacheUnit struct {
	respBody   []byte
	respStatus int
	addingTime string
	eTag       string
}

var bList = flag.String("bl", "blacklist.txt", "Black list of sites and domains")
var addr = flag.String("addr", ":8081", "Addr of the localhost server. Example: \":8081\"")

func main() {
	flag.Parse()

	// Create a log file
	logFile, err := os.OpenFile("proxy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	cache := map[string]CacheUnit{}

	file, _ := os.Open(*bList)
	scanner := bufio.NewScanner(file)
	var blackList []string
	for scanner.Scan() {
		blackList = append(blackList, scanner.Text())
	}
	file.Close()

	// Start the server and listen on port
	http.HandleFunc("/", handleRequest(logFile, cache, blackList))
	http.ListenAndServe(*addr, nil)
}

func handleRequest(logFile *os.File, cache map[string]CacheUnit, blackList []string) http.HandlerFunc {
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

		if cacheUnit, inCache := cache[targetURL]; inCache {
			targetReq.Header.Add("If-Modified-Since", cacheUnit.addingTime)
			if cacheUnit.eTag != "" {
				targetReq.Header.Add("If-None-Match", cacheUnit.eTag)
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

		// Copy the target response body to the outgoing response
		if targetResp.StatusCode == 304 {
			fromCache, _ := cache[targetURL]
			body = fromCache.respBody
			statusCode = 304
		} else {
			body, err = ioutil.ReadAll(targetResp.Body)
			if err != nil {
				logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, http.StatusInternalServerError)
				logFile.WriteString(logMessage)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			cacheUnit := CacheUnit{respBody: body, respStatus: targetResp.StatusCode, addingTime: time.Now().Format(http.TimeFormat), eTag: targetResp.Header.Get("Etag")}
			cache[targetURL] = cacheUnit
			statusCode = targetResp.StatusCode
		}

		w.Write(body)

		// Log the response
		logMessage := fmt.Sprintf("%s %s %d\n", r.Method, targetURL, statusCode)
		if r.Method == "POST" {
			println(fmt.Sprintf("%v", body))
			logMessage = fmt.Sprintf("%s %s %d\n%s\n", r.Method, targetURL, statusCode, string(body))
		}
		logFile.WriteString(logMessage)
	}
}
