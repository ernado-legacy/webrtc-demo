package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"html/template"
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

type Offer struct {
	Offer interface{} `json:"offer"`
}

func Translate() {
	for message := range messages {
		conn, ok := connections[message.ID]
		if !ok {
			if message.ID != "" {
				log.Println(message.ID, "not found")
			}
			continue
		}
		log.Println("message for", message.ID)
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

	b := make([]byte, 2)
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
		var t string
		json.Unmarshal(message, &m)
		if v, ok := m["target"]; ok {
			target = v.(string)
			t = "target"
			messages <- Message{ID: target, Body: Target{id}}
		}
		if _, ok := m["description"]; ok {
			t = "description"
			messages <- Message{ID: target, Body: m}
		}
		if _, ok := m["candidate"]; ok {
			t = "candidate"
			messages <- Message{ID: target, Body: m}
		}
		if _, ok := m["offer"]; ok {
			t = "offer"
			messages <- Message{ID: target, Body: m}
		}
		if _, ok := m["answer"]; ok {
			t = "answer"
			messages <- Message{ID: target, Body: m}
		}
		if t == "" {
			log.Println("unknown message", m)
		}
		log.Printf("[%s] %s -> %s", t, id, target)
	}

}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	indexTemplate, err := template.ParseFiles("static/index.html")
	if err != nil {
		log.Println(err)
		return
	}
	data := make(map[string]interface{})
	indexTemplate.Execute(w, data)
}

func main() {
	router := httprouter.New()
	router.ServeFiles("/static/*filepath", http.Dir("static"))
	router.GET("/realtime", Realtime)
	router.GET("/", Index)
	go Translate()
	log.Println("listening on :5555")
	log.Fatal(http.ListenAndServe(":5555", router))
}
