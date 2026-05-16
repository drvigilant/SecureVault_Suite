# SecureVault Suite

> Peer-to-peer encrypted file transfer. No cloud storage. No logs. No trace.

🔴 **Live Demo:** https://securevaultsuite-production.up.railway.app

---

## What it does

Two nodes establish a direct WebSocket session. The sender encrypts the file with AES-256-GCM and a passphrase before it leaves their machine. The receiver decrypts it with the same passphrase. All temporary files are securely shredded after transfer. Nothing is stored permanently.

---

## How to use

1. Both users open the app and share their Node ID
2. Enter your partner's Node ID and click **Sync Session**
3. One user clicks **Sender**, the other is automatically assigned **Receiver**
4. Sender uploads a file with a passphrase
5. Receiver enters the same passphrase to decrypt and download

---

## Quick Start (Docker)

```bash
git clone https://github.com/drvigilant/SecureVault_Suite.git
cd SecureVault_Suite
cp .env.example .env
```

Fill in `.env` with generated secrets:

```bash
python3 -c "import secrets; print('FLASK_SECRET_KEY=' + secrets.token_hex(32))"
python3 -c "import secrets; print('VAULT_SECRET_PASS=' + secrets.token_hex(32))"
```

Run:

```bash
docker compose up -d --build
```

Open `http://localhost:5000`

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 |
| WebSockets | gorilla/websocket |
| Encryption | AES-256-GCM (Go stdlib) |
| Container | Docker multi-stage build |
| Deployment | Railway |
| Image size | ~20MB |

---

## Security

- AES-256-GCM with unique nonce per 64KB chunk
- Wrong password or tampered file rejected by auth tag
- Files shredded with 3-pass overwrite after transfer
- Secrets loaded from environment — never hardcoded
- Non-root user inside Docker container
- HTTPS enforced in production

---

## Limitations

- Requires both users online simultaneously
- No user authentication — session based on Node ID exchange
- Password must be shared out-of-band
- Not yet quantum-safe (ML-KEM/ML-DSA planned)

---

*Created by [@drvigilant](https://github.com/drvigilant/SecureVault_Suite)*
