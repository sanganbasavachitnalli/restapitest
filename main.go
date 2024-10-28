package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	uniqueMap = make(map[int]struct{})
	m         sync.Mutex
	logfile   *os.File
	count     int
)

func init() {
	fmt.Println("Initializing...")
	var err error
	logfile, err = os.OpenFile("uniqueReqLog.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	go scheduleLogs()
}

func scheduleLogs() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.Lock()
		message := fmt.Sprintf("Unique requests received for the minute: %d\n", count)
		if _, err := logfile.WriteString(message); err != nil {
			log.Printf("Error writing to log file: %v", err)
		}
		count = 0
		m.Unlock()
	}
}

func verveHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("id")
	if key == "" {
		http.Error(w, "ID parameter is missing", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(key)
	if err != nil {
		http.Error(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	m.Lock()
	if _, exists := uniqueMap[id]; !exists {
		uniqueMap[id] = struct{}{}
		count++
	}
	m.Unlock()

	endpoint := r.URL.Query().Get("endpoint")
	method := strings.ToUpper(r.URL.Query().Get("method"))

	if endpoint != "" {
		var success bool
		if method == http.MethodGet {
			_, success = makeGetReq(endpoint)
		} else if method == http.MethodPost {
			_, success = makePostReq(endpoint)
		}

		if success {
			w.Write([]byte("OK"))
			return
		}
		w.Write([]byte("Failed"))
		return
	}

	w.Write([]byte("OK"))
}

func makeGetReq(endpoint string) ([]byte, bool) {
	resp, err := http.Get(endpoint)
	if err != nil {
		log.Printf("GET request failed for endpoint %s: %v", endpoint, err)
		return nil, false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed reading GET response: %v", err)
		return nil, false
	}
	return data, true
}

type PayloadStruct struct {
	RandomInt int `json:"randomInt"`
}

func makePostReq(endpoint string) ([]byte, bool) {
	payload := PayloadStruct{RandomInt: 1}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		return nil, false
	}

	resp, err := http.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Printf("POST request failed for endpoint %s: %v", endpoint, err)
		return nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed reading POST response: %v", err)
		return nil, false
	}
	return body, true
}

func main() {
	http.HandleFunc("/api/verve/accept", verveHandle)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
