@echo off
title SecureVault Suite - Engine Bootstrapper
color 0A

echo ===================================================
echo    SECUREVAULT INITIALIZATION SEQUENCE
echo    Created by @drvigilant
echo    https://github.com/drvigilant/SecureVault_Suite.git
echo ===================================================
echo [SYS] Verifying Core Dependencies...

:: Check for Python
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [CRITICAL ERROR] Python is not installed or not in PATH.
    echo Please install Python 3.10+ from python.org to proceed.
    pause
    exit
)

:: Create secure sandbox if it doesn't exist
if not exist ".secure_env" (
    echo [SYS] Generating Isolated Quantum Sandbox (First run only)...
    python -m venv .secure_env
)

:: Activate and silently install
call .secure_env\Scripts\activate
echo [SYS] Synchronizing Cryptographic & Network Modules...
python -m pip install --upgrade pip -q
pip install flask flask-socketio gevent gevent-websocket cryptography -q

echo [SYS] Boot Sequence Complete.
echo [SYS] Launching Secure Interface...

:: Wait 2 seconds, then automatically open the user's default web browser
start "" http://localhost:5000

:: Start the server engine
python app.py

pause
