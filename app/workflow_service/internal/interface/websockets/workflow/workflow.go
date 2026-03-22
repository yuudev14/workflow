package workflow_websockets

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var workflowStatusWsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WorkflowStatusWsMessage struct {
	// sender *websocket.Conn
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type WorkflowStatusWsHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan WorkflowStatusWsMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var WorkflowStatusWsHubInstance = WorkflowStatusWsHub{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan WorkflowStatusWsMessage),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

func (h *WorkflowStatusWsHub) Run() {
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
				err := client.WriteJSON(message.Data)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}

				// }

			}
		}
	}
}

func (h *WorkflowStatusWsHub) AssignValueToBroadcast(data WorkflowStatusWsMessage) {
	WorkflowStatusWsHubInstance.broadcast <- data
}

func WorkflowWsHandler(c *gin.Context) {
	conn, err := workflowStatusWsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	WorkflowStatusWsHubInstance.register <- conn

	defer func() {
		WorkflowStatusWsHubInstance.unregister <- conn
	}()

	select {}
	// commenting below for now. reference
	// for {
	// _, msg, err := conn.ReadMessage()
	// if err != nil {
	// 	break
	// }

	// var response string

	// switch string(msg) {
	// case "create":
	// 	response = "Create request received"
	// case "update":
	// 	response = "Update request received"
	// case "delete":
	// 	response = "Delete request received"
	// default:
	// 	response = "Unknown request"
	// }

	// WorkflowHub.AssignValueToBroadcast(Message{
	// 	Data: response,
	// 	// sender: conn,
	// })

	// }
}
