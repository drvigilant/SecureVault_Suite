import os, shutil

def full_reset():
    print("[RESET] Initializing SecureVault System Wipe...")
    
    target_dirs = ['uploads', 'vault_data', '__pycache__']
    for folder in target_dirs:
        if os.path.exists(folder):
            try:
                shutil.rmtree(folder)
                os.makedirs(folder)
                print(f"[SUCCESS] Purged and reset: {folder}/")
            except Exception as e:
                print(f"[ERROR] Failed to clear {folder}: {e}")

    for file in os.listdir('.'):
        if file.endswith(".enc") or file.endswith("_unlocked.dat") or file.endswith(".zip"):
            try:
                os.remove(file)
                print(f"[SUCCESS] Removed loose artifact: {file}")
            except:
                pass
    print("[COMPLETE] System is now in a clean state.")

if __name__ == "__main__":
    confirm = input("Wipe all buffers and temporary locks? (y/n): ")
    if confirm.lower() == 'y':
        full_reset()
