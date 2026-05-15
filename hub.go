package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	nodeID string
	send   chan []byte
}

type Hub struct {
	mu          sync.RWMutex
	nodeClients map[string]*Client  // nodeID → client
	roomMembers map[string][]*Client // room → clients
}

func newHub() *Hub {
	return &Hub{
		nodeClients: make(map[string]*Client),
		roomMembers: make(map[string][]*Client),
	}
}

func getSessionRoom(id1, id2 string) string {
	ids := []string{id1, id2}
	sort.Strings(ids)
	combined := fmt.Sprintf("%s-SECURE-%s", ids[0], ids[1])
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%X", hash[:6]) // 12 char hex
}

func (h *Hub) register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nodeClients[client.nodeID] = client
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.nodeClients, client.nodeID)
	for room, members := range h.roomMembers {
		updated := []*Client{}
		for _, m := range members {
			if m != client {
				updated = append(updated, m)
			}
		}
		if len(updated) == 0 {
			delete(h.roomMembers, room)
		} else {
			h.roomMembers[room] = updated
		}
	}
}

func (h *Hub) joinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.roomMembers[room] {
		if m == client {
			return // already in room
		}
	}
	h.roomMembers[room] = append(h.roomMembers[room], client)
}

func (h *Hub) emitToClient(client *Client, event string, data map[string]interface{}) {
	msg := map[string]interface{}{"event": event, "data": data}
	b, _ := json.Marshal(msg)
	client.send <- b
}

func (h *Hub) emitToRoom(room, event string, data map[string]interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.roomMembers[room] {
		h.emitToClient(c, event, data)
	}
}

func (h *Hub) handleMessage(client *Client, raw []byte) {
	var msg struct {
		Event string                 `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	data := msg.Data

	switch msg.Event {

	case "register_node":
		nodeID, _ := data["my_id"].(string)
		client.nodeID = nodeID
		h.register(client)

	case "join_session":
		myID, _ := data["my_id"].(string)
		partnerID, _ := data["partner_id"].(string)
		room := getSessionRoom(myID, partnerID)
		h.joinRoom(client, room)
		h.emitToRoom(room, "session_established", map[string]interface{}{
			"room": room,
		})

	case "page_partner":
		myID, _ := data["my_id"].(string)
		partnerID, _ := data["partner_id"].(string)
		room := getSessionRoom(myID, partnerID)
		h.mu.RLock()
		partner, ok := h.nodeClients[partnerID]
		h.mu.RUnlock()
		if ok {
			h.emitToClient(partner, "incoming_connection", map[string]interface{}{
				"room":       room,
				"partner_id": myID,
			})
		}

	case "ready_in_room":
		room, _ := data["room"].(string)
		h.emitToRoom(room, "partner_ready", map[string]interface{}{})

	case "select_role":
		room, _ := data["room"].(string)
		h.mu.RLock()
		members := h.roomMembers[room]
		h.mu.RUnlock()
		for _, m := range members {
			if m == client {
				h.emitToClient(m, "role_assigned", map[string]interface{}{"role": "sender"})
			} else {
				h.emitToClient(m, "role_assigned", map[string]interface{}{"role": "receiver"})
			}
		}

	case "file_sealed":
		room, _ := data["room"].(string)
		h.emitToRoom(room, "package_ready", map[string]interface{}{
			"message": "Encrypted package waiting...",
		})
	}
}
