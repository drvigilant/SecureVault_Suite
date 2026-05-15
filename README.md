# SecureVault Suite

End-to-end encrypted peer-to-peer file transfer. Files are encrypted with AES-256-GCM before leaving your machine and shredded after delivery. No cloud storage. No logs. No trace.

Built with Go + WebSockets + Docker.

---

## How it works

1. Both nodes open the app and exchange Node IDs
2. A session room is established via WebSocket
3. Sender encrypts the file with AES-256-GCM and a passphrase
4. Receiver decrypts with the same passphrase
5. All temporary files are securely shredded after transfer

---

## Quick Start (Docker)

**Requirements:** Docker + Docker Compose

```bash
git clone https://github.com/drvigilant/SecureVault_Suite.git
cd SecureVault_Suite
cp .env.example .env
```

Edit `.env` with your own secrets:

```bash
python3 -c "import secrets; print('FLASK_SECRET_KEY=' + secrets.token_hex(32))"
python3 -c "import secrets; print('VAULT_SECRET_PASS=' + secrets.token_hex(32))"
```

Then run:

```bash
docker compose up -d --build
```

Open `http://localhost:5000`

---

## Security

- AES-256-GCM encryption with unique nonce per chunk
- Wrong password or tampered file is rejected by auth tag
- Raw files are shredded (3-pass overwrite) after encryption
- Secrets loaded from `.env` — never hardcoded
- Non-root user inside Docker container
- `.env` is gitignored and never committed

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 |
| WebSockets | gorilla/websocket |
| Encryption | AES-256-GCM (Go stdlib) |
| Container | Docker (multi-stage build) |
| Image size | ~20MB |

---

## Created by [@drvigilant](https://github.com/drvigilant/SecureVault_Suite)
