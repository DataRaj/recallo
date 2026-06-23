package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"recallo/db"
	"recallo/internals/configs"
	"recallo/internals/handlers"
	livekit "recallo/internals/livekit"
	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/realtime"
	"recallo/internals/rooms"
	"recallo/internals/routes"
	"recallo/internals/utils"
	"recallo/internals/webhooks"
)

func main() {
	cfg := configs.LoadConfig()

	// ── Logger ────────────────────────────────────────────────────────────────
	// Initialise shared logger — writes to stdout AND logs/app.log.
	closeLog, err := logger.Init()
	if err != nil {
		log.Fatalf("[startup] failed to initialise logger: %v", err)
	}
	defer closeLog()

	logger.App.Printf("[startup] logger initialised — output also tailing logs/app.log")

	// ── Auth ──────────────────────────────────────────────────────────────────
	utils.InitJWT(cfg.JWTSecretKey)
	handlers.InitOAuth(cfg.GithubClientID, cfg.GithubClientSecret, cfg.GithubOAuthRedirectURL)

	// ── Database ──────────────────────────────────────────────────────────────
	if err := db.InitDB(cfg.DatabaseURL, db.DefaultConfig()); err != nil {
		log.Fatalf("[startup] failed to initialise database: %v", err)
	}
	defer db.CloseDBConnection()

	// ── LiveKit service ───────────────────────────────────────────────────────
	// NewService constructs the gRPC RoomServiceClient + token access key.
	// Passed explicitly to every component that needs LiveKit access —
	// no global state, no package-level singletons.
	lkService, err := livekit.NewService(cfg.LiveKit)
	if err != nil {
		log.Fatalf("[startup] failed to initialise livekit service: %v", err)
	}
	logger.App.Printf("[startup] livekit service connected host=%s", cfg.LiveKit.Host)

	// ── Rooms domain ──────────────────────────────────────────────────────────
	// roomsSvc depends on the LiveKitService interface (not the concrete *livekit.Service type).
	// The interface is declared in the rooms package (Kennedy's rule: interface at consumer).
	roomsSvc := rooms.NewService(lkService, cfg.GuestTier)
	roomsHandler := rooms.NewHandler(roomsSvc)

	// ── Webhook handler ───────────────────────────────────────────────────────
	// Needs the webhook secret (separate from API secret) for HMAC validation.
	webhookHandler := webhooks.NewHandler(cfg.LiveKit.APIKey, cfg.LiveKit.WebhookSecret)

	// ── WebSocket hub ─────────────────────────────────────────────────────────
	hub := realtime.NewHub()
	defer hub.Shutdown()

	// ── Session enforcer ──────────────────────────────────────────────────────
	// Background goroutine that ends guest rooms when their session duration elapses.
	// Runs every 60 seconds. Stopped cleanly when ctx is cancelled on shutdown.
	enforcerCtx, stopEnforcer := context.WithCancel(context.Background())
	enforcer := rooms.NewSessionEnforcer(roomsSvc, 60*time.Second)
	go enforcer.Run(enforcerCtx)

	// ── Routes ────────────────────────────────────────────────────────────────
	routeHandler := routes.RegisterRoutes(hub, roomsHandler, webhookHandler)

	// Apply logging middleware on top of the route handler.
	handler := middleware.Loggingmiddleware(routeHandler)

	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.App.Printf("[startup] server listening on %s  env=%s", cfg.HTTPServer.Address, cfg.Env)
		logger.App.Printf("[startup] registered routes:")
		logger.App.Printf("  GET  /api/v1/healthcheck")
		logger.App.Printf("  POST /api/v1/auth/register")
		logger.App.Printf("  POST /api/v1/auth/login")
		logger.App.Printf("  POST /api/v1/auth/refresh-session")
		logger.App.Printf("  POST /api/v1/auth/logout              [protected]")
		logger.App.Printf("  GET  /api/v1/auth/current-user        [protected]")
		logger.App.Printf("  GET  /api/v1/users/{id}               [protected]")
		logger.App.Printf("  GET  /api/v1/conversations            [protected]")
		logger.App.Printf("  POST /api/v1/conversation/private/create             [protected]")
		logger.App.Printf("  GET  /api/v1/conversation/private/{private_id}       [protected]")
		logger.App.Printf("  GET  /api/v1/conversation/private/{private_id}/messages [protected]")
		logger.App.Printf("  POST /api/v1/files/{private_id}       [protected]")
		logger.App.Printf("  GET  /api/v1/files/                   [protected]")
		logger.App.Printf("  GET  /api/v1/ws                       [websocket]")
		// LiveKit / rooms routes
		logger.App.Printf("  POST   /api/v1/rooms                  [guest - public]")
		logger.App.Printf("  GET    /api/v1/rooms/{id}             [guest - public]")
		logger.App.Printf("  DELETE /api/v1/rooms/{id}             [guest - host only]")
		logger.App.Printf("  GET    /api/v1/rooms/{id}/token       [guest - token boundary]")
		logger.App.Printf("  POST   /api/v1/rooms/{id}/extend      [guest - one-time extend]")
		logger.App.Printf("  POST   /webhooks/livekit              [livekit webhook receiver]")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.App.Printf("[server] fatal err=%v", err)
		}
	}()

	sig := <-shutdownCh
	logger.App.Printf("[server] signal received: %v — initiating graceful shutdown", sig)

	// Stop the session enforcer first so it doesn't trigger new EndRoom calls
	// while the HTTP server is draining.
	stopEnforcer()

	// Drain existing HTTP connections (20s window).
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.App.Printf("[server] shutdown error: %v", err)
	} else {
		logger.App.Printf("[server] shut down cleanly")
	}
}
