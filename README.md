# SecureVault Suite
Post-Quantum Encryption with Silent Environmental Pulse Authentication.

SecureVault is a file protection utility designed to bind sensitive data to specific hardware. Rather than relying solely on a passphrase, it validates the hardware fingerprint of the local machine before attempting decryption.

## Core Innovation: Silent Hardware Scan
The system eliminates the manual input of machine IDs during the decryption phase by performing a background scan of system metadata, including UUIDs, OS Volume IDs, and Node Entropy.

- Hardware-Locked: Encryption keys are derived using local hardware identifiers, making files unusable on unauthorized devices.
- Zero-Input Decryption: The end-user provides only the password; the system handles machine verification autonomously.
- Anti-Brute Force: The application blacklists the local environment after three failed authorization attempts.

## Quick Start
1. Clone the repository: 
   git clone https://github.com/drvigilant/SecureVault_Suite.git
2. Install dependencies: 
   pip install flask cryptography
3. Initialize the server: 
   python3 app.py
4. Interface: 
   Access the local vault at http://localhost:5000

Created by drvigilant. Focused on data integrity and hardware-level privacy.
