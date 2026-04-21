package ws

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"nhooyr.io/websocket"

	"fpgwiki/backend/internal/config"
	applogger "fpgwiki/backend/internal/logger"
)

const (
	accessSubprotocolPrefix = "access."
	writeWait               = 10 * time.Second
)

func Handler(cfg config.Config, hm *HubManager, log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = cfg

		nodeID, ok := parseNodeID(c)
		if !ok {
			return
		}

		subprotocol, token, ok := parseAccessSubprotocol(c.GetHeader("Sec-WebSocket-Protocol"))
		if !ok {
			writeErr(c, http.StatusUnauthorized, "unauthorized", "missing or invalid websocket credentials")
			return
		}
		_ = token

		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
			Subprotocols:       []string{subprotocol},
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Warn().
				Err(err).
				Str(applogger.NodeIDField, nodeID.String()).
				Msg("ws handshake failed")
			return
		}

		hub := hm.GetOrCreate(nodeID)
		client := &Client{
			ID:     uuid.NewString(),
			UserID: uuid.Nil,
			Level:  "readable",
			Conn:   conn,
			Send:   make(chan []byte, 8),
		}

		select {
		case hub.register <- client:
		case <-hub.done:
			hub = hm.GetOrCreate(nodeID)
			select {
			case hub.register <- client:
			case <-hub.done:
				_ = conn.Close(websocket.StatusTryAgainLater, "hub unavailable")
				return
			}
		}

		connCtx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			readPump(connCtx, hub, client, log)
		}()
		go func() {
			defer cancel()
			writePump(connCtx, hub, client, log)
		}()
	}
}

func parseNodeID(c *gin.Context) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query("doc"))
	if raw == "" {
		writeErr(c, http.StatusBadRequest, "invalid_doc", "missing doc query parameter")
		return uuid.Nil, false
	}

	nodeID, err := uuid.Parse(raw)
	if err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_doc", "doc query parameter must be uuid")
		return uuid.Nil, false
	}

	return nodeID, true
}

func parseAccessSubprotocol(header string) (subprotocol string, token string, ok bool) {
	parts := strings.Split(header, ",")
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if !strings.HasPrefix(candidate, accessSubprotocolPrefix) {
			continue
		}

		token = strings.TrimPrefix(candidate, accessSubprotocolPrefix)
		if token == "" {
			return "", "", false
		}
		return candidate, token, true
	}

	return "", "", false
}

func readPump(ctx context.Context, hub *Hub, client *Client, log zerolog.Logger) {
	defer func() {
		select {
		case hub.unregister <- client:
		case <-hub.done:
		}
	}()

	for {
		_, payload, err := client.Conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				status := websocket.CloseStatus(err)
				if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
					log.Debug().
						Err(err).
						Str(applogger.NodeIDField, hub.NodeID.String()).
						Str("client_id", client.ID).
						Msg("ws read stopped")
				}
			}
			return
		}

		select {
		case hub.inbound <- inboundMsg{Client: client, Payload: payload}:
		case <-hub.done:
			return
		}
	}
}

func writePump(ctx context.Context, hub *Hub, client *Client, log zerolog.Logger) {
	defer func() {
		_ = client.Conn.Close(websocket.StatusNormalClosure, "bye")
	}()

	for {
		select {
		case payload, ok := <-client.Send:
			if !ok {
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := client.Conn.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				log.Debug().
					Err(err).
					Str(applogger.NodeIDField, hub.NodeID.String()).
					Str("client_id", client.ID).
					Msg("ws write failed")
				return
			}

		case <-hub.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Error   *errorResponse `json:"error"`
}

func writeErr(c *gin.Context, status int, code string, message string) {
	c.JSON(status, envelope{
		Success: false,
		Data:    nil,
		Error: &errorResponse{
			Code:    code,
			Message: message,
		},
	})
}
