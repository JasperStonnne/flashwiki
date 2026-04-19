package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	applogger "fpgwiki/backend/internal/logger"
)

type HubManager struct {
	mu   sync.Mutex
	hubs map[uuid.UUID]*Hub
	log  zerolog.Logger
}

func NewHubManager(log zerolog.Logger) *HubManager {
	return &HubManager{
		hubs: make(map[uuid.UUID]*Hub),
		log:  log,
	}
}

func (m *HubManager) GetOrCreate(nodeID uuid.UUID) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[nodeID]; ok {
		select {
		case <-hub.done:
			delete(m.hubs, nodeID)
		default:
			return hub
		}
	}

	hub := NewHub(nodeID, m.log, m.Release)
	m.hubs[nodeID] = hub

	go hub.Run(context.Background())

	m.log.Info().
		Str(applogger.NodeIDField, nodeID.String()).
		Msg("created ws hub")

	return hub
}

func (m *HubManager) Release(nodeID uuid.UUID) {
	m.mu.Lock()
	hub, ok := m.hubs[nodeID]
	if ok {
		delete(m.hubs, nodeID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	close(hub.shutdown)
	m.log.Info().
		Str(applogger.NodeIDField, nodeID.String()).
		Msg("released ws hub")
}
