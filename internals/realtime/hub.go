package realtime

import (
	"log"
	"sync"

	"recallo/internals/models"
)

type Hub struct {
	Clients map[int64]map[*Client]struct{} `json:"clients"`
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Clients: make(map[int64]map[*Client]struct{}),
	}
}

func (h Hub) BroadcastToAll(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.Clients {
		for client := range clients {
			client.SendEvent(event)
		}
	}
}

func (h Hub) GetClients(userID int64) ([]*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.Clients[userID]

	if !ok || len(conns) < 1 {
		return nil, false
	}

	clients := make([]*Client, 0, len(conns))
	for client := range conns {
		clients = append(clients, client)
	}

	return clients, true
}

func (h *Hub) SendEventToUserIds(userIds []int64, sendId int64, eventType EventType, payload map[string]any) {
	for _, userIds := range userIds {
		h.mu.RLock()
		clients, ok := h.Clients[userIds]
		h.mu.RUnlock()

		if !ok {
			continue
		}

		for client := range clients {
			if client.User.ID == sendId {
				continue
			}
			client.SendEvent(Event{
				eventType,
				payload,
			})
		}
	}
}

func (h *Hub) RegisterClientConnection(client *Client) {
	h.mu.Lock()

	conns, ok := h.Clients[client.User.ID]
	if !ok {
		conns = make(map[*Client]struct{})
		h.Clients[client.User.ID] = conns
	}

	conns[client] = struct{}{}
	firstConn := len(conns) == 1
	h.mu.Unlock()

	if firstConn {
		// Broadcast online presence to all connected clients.
		h.BroadcastToAll(Event{
			EventType: EventUserOnline,
			Payload:   client.User.ToMap(),
		})

		// Silently mark all incoming undelivered messages as delivered in the DB.
		// A single UPDATE query — no loops, no WebSocket fanout to the sender.
		// Senders see the ✓✓ state the next time they load the conversation via REST.
		go func() {
			if err := models.MarkAllIncomingMessagesAsDelivered(client.User.ID); err != nil {
				log.Printf("[HUB] failed to mark messages delivered for user %d: %v", client.User.ID, err)
			}
		}()
	}
}
