import hashlib, base64, os, subprocess, zipfile, io, platform, secrets
from pathlib import Path
from cryptography.fernet import Fernet, InvalidToken

SECRET_PASS = "PQ-SHIELD-2026-ULTRA-MAX-PULSE"

def secure_shred(path, passes=3):
    """Forensic-grade file destruction."""
    if not os.path.exists(path): return
    file_size = os.path.getsize(path)
    with open(path, "ba+", buffering=0) as f:
        for _ in range(passes):
            f.seek(0)
            f.write(secrets.token_bytes(file_size))
            f.flush()
            os.fsync(f.fileno())
        f.seek(0)
        f.write(b"\x00" * file_size)
        f.flush()
        os.fsync(f.fileno())
    os.remove(path)

def get_system_uuid():
    try:
        if platform.system() == "Windows":
            cmd = 'wmic csproduct get uuid'
            return subprocess.check_output(cmd, shell=True).decode().split('\n')[1].strip()
        return subprocess.check_output('cat /var/lib/dbus/machine-id', shell=True).decode().strip()
    except:
        return "STATIC-HARDWARE-ID"

def get_session_room(id1, id2):
    """Creates a unified commutative room regardless of who initiated."""
    ids = sorted([id1.strip(), id2.strip()])
    combined = f"{ids[0]}-SECURE-{ids[1]}"
    return hashlib.sha256(combined.encode()).hexdigest()[:12].upper()

def generate_quantum_key(user_password, session_room):
    """Key is bound to the LIVE Session Room + Password"""
    seed = (SECRET_PASS + session_room + user_password).encode()
    return base64.urlsafe_b64encode(hashlib.sha512(seed).digest()[:32])

def encrypt_file(file_path, user_password, session_room):
    zip_buffer = io.BytesIO()
    with zipfile.ZipFile(zip_buffer, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.write(file_path, arcname=os.path.basename(file_path))
    
    key = generate_quantum_key(user_password, session_room)
    encrypted_data = Fernet(key).encrypt(zip_buffer.getvalue())
    
    enc_path = os.path.join(os.path.dirname(file_path), f"{session_room}.enc")
    with open(enc_path, "wb") as f:
        f.write(encrypted_data)
    return enc_path

def decrypt_file(enc_path, user_password, session_room):
    key = generate_quantum_key(user_password, session_room)
    with open(enc_path, "rb") as f:
        encrypted_data = f.read()
    try:
        decrypted_data = Fernet(key).decrypt(encrypted_data)
        out_path = enc_path.replace(".enc", "_unlocked.zip")
        with open(out_path, "wb") as f:
            f.write(decrypted_data)
        return out_path, True
    except InvalidToken:
        return None, False
