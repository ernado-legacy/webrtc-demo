package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var messages = make(chan Message)
var connections = make(map[string]*websocket.Conn)

type ID struct {
	ID string `json:"id"`
}

type Message struct {
	ID   string
	Body interface{}
}

type Description struct {
	Description interface{} `json:"description"`
}

type Candidate struct {
	Candidate interface{} `json:"candidate"`
}

type Target struct {
	Target string `json:"target"`
}

func Translate() {
	for message := range messages {
		log.Println("message for", message.ID)
		conn, ok := connections[message.ID]
		if !ok {
			log.Println(message.ID, "not found")
			continue
		}
		conn.WriteJSON(message.Body)
	}
}

func Realtime(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var target string
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	b := make([]byte, 4)
	rand.Read(b)
	id := hex.EncodeToString(b)

	connections[id] = conn
	defer func() {
		delete(connections, id)
		conn.Close()
		log.Println("[realtime] closed")
	}()

	conn.WriteJSON(ID{id})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var m map[string]interface{}
		json.Unmarshal(message, &m)
		if v, ok := m["target"]; ok {
			target = v.(string)
			log.Println("got target", target)
		}
		if _, ok := m["description"]; ok {
			log.Println("got description")
			messages <- Message{ID: target, Body: m}
		}
		if _, ok := m["candidate"]; ok {
			log.Println("got candidate")
			messages <- Message{ID: target, Body: m}
		}
	}

}

func main() {
	router := httprouter.New()
	router.ServeFiles("/static/*filepath", http.Dir("."))
	router.GET("/realtime", Realtime)
	go Translate()
	log.Println("listening on :5555")
	log.Fatal(http.ListenAndServe(":5555", router))
}
