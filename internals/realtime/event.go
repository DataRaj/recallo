package realtime

type EventType string

const (
	EventCurrentUsers EventType = "current_user"
	EventUserOnline   EventType = "online"
	EventUserOffline  EventType = "offline"
	EventNewPrivate   EventType = "new_private"
	EventMessage      EventType = "message"
	EventDelivered    EventType = "delivered"
	EventRead         EventType = "read"
	EventTyping       EventType = "typing"
	EventError        EventType = "error"
	EventHeartbeat    EventType = "heartbeat"

	EventShutdown EventType = "shutdown"
)

type Event struct {
	EventType EventType `json:"event_type"`
	Payload   any       `json:"payload"`
}
