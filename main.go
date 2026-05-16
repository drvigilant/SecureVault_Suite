package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const uploadFolder = "uploads"
const maxUploadSize = 500 << 20 // 500MB

var hub = newHub()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	godotenv.Load()

	if os.Getenv("FLASK_SECRET_KEY") == "" {
		log.Fatal("[ERROR] FLASK_SECRET_KEY not set")
	}

	os.MkdirAll(uploadFolder, 0700)

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler).Methods("GET")
	r.HandleFunc("/ws", wsHandler)
	r.HandleFunc("/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/download", downloadHandler).Methods("POST")

	fmt.Println("[SYSTEM] SecureVault Go Engine running on :5000")
	log.Fatal(http.ListenAndServe(":5000", r))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS ERROR]", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	go func() {
		defer conn.Close()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	defer func() {
		hub.unregister(client)
		close(client.send)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		hub.handleMessage(client, msg)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonError(w, "File exceeds 500MB limit", 413)
		return
	}

	room := r.FormValue("room")
	file, header, err := r.FormFile("file")
	if err != nil || room == "" {
		jsonError(w, "Missing data", 400)
		return
	}
	defer file.Close()

	hub.mu.RLock()
	sessionKey := hub.roomKeys[room]
	hub.mu.RUnlock()

	if sessionKey == nil {
		jsonError(w, "ML-KEM handshake not complete", 403)
		return
	}

	tmpPath := filepath.Join(uploadFolder, filepath.Base(header.Filename))
	tmp, err := os.Create(tmpPath)
	if err != nil {
		jsonError(w, "Server error", 500)
		return
	}
	io.Copy(tmp, file)
	tmp.Close()

	encPath := filepath.Join(uploadFolder, room+".enc")
	if err := encryptFile(tmpPath, encPath, sessionKey); err != nil {
		jsonError(w, "Encryption failed", 500)
		return
	}
	secureShred(tmpPath, 3)

	nameFile := filepath.Join(uploadFolder, room+".name")
	os.WriteFile(nameFile, []byte(filepath.Base(header.Filename)), 0600)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1 << 20)
	room := r.FormValue("room")

	encPath := filepath.Join(uploadFolder, room+".enc")
	if _, err := os.Stat(encPath); os.IsNotExist(err) {
		jsonError(w, "Package not found", 404)
		return
	}

	hub.mu.RLock()
	sessionKey := hub.roomKeys[room]
	hub.mu.RUnlock()

	if sessionKey == nil {
		jsonError(w, "ML-KEM handshake not complete", 403)
		return
	}

	decPath := filepath.Join(uploadFolder, room+"_unlocked.bin")
	if err := decryptFile(encPath, decPath, sessionKey); err != nil {
		jsonError(w, "Decryption failed", 401)
		return
	}

	defer func() {
		secureShred(encPath, 3)
		secureShred(decPath, 3)
	}()

	originalName := "SecureVault_Payload.bin"
	nameFile := filepath.Join(uploadFolder, room+".name")
	if b, err := os.ReadFile(nameFile); err == nil {
		originalName = string(b)
		os.Remove(nameFile)
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+originalName+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, decPath)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
