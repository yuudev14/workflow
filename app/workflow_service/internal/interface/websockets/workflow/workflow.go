package workflow_websockets

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Message struct {
	// sender *websocket.Conn
	Data []byte
}
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var WorkflowHub = Hub{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan Message),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true

		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				// if client != message.sender {
				err := client.WriteMessage(websocket.TextMessage, message.Data)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}

				// }

			}
		}
	}
}

func (h *Hub) AssignValueToBroadcast(data Message) {
	WorkflowHub.broadcast <- data
}

func WorkflowWsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	WorkflowHub.register <- conn

	defer func() {
		WorkflowHub.unregister <- conn
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var response string

		switch string(msg) {
		case "create":
			response = "Create request received"
		case "update":
			response = "Update request received"
		case "delete":
			response = "Delete request received"
		default:
			response = "Unknown request"
		}

		WorkflowHub.AssignValueToBroadcast(Message{
			Data: []byte(response),
			// sender: conn,
		})

	}
}
