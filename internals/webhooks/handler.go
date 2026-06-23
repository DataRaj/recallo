// Package webhooks handles inbound LiveKit webhook events.
//
// Architecture:
//   - One HTTP handler (Handler) at POST /webhooks/livekit
//   - Handler validates HMAC via webhook.ReceiveWebhookEvent (LiveKit SDK) which:
//       1. Reads raw body
//       2. Parses + verifies Authorization header JWT
//       3. Validates sha256(body) == token's SHA256 claim (base64-encoded)
//   - Deduplication via webhook_events table (ON CONFLICT (event_id) DO NOTHING)
//   - All event processing is synchronous DB writes — no external API calls in this path
//
// Idempotency guarantee:
//   LiveKit delivers at-least-once with retries. The webhook_events table stores the
//   event_id (LiveKit's globally unique per-event ID). ON CONFLICT DO NOTHING means
//   re-delivery of the same event is a no-op. RowsAffected == 0 → skip processing.
package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	lkauth "github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	lkwebhook "github.com/livekit/protocol/webhook"
	"recallo/db"
)

// Handler is the HTTP handler for inbound LiveKit webhooks.
// Constructed via NewHandler and registered via RegisterRoutes.
type Handler struct {
	// keyProvider implements auth.KeyProvider and maps API key → secret.
	// Used by lkwebhook.ReceiveWebhookEvent for HMAC validation.
	keyProvider lkauth.KeyProvider
}

// NewHandler constructs the webhook handler.
// apiKey and webhookSecret come from configs.LiveKitConfig.
// Internally constructs a SimpleKeyProvider(apiKey, webhookSecret) — the simplest
// key provider that covers single-project deployments.
func NewHandler(apiKey, webhookSecret string) *Handler {
	return &Handler{
		keyProvider: lkauth.NewSimpleKeyProvider(apiKey, webhookSecret),
	}
}

// RegisterRoutes wires the webhook endpoint into the mux.
// No auth middleware — LiveKit authenticates via HMAC, not our JWT.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/livekit", h.handleWebhook)
}

// handleWebhook is the single entry point for all inbound LiveKit webhook events.
//
// Processing sequence:
//  1. lkwebhook.ReceiveWebhookEvent: reads body, validates HMAC, parses proto → *lkproto.WebhookEvent
//  2. Deduplication: INSERT INTO webhook_events ON CONFLICT DO NOTHING; skip if duplicate
//  3. Dispatch to per-event DB writer
//  4. Return HTTP 200 (always — non-200 triggers LiveKit retry, which risks duplicate processing)
//
// Error policy:
//   - HMAC failure → 401 (reject invalid callers)
//   - DB error on deduplication → 500 (LiveKit retries; idempotency handles it)
//   - Dispatch error → log + 200 (don't trigger retry; reconciler fixes divergence)
func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// ── 1. Validate signature + parse event (SDK handles raw body + HMAC) ─────
	// ReceiveWebhookEvent: reads body → validates Authorization JWT + SHA256 checksum
	// → unmarshals protobuf JSON into *lkproto.WebhookEvent.
	// Returns error if signature invalid, header missing, or payload malformed.
	event, err := lkwebhook.ReceiveWebhookEvent(r, h.keyProvider)
	if err != nil {
		log.Printf("[webhooks] signature/parse error: %v", err)
		http.Error(w, "invalid webhook signature or payload", http.StatusUnauthorized)
		return
	}

	eventID := event.GetId()
	eventType := event.GetEvent()

	// ── 2. Deduplication ──────────────────────────────────────────────────────
	// Marshal the proto event back to JSON for storage in webhook_events.payload.
	rawPayload, _ := json.Marshal(event)

	inserted, err := insertWebhookEvent(r.Context(), eventID, eventType, rawPayload)
	if err != nil {
		log.Printf("[webhooks] db error storing event %s: %v", eventID, err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !inserted {
		log.Printf("[webhooks] duplicate event ignored: id=%s type=%s", eventID, eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	// ── 3. Dispatch ───────────────────────────────────────────────────────────
	log.Printf("[webhooks] processing event: id=%s type=%s", eventID, eventType)
	if err := h.dispatch(r.Context(), eventType, event); err != nil {
		// Log but return 200: we've already deduped, so a retry won't help.
		// The reconciler will detect and repair any DB divergence.
		log.Printf("[webhooks] dispatch error for %s id=%s: %v", eventType, eventID, err)
	}

	w.WriteHeader(http.StatusOK)
}

// dispatch routes the parsed event to the appropriate DB writer based on event type.
// All handlers only do DB writes — no external HTTP calls in this path.
//
// Status advancement rule: all UPDATE queries use WHERE status != '<final_status>'
// guards so re-processing the same event is a no-op (idempotent beyond deduplication).
func (h *Handler) dispatch(ctx context.Context, eventType string, event *lkproto.WebhookEvent) error {
	switch eventType {
	case "room_started":
		return handleRoomStarted(ctx, event)
	case "room_finished":
		return handleRoomFinished(ctx, event)
	case "participant_joined":
		return handleParticipantJoined(ctx, event)
	case "participant_left":
		return handleParticipantLeft(ctx, event)
	default:
		// Unknown event types: not errors — LiveKit may add new types.
		log.Printf("[webhooks] unhandled event type: %s", eventType)
		return nil
	}
}

// ── Event handlers ────────────────────────────────────────────────────────────

// handleRoomStarted transitions room status from 'draft' → 'live'.
// WHERE status = 'draft' ensures this is idempotent: room_started re-delivered
// after status is already 'live' is a no-op.
func handleRoomStarted(ctx context.Context, event *lkproto.WebhookEvent) error {
	if event.Room == nil {
		return fmt.Errorf("webhooks.handleRoomStarted: missing room in event")
	}
	roomName := event.Room.Name

	_, err := db.DB.ExecContext(ctx, `
		UPDATE rooms
		SET status = 'live', started_at = $1
		WHERE livekit_room_name = $2
		  AND status = 'draft'
	`, time.Now().UTC(), roomName)
	if err != nil {
		return fmt.Errorf("webhooks.handleRoomStarted: db: %w", err)
	}

	log.Printf("[webhooks] room_started: room=%s → status=live", roomName)
	return nil
}

// handleRoomFinished transitions room status to 'ended'.
// WHERE status != 'ended' prevents double-setting ended_at if event is re-delivered.
func handleRoomFinished(ctx context.Context, event *lkproto.WebhookEvent) error {
	if event.Room == nil {
		return fmt.Errorf("webhooks.handleRoomFinished: missing room in event")
	}
	roomName := event.Room.Name

	_, err := db.DB.ExecContext(ctx, `
		UPDATE rooms
		SET status = 'ended', ended_at = $1
		WHERE livekit_room_name = $2
		  AND status != 'ended'
	`, time.Now().UTC(), roomName)
	if err != nil {
		return fmt.Errorf("webhooks.handleRoomFinished: db: %w", err)
	}

	log.Printf("[webhooks] room_finished: room=%s → status=ended", roomName)
	return nil
}

// handleParticipantJoined upserts a participant attendance record.
// ON CONFLICT DO UPDATE handles out-of-order delivery (left before joined scenarios):
// if a left event already created the row, joining overwrites it and clears left_at.
func handleParticipantJoined(ctx context.Context, event *lkproto.WebhookEvent) error {
	if event.Room == nil || event.Participant == nil {
		return fmt.Errorf("webhooks.handleParticipantJoined: missing room or participant in event")
	}

	roomName := event.Room.Name
	identity := event.Participant.Identity
	displayName := event.Participant.Name

	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO room_participants (room_livekit_name, identity, display_name, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_livekit_name, identity)
		DO UPDATE SET joined_at = EXCLUDED.joined_at, left_at = NULL
	`, roomName, identity, displayName, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("webhooks.handleParticipantJoined: db: %w", err)
	}

	log.Printf("[webhooks] participant_joined: room=%s identity=%s", roomName, identity)
	return nil
}

// handleParticipantLeft sets left_at on the attendance record.
// ON CONFLICT handles the case where participant_left arrives before participant_joined
// (out-of-order delivery): the row is upserted with left_at set, and participant_joined
// will later clear left_at when it arrives.
func handleParticipantLeft(ctx context.Context, event *lkproto.WebhookEvent) error {
	if event.Room == nil || event.Participant == nil {
		return fmt.Errorf("webhooks.handleParticipantLeft: missing room or participant in event")
	}

	roomName := event.Room.Name
	identity := event.Participant.Identity
	displayName := event.Participant.Name

	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO room_participants (room_livekit_name, identity, display_name, left_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_livekit_name, identity)
		DO UPDATE SET left_at = EXCLUDED.left_at
	`, roomName, identity, displayName, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("webhooks.handleParticipantLeft: db: %w", err)
	}

	log.Printf("[webhooks] participant_left: room=%s identity=%s", roomName, identity)
	return nil
}

// ── Repository ────────────────────────────────────────────────────────────────

// insertWebhookEvent persists an event_id for idempotency checking.
// Returns (true, nil)  → newly inserted — process this event.
// Returns (false, nil) → duplicate (ON CONFLICT hit) — skip processing.
// Returns (false, err) → real DB error.
var insertWebhookEvent = func(ctx context.Context, eventID, eventType string, payload []byte) (bool, error) {
	result, err := db.DB.ExecContext(ctx, `
		INSERT INTO webhook_events (event_id, event_type, payload, received_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, eventType, payload)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
