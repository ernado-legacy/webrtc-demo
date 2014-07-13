package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
var rooms = make(map[string]*Room)

type ID struct {
	ID string `json:"id"`
}

type Client struct {
	ID         string
	Candidates []Candidate
	Offer      Offer
	Conn       *websocket.Conn
}

func (client Client) Send(message interface{}) (err error) {
	return client.Conn.WriteJSON(message)
}

type Room struct {
	ID          string
	Clients     map[string]Client
	newClients  chan Client
	deadClients chan string
}

func NewRoom(id string) *Room {
	r := &Room{}
	r.ID = id
	r.Clients = make(map[string]Client)
	r.newClients = make(chan Client)
	r.deadClients = make(chan string)
	r.Start()
	return r
}

func NewClient(id string, conn *websocket.Conn) Client {
	log.Println("creating client", id)
	c := Client{}
	c.ID = id
	c.Conn = conn
	return c
}

func (room *Room) Start() {
	log.Println("starting cycle for", room.ID)
	go func() {
		for client := range room.newClients {
			log.Println("adding client", client.ID, "to room", room.ID)
			room.Clients[client.ID] = client

			for k, v := range room.Clients {
				if k == client.ID {
					continue
				}
				go v.Send(map[string]string{"client": client.ID})
				go client.Send(map[string]string{"client": k})
			}
		}

	}()
	go func() {
		for client := range room.deadClients {
			log.Println("removing client", client, "from room", room.ID)
			delete(room.Clients, client)
			for _, v := range room.Clients {
				v.Send(map[string]string{"dead": client})
			}
		}
	}()
	log.Println("started cycle for", room.ID)
}

func (room *Room) Add(client Client) {
	log.Println("sending client to newClients of", room.ID)
	room.newClients <- client
	log.Println("client sent")
}

func (room *Room) Del(id string) {
	room.deadClients <- id
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
	var id string
	var room *Room
	cookie, err := r.Cookie("id")
	if err == nil && cookie != nil {
		id = cookie.Value
	} else {
		b := make([]byte, 2)
		rand.Read(b)
		id = hex.EncodeToString(b)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	connections[id] = conn
	defer func() {
		delete(connections, id)
		conn.Close()
		room.Del(id)
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
		if v, ok := m["description"]; ok {
			t = "description"
			d := Description{v}
			log.Println("description", d)
			messages <- Message{ID: target, Body: m}
		}
		if v, ok := m["candidate"]; ok {
			t = "candidate"
			candidate := Candidate{v}
			log.Println(t, candidate)
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
		if v, ok := m["room"]; ok {
			log.Println("got room")
			t = "room"
			room, ok = rooms[v.(string)]
			if !ok {
				log.Println("bad room", v)
			} else {
				log.Println("adding client")
				room.Add(NewClient(id, conn))
				log.Println("added client")
			}
		}
		if t == "" {
			log.Println("unknown message", m)
		}
		log.Printf("[%s] %s -> %s", t, id, target)
	}

}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b := make([]byte, 12)
	rand.Read(b)
	id := hex.EncodeToString(b)
	roomUrl := fmt.Sprintf("room/%s", id)
	http.Redirect(w, r, roomUrl, http.StatusTemporaryRedirect)
}

func RoomHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	t, err := template.ParseFiles("static/index.html")
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		roomId := p.ByName("room")
		log.Println("room", roomId)
		_, ok := rooms[roomId]
		if !ok {
			log.Println("creating room", roomId)
			rooms[roomId] = NewRoom(roomId)
		}
	}()

	t.Execute(w, nil)
}

func main() {
	router := httprouter.New()
	router.ServeFiles("/static/*filepath", http.Dir("static"))
	router.GET("/realtime", Realtime)
	router.GET("/room/:room", RoomHandler)
	router.GET("/", Index)
	go Translate()
	log.Println("listening on :5555")
	log.Fatal(http.ListenAndServe(":5555", router))
}
