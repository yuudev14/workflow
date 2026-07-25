package ws

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// writeTimeout bounds a single frame write; sendBuffer is how many events a
	// client may fall behind before it is dropped. Together they keep one slow
	// client from stalling the broadcast loop for everyone else.
	writeTimeout = 10 * time.Second
	sendBuffer   = 64
)

// newUpgrader restricts which pages may open a socket. CORS does not apply to
// WebSockets, so without this check any site could open an authenticated
// connection using the visitor's cookie.
func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Non-browser clients (curl, the CLI) send no Origin at all;
			// they still have to pass the auth middleware.
			if origin == "" {
				return true
			}
			for _, allowed := range allowedOrigins {
				if strings.EqualFold(origin, allowed) {
					return true
				}
			}
			return false
		},
	}
}

// client pairs a connection with its buffered send queue; a dedicated
// writePump goroutine drains the queue so writes never block the hub.
type client struct {
	conn *websocket.Conn
	send chan any
}

func (c *client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := c.conn.WriteJSON(message); err != nil {
			// drain the rest so the hub's channel sends never block; the read
			// loop notices the closed conn and unregisters us
			for range c.send {
			}
			return
		}
	}
}

// Hub fans playbook/task status updates out to connected websocket clients.
// It implements contracts.StatusBroadcaster.
type Hub struct {
	clients    map[*client]bool
	broadcast  chan any
	register   chan *client
	unregister chan *client
	upgrader   websocket.Upgrader
}

func NewHub(allowedOrigins []string) *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan any),
		register:   make(chan *client),
		unregister: make(chan *client),
		upgrader:   newUpgrader(allowedOrigins),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case message := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- message:
				default:
					// client is too far behind — drop it rather than block
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(data any) {
	h.broadcast <- data
}

// ServeWS upgrades the request and keeps the connection registered until the
// client disconnects. Incoming messages are read and discarded so pings and
// close frames are processed.
func (h *Hub) ServeWS(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cl := &client{conn: conn, send: make(chan any, sendBuffer)}
	h.register <- cl
	go cl.writePump()

	defer func() {
		h.unregister <- cl
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
