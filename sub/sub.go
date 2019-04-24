// This package implements a websocket based subscription service for edits
package sub

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

// Interface to allow mocking the pubsub package for unit testing
// for other packages that use this package.
type PubSubber interface {
	NewTopic(string)
	Publish(string) error
	Subscribe(string, *websocket.Conn) error
	Unsubscribe(string, *websocket.Conn) error
}

// The production implementation of PubSubber
type PubSub struct {
	Topics map [string]*topic
}

func NewPubSub() *PubSub{
	topics := map[string]*topic{}
	return &PubSub{Topics: topics}
}

func (ps *PubSub) NewTopic(topicName string) {
	t := &topic{
		Publish:   make(chan bool),
		Subscribe: make(chan *websocket.Conn),
		Unsubscribe: make(chan *websocket.Conn),
		subscribers: make(map[*websocket.Conn]bool),
	}
	go t.Run()
	ps.Topics[topicName] = t
}

func (ps *PubSub) getTopic(topicName string) (topic *topic, err error){
	topic = ps.Topics[topicName]
	if topic == nil {
		err = fmt.Errorf("topic %s not found", topicName)
	}
	return
}

func (ps *PubSub) Publish(topicName string) error {
	topic, err := ps.getTopic(topicName)
	if err != nil {
		return err
	}
	topic.Publish <- true
	return nil
}

func (ps *PubSub) Subscribe(topicName string, ws *websocket.Conn) error {
	topic, err := ps.getTopic(topicName)
	if err != nil {
		return err
	}
	topic.Subscribe <- ws
	return nil
}

func (ps *PubSub) Unsubscribe(topicName string, ws *websocket.Conn) error {
	topic, err := ps.getTopic(topicName)
	if err != nil {
		return err
	}
	topic.Unsubscribe <- ws
	return nil
}

// The topic type implements the subscription service
// for one topic.
type topic struct {
	subscribers map[*websocket.Conn]bool
	// Publish notification to subscribers does not carry a payload.  It is
	// a subscriber's responsibility to make a query the service for new
	// edits.
	Publish chan bool
	Subscribe chan *websocket.Conn
	Unsubscribe chan *websocket.Conn
}

func (t *topic) Run() {
	log.Println("topic service started")
	for {
		select {
		case subscriber := <-t.Subscribe:
			t.subscribers[subscriber] = true
			log.Println("subscriber added to topic")
		case subscriber := <-t.Unsubscribe:
			if _, ok := t.subscribers[subscriber]; ok {
				delete(t.subscribers, subscriber)
				log.Println("subscriber removed")
			}
		case <-t.Publish:
			log.Println("Publish called")
			for client := range t.subscribers {
				if err := client.WriteMessage(websocket.TextMessage, []byte("hey")); err != nil {
					log.Printf("write error: %v", err)
					delete(t.subscribers, client)
					log.Println("subscriber removed due to connection problem")
				}
			}
			log.Printf("publish to %d subscribers", len(t.subscribers))
		}
	}
}
