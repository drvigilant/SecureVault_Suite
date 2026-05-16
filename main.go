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

	secretKey := os.Getenv("FLASK_SECRET_KEY")
	if secretKey == "" {
		log.Fatal("[ERROR] FLASK_SECRET_KEY not set")
	}

	vaultPass := os.Getenv("VAULT_SECRET_PASS")
	if vaultPass == "" {
		log.Fatal("[ERROR] VAULT_SECRET_PASS not set")
	}

	os.MkdirAll(uploadFolder, 0700)

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler).Methods("GET")
	r.HandleFunc("/ws", wsHandler)
	r.HandleFunc("/upload", uploadHandler(vaultPass)).Methods("POST")
	r.HandleFunc("/download", downloadHandler(vaultPass)).Methods("POST")

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

	// writer + keepalive ping goroutine
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

	// reader loop
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

func uploadHandler(vaultPass string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			jsonError(w, "File exceeds 500MB limit", 413)
			return
		}

		password := r.FormValue("password")
		room := r.FormValue("room")
		file, header, err := r.FormFile("file")
		if err != nil || password == "" || room == "" {
			jsonError(w, "Missing data", 400)
			return
		}
		defer file.Close()

		tmpPath := filepath.Join(uploadFolder, filepath.Base(header.Filename))
		tmp, err := os.Create(tmpPath)
		if err != nil {
			jsonError(w, "Server error", 500)
			return
		}
		io.Copy(tmp, file)
		tmp.Close()

		encPath := filepath.Join(uploadFolder, room+".enc")
		if err := encryptFile(tmpPath, encPath, password, room, vaultPass); err != nil {
			jsonError(w, "Encryption failed", 500)
			return
		}
		secureShred(tmpPath, 3)

		nameFile := filepath.Join(uploadFolder, room+".name")
		os.WriteFile(nameFile, []byte(filepath.Base(header.Filename)), 0600)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func downloadHandler(vaultPass string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(1 << 20)
		password := r.FormValue("password")
		room := r.FormValue("room")

		encPath := filepath.Join(uploadFolder, room+".enc")
		if _, err := os.Stat(encPath); os.IsNotExist(err) {
			jsonError(w, "Package not found", 404)
			return
		}

		decPath := filepath.Join(uploadFolder, room+"_unlocked.bin")
		if err := decryptFile(encPath, decPath, password, room, vaultPass); err != nil {
			jsonError(w, "Invalid password or pulse", 401)
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
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
