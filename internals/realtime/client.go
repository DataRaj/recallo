package realtime

import (
	"sync"

	"recallo/internals/models"

	"github.com/coder/websocket"
)

type Client struct {
	User *models.User    `json:"user"`
	Conn *websocket.Conn `json:"-"`
	Send chan Event      `json:"-"`
	once sync.Once       `json:"-"`
}

func NewClient(user *models.User, conn *websocket.Conn) *Client {
	return &Client{
		User: user,
		Conn: conn,
		Send: make(chan Event, 512),
	}
}

func (c *Client) SendEvent(event Event) {
	select {
	case c.Send <- event:
	default:
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		if c.Conn != nil {
			c.Conn.Close(websocket.StatusNormalClosure, "Connection closed")
		}
		close(c.Send)
	})
}
