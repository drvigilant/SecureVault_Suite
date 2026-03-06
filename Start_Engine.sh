#!/bin/bash
echo "========================================="
echo "   SECUREVAULT INITIALIZATION SEQUENCE   "
echo "========================================="

# 1. Create the virtual environment if it doesn't exist
if [ ! -d ".secure_env" ]; then
    echo "[SYS] Generating Isolated Sandbox..."
    python3 -m venv .secure_env
fi

# 2. Activate the environment
source .secure_env/bin/activate

# 3. Install dependencies quietly
echo "[SYS] Synchronizing Modules..."
pip install --upgrade pip -q
pip install flask flask-socketio gevent gevent-websocket cryptography -q

# 4. Launch the engine
echo "[SYS] Boot Sequence Complete. Engine Live."
python3 app.py
