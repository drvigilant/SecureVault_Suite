#!/bin/bash
# ==========================================
#   SECUREVAULT INITIALIZATION SEQUENCE
# ==========================================

if [ "$EUID" -eq 0 ]; then
    echo "[ERROR] Do not run this script with sudo or as root."
    echo "        Run as a normal user: bash Start_Engine.sh"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [ -d ".secure_env" ] && [ ! -w ".secure_env" ]; then
    echo "[WARN] .secure_env is root-owned. Attempting fix..."
    sudo chown -R "$(whoami)":"$(whoami)" .secure_env uploads 2>/dev/null \
        && echo "[OK] Ownership fixed." \
        || { echo "[ERROR] Run: sudo chown -R \$(whoami) ."; exit 1; }
fi

if [ ! -d ".secure_env" ]; then
    echo "[SYS] Generating isolated sandbox..."
    python3 -m venv .secure_env
fi

source .secure_env/bin/activate

echo "[SYS] Synchronizing modules..."
pip install --upgrade pip -q
pip install flask flask-socketio gevent gevent-websocket cryptography python-dotenv -q

mkdir -p uploads
chmod 700 uploads

if [ ! -f ".env" ]; then
    echo "[SYS] No .env found. Generating secure defaults..."
    SECRET_KEY=$(python3 -c "import secrets; print(secrets.token_hex(32))")
    SECRET_PASS=$(python3 -c "import secrets; print(secrets.token_hex(32))")
    cat > .env << ENVEOF
# SecureVault Runtime Secrets — do not commit this file
FLASK_SECRET_KEY=${SECRET_KEY}
VAULT_SECRET_PASS=${SECRET_PASS}
FLASK_ENV=production
ENVEOF
    chmod 600 .env
    echo "[OK] .env created with random secrets."
fi

echo "[SYS] Boot sequence complete. Engine live."
python3 app.py
