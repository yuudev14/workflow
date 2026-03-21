package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/mq"
	"github.com/yuudev14-workflow/workflow-service/internal/interface/api"
	"github.com/yuudev14-workflow/workflow-service/internal/interface/grpc"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Message struct {
	sender *websocket.Conn
	data   []byte
}
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var hub = Hub{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan Message),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

func (h *Hub) run() {
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
				if client != message.sender {
					err := client.WriteMessage(websocket.TextMessage, message.data)
					if err != nil {
						client.Close()
						delete(h.clients, client)
					}

				}

			}
		}
	}
}

func wsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub.register <- conn

	defer func() {
		hub.unregister <- conn
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

		hub.broadcast <- Message{
			data:   []byte(response),
			sender: conn,
		}

	}
}

// @title 	Workflow Service API
// @version	1.0
// @description A Workflow Service in Go using Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	settings := environment.Setup()
	logging.Setup(settings.LOGGER_MODE)
	sqlxDB, err := db.SetupDB(settings.DB_URL)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}

	mqInstance := mq.ConnectToMQ(settings.MQ_URL, settings.SenderQueueName, settings.ReceiverQueueName)
	defer mqInstance.MQConn.Close()
	defer mqInstance.MQChannel.Close()
	go hub.run()
	app := api.InitRouter(sqlxDB, mqInstance)
	go grpc.SetupGRPCServer()
	app.GET("/ws", wsHandler)
	go app.Run()
	select {}
}
