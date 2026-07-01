package service

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/olympics-api/internal/model"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 30 * time.Second
	wsSendBuffer = 32
)

// Broadcaster lets the domain services fan out real-time updates without
// knowing about the websocket transport.
type Broadcaster interface {
	Broadcast(tournamentID uuid.UUID, msg *model.WSMessage)
}

type WebSocketService interface {
	Broadcaster
	// Register attaches a connection to a tournament's room and blocks until it
	// disconnects, so the caller (the HTTP handler) holds the request open.
	Register(tournamentID uuid.UUID, conn *websocket.Conn)
}

type wsClient struct {
	send chan []byte
}

// websocketService is an in-process hub. With a single replica there is no need
// for NATS; every connection is held in memory and broadcasts go out directly.
type websocketService struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[*wsClient]bool
}

func NewWebSocketService() WebSocketService {
	return &websocketService{
		rooms: make(map[uuid.UUID]map[*wsClient]bool),
	}
}

func (s *websocketService) Broadcast(tournamentID uuid.UUID, msg *model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal ws message", "error", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.rooms[tournamentID] {
		// drop the update for any client that cannot keep up rather than block
		select {
		case client.send <- data:
		default:
		}
	}
}

func (s *websocketService) Register(tournamentID uuid.UUID, conn *websocket.Conn) {
	client := &wsClient{send: make(chan []byte, wsSendBuffer)}
	s.add(tournamentID, client)
	defer s.remove(tournamentID, client)

	done := make(chan struct{})
	go s.writePump(conn, client, done)

	// read pump: we do not expect inbound messages, but reading detects closes
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	close(done)
	_ = conn.Close()
}

func (s *websocketService) writePump(conn *websocket.Conn, client *wsClient, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-client.send:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (s *websocketService) add(tournamentID uuid.UUID, client *wsClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rooms[tournamentID] == nil {
		s.rooms[tournamentID] = make(map[*wsClient]bool)
	}
	s.rooms[tournamentID][client] = true
}

func (s *websocketService) remove(tournamentID uuid.UUID, client *wsClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if room := s.rooms[tournamentID]; room != nil {
		if _, ok := room[client]; ok {
			delete(room, client)
			close(client.send)
		}
		if len(room) == 0 {
			delete(s.rooms, tournamentID)
		}
	}
}
