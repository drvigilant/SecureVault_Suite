import hashlib, os, secrets
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from dotenv import load_dotenv

load_dotenv()

SECRET_PASS = os.environ.get('VAULT_SECRET_PASS')
if not SECRET_PASS:
    raise RuntimeError("VAULT_SECRET_PASS is not set. Copy .env.example to .env and fill it in.")

def secure_shred(path, passes=3):
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
    ids = sorted([id1.strip(), id2.strip()])
    combined = f"{ids[0]}-SECURE-{ids[1]}"
    return hashlib.sha256(combined.encode()).hexdigest()[:12].upper()

def generate_key(user_password, session_room):
    seed = (SECRET_PASS + session_room + user_password).encode()
    return hashlib.sha256(seed).digest()

def encrypt_file(file_path, user_password, session_room):
    key = generate_key(user_password, session_room)
    aesgcm = AESGCM(key)
    enc_path = os.path.join(os.path.dirname(file_path), f"{session_room}.enc")

    with open(file_path, 'rb') as f_in, open(enc_path, 'wb') as f_out:
        while True:
            chunk = f_in.read(64 * 1024)
            if not chunk:
                break
            nonce = os.urandom(12)
            ct = aesgcm.encrypt(nonce, chunk, None)
            f_out.write(nonce)
            f_out.write(len(ct).to_bytes(4, 'big'))
            f_out.write(ct)

    return enc_path

def decrypt_file(enc_path, user_password, session_room):
    key = generate_key(user_password, session_room)
    aesgcm = AESGCM(key)
    out_path = enc_path.replace(".enc", "_unlocked.bin")

    try:
        with open(enc_path, 'rb') as f_in, open(out_path, 'wb') as f_out:
            while True:
                nonce = f_in.read(12)
                if not nonce:
                    break
                size_bytes = f_in.read(4)
                chunk_size = int.from_bytes(size_bytes, 'big')
                ct = f_in.read(chunk_size)
                plaintext = aesgcm.decrypt(nonce, ct, None)
                f_out.write(plaintext)
        return out_path, True
    except Exception as e:
        print(f"[ERROR] Decryption failure: {e}")
        if os.path.exists(out_path):
            os.remove(out_path)
        return None, False
