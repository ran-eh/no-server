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
	"no-server/sub"
	"strconv"
	"sync"
)

// Objects and methods that can be mocked for testing.
var inject = struct {
	storage   store.Store
	sendSteps func(w http.ResponseWriter, file store.File, version int)
	ps        sub.PubSubber
}{
	// In production, editor steps (edit history) is store using the in-memory-storage
	storage: inmemorystore.New(nil),
	// Method used web handlers to return editor updates to the client.
	sendSteps: sendSteps,
	// subscription service for publishing edits to subscribers
	ps: sub.NewPubSub(),
}

// An editor instance.  A single editor instance can be shared between multiple clients.
type instance struct {
	// Instance history (edit steps since creation).
	File store.File
	// pubsub topic on which editor updates are published to subscribers
	TopicName string
	lock      sync.RWMutex
}

// Create a new editor instance
func newInstance() *instance {
	// create storage file for editor history
	f := inject.storage.NewFile()
	// Create pubsub topic
	inject.ps.NewTopic(f.Name())
	return &instance{File: f, TopicName: f.Name()}
}

var instances = map[string]*instance{}

// Web calls return editor update history to the client using this method.
// Takes the full edit history file, and the version up to which the client
// is in sync, and returns edits that happened after that version.
func sendSteps(w http.ResponseWriter, file store.File, version int) {
	w.Header().Set("Content-Type", "application/json")
	steps, err := file.StepsSince(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error: %s", err.Error())
		return
	}
	msg := map[string]interface{}{
		"fileName": file.Name(),
		"steps":    steps,
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// Handle POST / new calls.  The handler creates a new editor history
// instance and returns it to the client.
func handleNew(w http.ResponseWriter, _ *http.Request) {
	instance := newInstance()
	instances[instance.TopicName] = instance
	inject.sendSteps(w, instance.File, 0)
}

// Handle GET calls.  The call is used by a client to obtain
// edits that occurred since it's last sync.
func handleGet(w http.ResponseWriter, req *http.Request) {
	fileName := req.FormValue("name")
	if fileName == "" {
		msg := "invalid fileName: \"\""
		http.Error(w, msg, http.StatusBadRequest)
		log.Printf("Error: %s", msg)
		return
	}
	instance, found := instances[fileName]
	if !found {
		msg := "instance not found: " + fileName
		http.Error(w, msg, http.StatusNotFound)
		log.Printf("Error: %s", msg)
		return
	}
	// Client is in sync up to this version.
	versionStr := req.FormValue("version")
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil || version < 0 {
		msg := "invalid version: " + versionStr
		http.Error(w, msg, http.StatusBadRequest)
		log.Printf("Error: %s", msg)
		return
	}

	instance.lock.RLock()
	defer instance.lock.RUnlock()
	inject.sendSteps(w, instance.File, int(version))
}

// Query data used in updates send by a client
type updateInfo struct {
	ClientID int    `json:"clientID"`
	FileName string `json:"fileName"`
	// Version up to which the client is in sync
	ClientVersion int `json:"version"`
	// Edit history since last sync
	ClientSteps []interface{} `json:"steps"`
}

func (u updateInfo) validate() error {
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
		log.Printf("Error: %s", err.Error())
		return
	}
	if err := info.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error: %s", err.Error())
		return
	}

	instance, found := instances[info.FileName]
	if !found {
		msg := "instance not found: "+info.FileName
		http.Error(w, msg, http.StatusNotFound)
		log.Printf("Error: %s", msg)
		return
	}
	instance.lock.Lock()
	defer instance.lock.Unlock()
	// If the client and server version are in sync, add edits provided
	// by the client to the edit history on the server.  Otherwise the client
	// will need to rebase outstanding edits and send it's edits again.
	if instance.File.Version() == info.ClientVersion {
		log.Printf("%s: Server += %d steps from client %d", info.FileName, len(info.ClientSteps), info.ClientID)
		instance.File.AddSteps(info.ClientSteps, info.ClientID)
		if err := inject.ps.Publish(instance.TopicName); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Printf("Error: %s", err.Error())
			return
		}
	} else {
		log.Printf("%s: client %d needs to rebase from %v to %v",
			info.FileName, info.ClientID, info.ClientVersion, instance.File.Version())
	}
	inject.sendSteps(w, instance.File, info.ClientVersion)
}

// Root handler
func handler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if req.Method == "POST" && req.URL.Path == "/new" {
		handleNew(w, req)
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
	log.Printf("Error: invalid method: %v", req.Method)
}

// Web socket handler.  Used by a client to subscribe to editor updates posted by other
// clients.
func serveWs(w http.ResponseWriter, req *http.Request) {
	fileName := req.FormValue("name")
	if fileName == "" {
		msg := "invalid fileName: \"\""
		http.Error(w, msg, http.StatusBadRequest)
		log.Printf("Error: %s", msg)
		return
	}
	log.Printf("ws connection requested for %s", fileName)

	// Establish websocket connection
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = func(_ *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if hsError, ok := err.(websocket.HandshakeError); !ok {
			log.Println(hsError)
			return
		}
		log.Println(err)
		return
	}

	// Create an edit history subscription for the file name topic
	if err = inject.ps.Subscribe(fileName, ws); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error: %s", err.Error())
	} else {
		log.Printf("ws connection established for %s ", fileName)
	}
}

var addr = flag.String("addr", ":8000", "http service address")

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	http.HandleFunc("/ws", serveWs)
	log.Printf("Editor service starting at %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
