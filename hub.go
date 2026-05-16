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
	conn       *websocket.Conn
	nodeID     string
	send       chan []byte
	kemSession *KEMSession
	sessionKey []byte // derived AES key after ML-KEM handshake
}

type Hub struct {
	mu           sync.RWMutex
	nodeClients  map[string]*Client
	roomMembers  map[string][]*Client
	roomKeys     map[string][]byte // room → shared session key
}

func newHub() *Hub {
	return &Hub{
		nodeClients: make(map[string]*Client),
		roomMembers: make(map[string][]*Client),
		roomKeys: make(map[string][]byte),
	}
}

func getSessionRoom(id1, id2 string) string {
	ids := []string{id1, id2}
	sort.Strings(ids)
	combined := fmt.Sprintf("%s-SECURE-%s", ids[0], ids[1])
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%X", hash[:6])
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
	if client.kemSession != nil {
		client.kemSession.clean()
	}
}

func (h *Hub) joinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.roomMembers[room] {
		if m == client {
			return
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

		// Generate ML-KEM keypair for this client
		kem, err := newKEMSession()
		if err != nil {
			fmt.Println("[ML-KEM ERROR] keygen failed:", err)
			return
		}
		client.kemSession = kem

		// Send our public key to the room
		h.emitToRoom(room, "kem_public_key", map[string]interface{}{
			"room":       room,
			"public_key": sharedSecretToHex(kem.publicKey),
			"node_id":    client.nodeID,
		})

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

	case "kem_encapsulate":
		// Receiver gets sender's public key, encapsulates a secret
		room, _ := data["room"].(string)
		peerPubKeyHex, _ := data["public_key"].(string)

		peerPubKey, err := hexToBytes(peerPubKeyHex)
		if err != nil {
			fmt.Println("[ML-KEM ERROR] bad public key:", err)
			return
		}

		ct, ss, err := client.kemSession.encapsulate(peerPubKey)
		if err != nil {
			fmt.Println("[ML-KEM ERROR] encapsulation failed:", err)
			return
		}

		// Derive session key
		client.sessionKey = deriveSessionKey(ss, room)
		h.mu.Lock()
		h.roomKeys[room] = client.sessionKey
		h.mu.Unlock()
		fmt.Printf("[ML-KEM] Room %s session key stored\n", room)

		// Send ciphertext to sender so they can decapsulate
		h.mu.RLock()
		members := h.roomMembers[room]
		h.mu.RUnlock()
		for _, m := range members {
			if m != client {
				h.emitToClient(m, "kem_ciphertext", map[string]interface{}{
					"room":       room,
					"ciphertext": sharedSecretToHex(ct),
				})
			}
		}

	case "kem_decapsulate":
		// Sender gets ciphertext, decapsulates to get same shared secret
		room, _ := data["room"].(string)
		ctHex, _ := data["ciphertext"].(string)

		ct, err := hexToBytes(ctHex)
		if err != nil {
			fmt.Println("[ML-KEM ERROR] bad ciphertext:", err)
			return
		}

		ss, err := client.kemSession.decapsulate(ct)
		if err != nil {
			fmt.Println("[ML-KEM ERROR] decapsulation failed:", err)
			return
		}

		client.sessionKey = deriveSessionKey(ss, room)
		fmt.Printf("[ML-KEM] Sender session key derived for room %s\n", room)

		// Both sides now have the same key — signal handshake complete
		h.emitToRoom(room, "kem_handshake_complete", map[string]interface{}{
			"room": room,
		})

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
