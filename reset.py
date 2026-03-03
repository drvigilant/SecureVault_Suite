import os
import subprocess
from pathlib import Path

LOCK_FILE = Path(os.environ.get('APPDATA', '.')) / ".system_lock"

def premium_reset():
    print("VAULT PRO | LICENSED RECOVERY UTILITY")
    token = input("\nENTER PREMIUM RESET TOKEN: ")
    
    # Requirement #4: Validation (Token can be changed for different users)
    if token == "ADMIN-RESET-2026":
        if LOCK_FILE.exists():
            # Remove system/hidden attributes then delete
            subprocess.run(["attrib", "-H", "-S", str(LOCK_FILE)], capture_output=True)
            os.remove(LOCK_FILE)
            print("\n[SUCCESS] Machine state cleared. Access restored.")
        else:
            print("\n[INFO] Machine is already authorized.")
    else:
        print("\n[ERROR] Invalid token. Authorization denied.")

if __name__ == "__main__":
    premium_reset()
    input("\nPress Enter to exit...")
