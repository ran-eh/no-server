package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"no-server/inmemorystore"
	"no-server/store"
	"no-server/wspool"
	"strconv"
	"sync"
)

var inject = struct {
	storage store.Store
	sendSteps func(w http.ResponseWriter, file store.File, version int)
}{
	storage: inmemorystore.New(nil),
	sendSteps: prodSendSteps,
}

// Using a var to allow injecting a mock store for testing,
type instance struct {
	File store.File
	Pool *wspool.Pool
	lock sync.RWMutex
}

func newInstance() *instance{
	pool := wspool.NewPool()
	go pool.Run()
	return &instance{File: inject.storage.NewFile(), Pool: pool}
}

var instances = map[string]*instance{}

func prodSendSteps(w http.ResponseWriter, file store.File, version int) {
	w.Header().Set("Content-Type", "application/json")
	steps, err := file.StepsSince(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg := map[string]interface{}{
		"fileName": file.Name(),
		"steps":    steps,
	}
	_ = json.NewEncoder(w).Encode(msg)
}

func halndleNew(w http.ResponseWriter, _ *http.Request) {
	instance := newInstance()
	instances[instance.File.Name()] = instance
	inject.sendSteps(w, instance.File, 0)
}

type updateInfo struct {
	ClientID      int           `json:"clientID"`
	FileName      string        `json:"fileName"`
	ClientVersion int           `json:"version"`
	ClientSteps   []interface{} `json:"steps"`
}

func (u updateInfo) validate(req *http.Request) error {
	if u.ClientID <= 0 {
		return fmt.Errorf("invalid ClientID: %d", u.ClientID)
	}
	if u.FileName == "" {
		return fmt.Errorf("invalid FileName: %s", u.FileName)
	}
	if u.ClientVersion < 0 {
		return fmt.Errorf("invalid ClientVersion: %d", u.ClientVersion)
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

	instance, found := instances[info.FileName]
	//file, err := oldFiles.GetFile(info.FileName)
	if !found {
		http.Error(w, "instance not found: " + info.FileName, http.StatusNotFound)
		return
	}
	instance.lock.Lock()
	defer instance.lock.Unlock()
	if instance.File.Version() == info.ClientVersion {
		log.Printf("%s: Server += %d steps from client %d", info.FileName, len(info.ClientSteps), info.ClientID)
		instance.File.AddSteps(info.ClientSteps, info.ClientID)
		instance.Pool.Broadcast <- true
	} else {
		log.Printf("%s: client %d needs to rebase from %v to %v",
			info.FileName, info.ClientID, info.ClientVersion, instance.File.Version())
	}
	// log.Printf("%s: client %d <= %d steps, %v => %v\n\n",
	// 	info.FileName, info.ClientID, len(steps)-info.ClientVersion, info.ClientVersion, len(steps))
	inject.sendSteps(w, instance.File, info.ClientVersion)
}

func handleGet(w http.ResponseWriter, req *http.Request) {
	fileName := req.FormValue("name")
	if fileName == "" {
		http.Error(w, "invalid fileName: \"\"", http.StatusBadRequest)
		return
	}
	instance, found := instances[fileName]
	if !found {
		http.Error(w, "instance not found: " + fileName, http.StatusNotFound)
		return
	}
	versionStr := req.FormValue("version")
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil || version < 0 {
		http.Error(w, "invalid version: "+versionStr, http.StatusBadRequest)
		return
	}

	instance.lock.RLock()
	defer instance.lock.RUnlock()
	inject.sendSteps(w, instance.File, int(version))
}

func handler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if req.Method == "POST" && req.URL.Path == "/new" {
		halndleNew(w, req)
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
	if req.Method == "OPTIONS" {
		return
	}
	_, _ = fmt.Fprintf(w, "Sorry, only POST, GET, OPTIONS methods are supported: %v\n", req.Method)
}

var addr = flag.String("addr", ":8000", "http service address")

func serveWs(w http.ResponseWriter, req *http.Request) {
	fileName := req.FormValue("name")
	if fileName == "" {
		http.Error(w, "invalid fileName: \"\"", http.StatusBadRequest)
		return
	}
	log.Printf("ws connection requested for %s", fileName)

	instance, found := instances[fileName]
	//file, err := oldFiles.GetFile(info.FileName)
	if !found {
		http.Error(w, "instance not found: " + fileName, http.StatusNotFound)
		return
	}


	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = func(_ *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
			return
		}
	}
	instance.Pool.Register <- ws
	log.Printf("ws connection established for %s ", fileName)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	http.HandleFunc("/ws", serveWs)
	log.Printf("Editor service starting at %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
