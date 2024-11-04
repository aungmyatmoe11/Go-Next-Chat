package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub *Hub
}

func NewHandler(h *Hub) *Handler {
	return &Handler{hub: h}
}

type CreateRoomReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) CreateRoom(c *gin.Context) {
	var req CreateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.Rooms[req.ID] = &Room{
		ID:      req.ID,
		Name:    req.Name,
		Clients: make(map[string]*Client),
	}

	c.JSON(http.StatusOK, req)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// origin := r.Header.Get("Origin")
		// return origin == "http://localhost:3000" // replace with your own origin
		return true
	},
}

func (h *Handler) JoinRoom(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// *** /ws/JoinRoom/:roomId?userId=1&username=user
	// Retrieve roomId, userId, and username from the request
	roomId := c.Param("roomId")
	userId := c.Query("userId")
	username := c.Query("username")

	room := h.hub.Rooms[roomId]
	if room == nil {
		conn.Close()
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	cl := &Client{
		Conn:     conn,
		Message:  make(chan *Message, 10),
		ID:       userId,
		RoomID:   roomId,
		Username: username,
	}

	m := &Message{
		Content:  "A new user has joined the room",
		RoomID:   roomId,
		UserID:   userId,
		Username: username,
	}

	// ? Register a new client through the register channel
	h.hub.Register <- cl
	// ? Broadcast that message
	h.hub.Broadcast <- m

	// ? writeMessage()
	go cl.writeMessage()

	// ? readMessage()
	cl.readMessage(h.hub)
}
