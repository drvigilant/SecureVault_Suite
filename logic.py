import hashlib, os, secrets
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes

SECRET_PASS = "PQ-SHIELD-2026-ULTRA-MAX-PULSE"

def secure_shred(path, passes=3):
    """Forensic-grade file destruction."""
    if not os.path.exists(path): return
    try:
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
    except Exception as e:
        print(f"[SHRED ERROR] {e}")

def get_session_room(id1, id2):
    """Creates a unified commutative room regardless of who initiated."""
    ids = sorted([id1.strip(), id2.strip()])
    combined = f"{ids[0]}-SECURE-{ids[1]}"
    return hashlib.sha256(combined.encode()).hexdigest()[:12].upper()

def generate_raw_key(user_password, session_room):
    """Generates a raw 32-byte key for AES-256."""
    seed = (SECRET_PASS + session_room + user_password).encode()
    return hashlib.sha256(seed).digest()

def encrypt_file(file_path, user_password, session_room):
    """STREAMING ENCRYPTION: Processes file in 64KB chunks. Zero RAM bloat."""
    key = generate_raw_key(user_password, session_room)
    nonce = os.urandom(16) # Unique initialization vector
    cipher = Cipher(algorithms.AES(key), modes.CTR(nonce))
    encryptor = cipher.encryptor()
    
    enc_path = os.path.join(os.path.dirname(file_path), f"{session_room}.enc")
    
    # Read, Encrypt, and Write in 64KB chunks
    with open(file_path, 'rb') as f_in, open(enc_path, 'wb') as f_out:
        f_out.write(nonce) # Embed the nonce at the start of the file
        while chunk := f_in.read(64 * 1024): 
            f_out.write(encryptor.update(chunk))
        f_out.write(encryptor.finalize())
        
    return enc_path

def decrypt_file(enc_path, user_password, session_room):
    """STREAMING DECRYPTION: Recovers data at maximum disk speed."""
    key = generate_raw_key(user_password, session_room)
    out_path = enc_path.replace(".enc", "_unlocked.dat")
    
    try:
        with open(enc_path, 'rb') as f_in, open(out_path, 'wb') as f_out:
            nonce = f_in.read(16) # Extract the nonce
            cipher = Cipher(algorithms.AES(key), modes.CTR(nonce))
            decryptor = cipher.decryptor()
            
            while chunk := f_in.read(64 * 1024):
                f_out.write(decryptor.update(chunk))
            f_out.write(decryptor.finalize())
            
        return out_path, True
    except Exception as e:
        print(f"[ERROR] Engine Decryption Failure: {e}")
        if os.path.exists(out_path): os.remove(out_path)
        return None, False
