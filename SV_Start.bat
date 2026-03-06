@echo off
title SecureVault Suite - Engine Bootstrapper
color 0A

echo ===================================================
echo    SECUREVAULT INITIALIZATION SEQUENCE
echo    Created by @drvigilant
echo    https://github.com/drvigilant/SecureVault_Suite.git
echo ===================================================
echo [SYS] Verifying Core Dependencies...

python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [CRITICAL ERROR] Python is not installed or not in PATH.
    echo Please install Python 3.10+ from python.org to proceed.
    pause
    exit
)

if not exist ".secure_env" (
    echo [SYS] Generating Isolated Quantum Sandbox (First run only)...
    python -m venv .secure_env
)

call .secure_env\Scripts\activate
echo [SYS] Synchronizing Cryptographic Modules...
python -m pip install --upgrade pip -q
pip install flask flask-socketio gevent gevent-websocket cryptography -q

echo [SYS] Boot Sequence Complete. Launching...
start "" http://localhost:5000
python app.py
pause
