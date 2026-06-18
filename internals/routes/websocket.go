package routes

import (
	"context"
	"net/http"
	"strings"
	"time"

	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/models"
	"recallo/internals/realtime"
	"recallo/internals/utils"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func HandleWebSocketConnection(hub *realtime.Hub, w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get(middleware.CtxAuthorization)
	if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		logger.App.Printf("[WEBSOCKET] error=missing_or_invalid_auth_header remote=%s", r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "User is unauthorized", nil)
		return
	}

	accessToken := strings.TrimSpace(authHeader[7:])
	userId, _, _, err := utils.VerifyJWT(accessToken)
	if err != nil {
		logger.App.Printf("[WEBSOCKET] error=invalid_jwt remote=%s err=%v", r.RemoteAddr, err)
		utils.JSON(w, http.StatusUnauthorized, false, "User is unauthorized", nil)
		return
	}

	user, err := models.GetUserByID(userId)
	if err != nil {
		logger.App.Printf("[WEBSOCKET] error=user_not_found user_id=%d err=%v", userId, err)
		utils.JSON(w, http.StatusUnauthorized, false, "User is unauthorized", nil)
		return
	}

	opts := websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	}

	conn, err := websocket.Accept(w, r, &opts)
	if err != nil {
		if user != nil {
			logger.App.Printf("[WEBSOCKET] error=accept_failed user_id=%d err=%v", user.ID, err)
		} else {
			logger.App.Printf("[WEBSOCKET] error=accept_failed remote=%s err=%v", r.RemoteAddr, err)
		}
		utils.JSON(w, http.StatusInternalServerError, false, "Could not form a websocket connection", nil)
		return
	}

	logger.App.Printf("[WEBSOCKET] success connected user_id=%d remote=%s", user.ID, r.RemoteAddr)

	client := realtime.NewClient(user, conn)

	realtime.NewHub().RegisterClientConnection(client)
	realtime.NewHub().SentCurrentClients(client)

	defer func() {
		realtime.NewHub().UnRegisterClientConnection(client)
		client.Close()
	}()

	ctx, cancel := context.WithCancel(r.Context())

	defer cancel()

	go Heartbeat(ctx, *client)

	go writePump(ctx, client)

	readPump(ctx, cancel, hub, client)
}

func Heartbeat(ctx context.Context, client realtime.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := client.Conn.Ping(ctx)
			if err != nil {
				logger.App.Printf("[WEBSOCKET] error=ping_failed client_id=%d err=%v", client.User.ID, err)
				cancel()
				return
			}
			cancel()
			client.Send <- realtime.Event{
				realtime.EventHeartbeat,
				nil,
			}
		}
	}
}

func writePump(ctx context.Context, client *realtime.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-client.Send:
			if !ok {
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			_ = wsjson.Write(writeCtx, client.Conn, event)
			cancel()
		}
	}
}

func readPump(ctx context.Context, cancelCtx context.CancelFunc, hub *realtime.Hub, client *realtime.Client) {
	defer cancelCtx()
	defer func() {
		r := recover()
		if r != nil {
			logger.App.Printf("[WEBSOCKET] error=readpump_panic client_id=%d panic=%v", client.User.ID, r)
		}
	}()

	for {
		var event realtime.Event
		err := wsjson.Read(ctx, client.Conn, &event)
		if err != nil {
			logger.App.Printf("[WEBSOCKET] error=read_failed client_id=%d err=%v", client.User.ID, err)
			return
		}
		
		// TODO: Handle incoming event
	}
}
