from gevent import monkey
monkey.patch_all()

from flask import Flask, render_template, request, send_file, jsonify, after_this_request
from flask_socketio import SocketIO, emit, join_room
from dotenv import load_dotenv
import os, logic

load_dotenv()

app = Flask(__name__)
app.config['SECRET_KEY'] = os.environ.get('FLASK_SECRET_KEY')
if not app.config['SECRET_KEY']:
    raise RuntimeError("FLASK_SECRET_KEY is not set. Check your .env file.")

socketio = SocketIO(app, async_mode='gevent', cors_allowed_origins="*")

UPLOAD_FOLDER = 'uploads'
os.makedirs(UPLOAD_FOLDER, exist_ok=True)

room_members = {}  # {room: set of socket SIDs}

@app.route('/')
def index():
    return render_template('index.html')

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
    room_members.setdefault(room, set()).add(request.sid)
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
    sender_sid = request.sid
    members = room_members.get(room, set())
    receiver_sids = members - {sender_sid}
    emit('role_assigned', {'role': 'sender'}, to=sender_sid)
    for sid in receiver_sids:
        emit('role_assigned', {'role': 'receiver'}, to=sid)

@socketio.on('file_sealed')
def handle_file_sealed(data):
    room = data['room']
    emit('package_ready', {'message': 'Encrypted package waiting...'}, room=room)

@socketio.on('disconnect')
def handle_disconnect():
    sid = request.sid
    for room, members in list(room_members.items()):
        members.discard(sid)
        if not members:
            del room_members[room]

@app.route('/upload', methods=['POST'])
def upload():
    file = request.files.get('file')
    password = request.form.get('password')
    room = request.form.get('room')

    if not file or not password or not room:
        return jsonify({"error": "Missing data"}), 400

    # 100MB limit
    file.seek(0, 2)
    if file.tell() > 100 * 1024 * 1024:
        return jsonify({"error": "File exceeds 100MB limit"}), 413
    file.seek(0)

    filename = os.path.basename(file.filename)
    path = os.path.join(UPLOAD_FOLDER, filename)
    file.save(path)
    logic.encrypt_file(path, password, room)
    logic.secure_shred(path)
    return jsonify({"status": "success"})

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
            return send_file(dec_path, as_attachment=True, download_name="SecureVault_Payload.bin")
        return jsonify({"error": "Invalid password or pulse"}), 401
    return jsonify({"error": "Package not found"}), 404

if __name__ == '__main__':
    print("[SYSTEM] Booting High-Performance Gevent Engine...")
    socketio.run(app, host='0.0.0.0', port=5000, debug=False)
