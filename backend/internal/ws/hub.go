package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"nhooyr.io/websocket"

	applogger "fpgwiki/backend/internal/logger"
)

const (
	maxClientsPerHub = 20
	hubIdleTTL       = 60 * time.Second
)

type Client struct {
	ID     string
	UserID uuid.UUID
	Level  string
	Conn   *websocket.Conn
	Send   chan []byte
}

type inboundMsg struct {
	Client  *Client
	Payload []byte
}

type wsMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Hub struct {
	NodeID     uuid.UUID
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	forceKick  chan uuid.UUID
	inbound    chan inboundMsg
	shutdown   chan struct{}
	done       chan struct{}

	log     zerolog.Logger
	release func(nodeID uuid.UUID)
}

func NewHub(nodeID uuid.UUID, log zerolog.Logger, release func(nodeID uuid.UUID)) *Hub {
	return &Hub{
		NodeID:     nodeID,
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		forceKick:  make(chan uuid.UUID),
		inbound:    make(chan inboundMsg),
		shutdown:   make(chan struct{}),
		done:       make(chan struct{}),
		log:        log,
		release:    release,
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)

	var idleTimer *time.Timer
	var idleTimerC <-chan time.Time

	for {
		select {
		case client := <-h.register:
			if idleTimer != nil {
				stopTimer(idleTimer)
				idleTimer = nil
				idleTimerC = nil
			}

			if len(h.clients) >= maxClientsPerHub {
				select {
				case client.Send <- mustEncode(wsMessage{
					Type: "join_rejected",
					Payload: map[string]string{
						"reason": "doc_full",
					},
				}):
				default:
				}
				close(client.Send)
				continue
			}

			h.clients[client.ID] = client
			h.log.Info().
				Str(applogger.NodeIDField, h.NodeID.String()).
				Str("client_id", client.ID).
				Int("online", len(h.clients)).
				Msg("ws client registered")

		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				h.log.Info().
					Str(applogger.NodeIDField, h.NodeID.String()).
					Str("client_id", client.ID).
					Int("online", len(h.clients)).
					Msg("ws client unregistered")
			}

			if len(h.clients) == 0 && idleTimer == nil {
				idleTimer = time.NewTimer(hubIdleTTL)
				idleTimerC = idleTimer.C
				h.log.Info().
					Str(applogger.NodeIDField, h.NodeID.String()).
					Dur("idle_ttl", hubIdleTTL).
					Msg("hub became idle, scheduled release")
			}

		case msg := <-h.inbound:
			select {
			case msg.Client.Send <- mustEncode(wsMessage{Type: "not_implemented"}):
			default:
				h.log.Warn().
					Str(applogger.NodeIDField, h.NodeID.String()).
					Str("client_id", msg.Client.ID).
					Msg("ws outbound queue full, dropped placeholder response")
			}

		case userID := <-h.forceKick:
			for _, client := range h.clients {
				if client.UserID != userID {
					continue
				}

				select {
				case client.Send <- mustEncode(wsMessage{Type: "force_logout"}):
				default:
				}

				delete(h.clients, client.ID)
				close(client.Send)
				_ = client.Conn.Close(websocket.StatusPolicyViolation, "force_logout")
			}

		case <-idleTimerC:
			if len(h.clients) == 0 {
				h.log.Info().
					Str(applogger.NodeIDField, h.NodeID.String()).
					Msg("releasing idle hub")
				h.release(h.NodeID)
				return
			}

			idleTimer = nil
			idleTimerC = nil

		case <-h.shutdown:
			return

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) DisconnectUser(userID uuid.UUID) {
	select {
	case h.forceKick <- userID:
	case <-h.done:
	}
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func mustEncode(msg wsMessage) []byte {
	payload, err := json.Marshal(msg)
	if err == nil {
		return payload
	}
	return []byte(`{"type":"internal_error"}`)
}
