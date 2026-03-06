from gevent import monkey
monkey.patch_all() # High-performance concurrency

from flask import Flask, render_template, request, send_file, jsonify, after_this_request
from flask_socketio import SocketIO, emit, join_room
import os, logic

app = Flask(__name__)
app.config['SECRET_KEY'] = 'DRVIGILANT_SECURE_VAULT_2026'

socketio = SocketIO(app, async_mode='gevent', cors_allowed_origins="*")

UPLOAD_FOLDER = 'uploads'
os.makedirs(UPLOAD_FOLDER, exist_ok=True)

@app.route('/')
def index():
    return render_template('index.html')

# --- SOCKET.IO PAGING SYSTEM ---
@socketio.on('register_node')
def handle_register(data):
    node_id = data['my_id']
    join_room(node_id)
    print(f"[NETWORK] Node Active: {node_id}")

@socketio.on('join_session')
def handle_join(data):
    my_id = data['my_id']
    partner_id = data['partner_id']
    room = logic.get_session_room(my_id, partner_id)
    join_room(room)
    emit('session_established', {'room': room}, room=room)

@socketio.on('page_partner')
def handle_page(data):
    my_id = data['my_id']
    partner_id = data['partner_id']
    room = logic.get_session_room(my_id, partner_id)
    emit('incoming_connection', {'room': room, 'partner_id': my_id}, room=partner_id)

@socketio.on('select_role')
def handle_role(data):
    room = data['room']
    sender_id = data['my_id']
    emit('role_assigned', {'sender_id': sender_id}, room=room)

@socketio.on('file_sealed')
def handle_file_sealed(data):
    room = data['room']
    emit('package_ready', {'message': 'Encrypted package waiting...'}, room=room)

# --- PAYLOAD TRANSMISSION ---
@app.route('/upload', methods=['POST'])
def upload():
    file = request.files.get('file')
    password = request.form.get('password')
    room = request.form.get('room')
    
    if file and password and room:
        path = os.path.join(UPLOAD_FOLDER, file.filename)
        file.save(path)
        logic.encrypt_file(path, password, room)
        logic.secure_shred(path) # Shred raw file instantly
        return jsonify({"status": "success"})
    return jsonify({"error": "Missing data"}), 400

@app.route('/download', methods=['POST'])
def download():
    password = request.form.get('password')
    room = request.form.get('room')
    enc_path = os.path.join(UPLOAD_FOLDER, f"{room}.enc")
    
    if os.path.exists(enc_path) and password:
        dec_path, success = logic.decrypt_file(enc_path, password, room)
        if success:
            @after_this_request
            def cleanup(response):
                logic.secure_shred(enc_path)
                logic.secure_shred(dec_path)
                return response
            # Sending as a .zip assuming the user is uploading zipped archives
            return send_file(dec_path, as_attachment=True, download_name="SecureVault_Payload.zip")
        return jsonify({"error": "Invalid Password or Pulse"}), 401
    return jsonify({"error": "Package not found"}), 404

if __name__ == '__main__':
    print("[SYSTEM] Booting High-Performance Gevent Engine...")
    socketio.run(app, host='0.0.0.0', port=5000, debug=False)
