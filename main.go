package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"no-server/store"
)

var files = store.New()

// All handled calls use this method to return a filename and a step array.
func sendSteps(w http.ResponseWriter, steps []interface{}, fileName string) {
	w.Header().Set("Content-Type", "application/json")
	msg := map[string]interface{}{
		"fileName": fileName,
		"steps":    steps,
	}
	json.NewEncoder(w).Encode(msg)
}

func createNewFile(w http.ResponseWriter, req *http.Request) {
	file := files.NewFile()
	sendSteps(w, file.Steps, file.Name)
}

type updateInfo struct {
	ClientID      int           `json:"clientID"`
	FileName      string        `json:"fileName"`
	ClientVersion int           `json:"version"`
	ClientSteps   []interface{} `json:"steps"`
}

func (u updateInfo) validate(req *http.Request) error {
	if u.ClientID <= 0 {
		return fmt.Errorf("Invalid ClientID: %d", u.ClientID)
	}
	if u.FileName == "" {
		return fmt.Errorf("Invalid FileName: %s", u.FileName)
	}
	if u.ClientVersion < 0 {
		return fmt.Errorf("Invalid ClientVersion: %d", u.ClientVersion)
	}
	return nil
}

func handleUpdate(w http.ResponseWriter, req *http.Request) {
	var info updateInfo
	if err := json.NewDecoder(req.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := info.validate(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		fmt.Printf("error: %v\n", err)
		return
	}

	file, err := files.GetFile(info.FileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// steps := files[info.FileName].steps
	// serverVersion := len(files[info.FileName].steps)
	// log.Printf("%s: client %d attempts to push %d steps",
	// 	info.FileName, info.ClientID, len(info.ClientSteps))

	if file.Version() == info.ClientVersion {
		log.Printf("%s: Server += %d steps from client %d", info.FileName, len(info.ClientSteps), info.ClientID)
		file.AddSteps(info.ClientSteps, info.ClientID)
		// log.Printf("%s: Server version %d => %d", info.FileName, serverVersion, len(steps))
	} else {
		log.Printf("%s: client %d needs to rebase from %v to %v",
			info.FileName, info.ClientID, info.ClientVersion, file.Version())
	}
	// log.Printf("%s: client %d <= %d steps, %v => %v\n\n",
	// 	info.FileName, info.ClientID, len(steps)-info.ClientVersion, info.ClientVersion, len(steps))
	stepsToSend, err := file.StepsSince(info.ClientVersion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sendSteps(w, stepsToSend, info.FileName)
}

func handleGet(w http.ResponseWriter, req *http.Request) {
	fileName := req.FormValue("name")
	if fileName == "" {
		http.Error(w, "invalid fileName: \"\"", http.StatusBadRequest)
		return
	}
	file, ok := files[fileName]
	if !ok {
		http.Error(w, fileName+"does not exist", http.StatusNotFound)
		return
	}
	versionStr := req.FormValue("version")
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil || version < 0 {
		http.Error(w, "invalid version: "+versionStr, http.StatusBadRequest)
		return
	}
	sendSteps(w, file.Steps, fileName)
}

func handler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if req.Method == "POST" && req.URL.Path == "/new" {
		createNewFile(w, req)
		return
	}
	if req.Method == "POST" && req.URL.Path == "/update" {
		handleUpdate(w, req)
		return
	}
	if req.Method == "GET" {
		handleGet(w, req)
		return
	}
	switch req.Method {
	case "OPTIONS":
		return
	// case "GET":
	// 	startVersion, err := strconv.ParseInt(req.FormValue("version"), 10, 32)
	// 	if err != nil {
	// 		startVersion = 0
	// 	}
	// 	sendSteps(w, steps, int(startVersion), fileName)
	// 	return
	default:
		fmt.Fprintf(w, "Sorry, only POST, GET, OPTIONS methods are supported: %v\n", req.Method)
		return
	}
}

var addr = flag.String("addr", "localhost:8000", "http service address")

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Printf("Editor service starting at %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
