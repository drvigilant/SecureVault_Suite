import hashlib, base64, os, subprocess, zipfile, io, platform
from pathlib import Path
from cryptography.fernet import Fernet, InvalidToken

SECRET_PASS = "PQ-SHIELD-2026-ULTRA-MAX-PULSE"

def get_lock_path():
    if platform.system() == "Windows":
        return Path(os.environ.get('APPDATA', '.')) / ".system_lock"
    return Path.home() / ".smt_system_lock"

LOCK_FILE = get_lock_path()

def get_environmental_pulse():
    """Gathers machine entropy to bind the key to this specific hardware."""
    try:
        os_info = platform.platform()
        node_name = platform.node()
        if platform.system() == "Windows":
            # Volume serial is highly unique to the OS drive
            vol = subprocess.check_output("vol C:", shell=True).decode().strip()
            return f"{os_info}-{node_name}-{vol}"
        return f"{os_info}-{node_name}"
    except:
        return "STABLE_ENV_PULSE"

def get_system_uuid():
    """Retrieves the unique Hardware UUID."""
    try:
        if platform.system() == "Windows":
            cmd = 'wmic csproduct get uuid'
            output = subprocess.check_output(cmd, shell=True).decode()
            return output.split('\n')[1].strip()
        return subprocess.check_output('cat /var/lib/dbus/machine-id', shell=True).decode().strip()
    except:
        return "STATIC-HARDWARE-ID"

def generate_quantum_key(user_password, target_uuid):
    """Derives a 512-bit seed into a 32-byte Fernet key."""
    pulse = get_environmental_pulse()
    seed = (SECRET_PASS + target_uuid + user_password + pulse).encode()
    return base64.urlsafe_b64encode(hashlib.sha512(seed).digest()[:32])

def encrypt_file(file_path, user_password, target_uuid):
    zip_buffer = io.BytesIO()
    with zipfile.ZipFile(zip_buffer, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.write(file_path, arcname=os.path.basename(file_path))
    key = generate_quantum_key(user_password, target_uuid)
    encrypted_data = Fernet(key).encrypt(zip_buffer.getvalue())
    enc_path = os.path.join(os.path.dirname(file_path), "package.enc")
    with open(enc_path, "wb") as f:
        f.write(encrypted_data)
    return enc_path

def decrypt_file(file_path, user_password, current_uuid):
    """Attempts decryption using the current machine's HW ID."""
    key = generate_quantum_key(user_password, current_uuid)
    with open(file_path, "rb") as f:
        encrypted_data = f.read()
    try:
        decrypted_data = Fernet(key).decrypt(encrypted_data)
        out_path = file_path.replace(".enc", "_decrypted.zip")
        with open(out_path, "wb") as f:
            f.write(decrypted_data)
        return out_path, True
    except InvalidToken:
        return None, False

def is_blacklisted():
    return LOCK_FILE.exists()

def blacklist_machine():
    with open(LOCK_FILE, "w") as f:
        f.write("PULSE_REJECTION_LOCKED")
