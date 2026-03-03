from flask import Flask, render_template, request, send_file, session, jsonify, after_this_request
import os, logic

app = Flask(__name__)
app.secret_key = os.urandom(64) 
UPLOAD_FOLDER = 'uploads'
os.makedirs(UPLOAD_FOLDER, exist_ok=True)

@app.route('/')
def index():
    return render_template('index.html', 
                           blacklisted=logic.is_blacklisted(), 
                           my_uuid=logic.get_system_uuid())

@app.route('/encrypt', methods=['POST'])
def handle_encrypt():
    file = request.files.get('file')
    uuid = request.form.get('uuid')
    password = request.form.get('password')
    if file and uuid and password:
        path = os.path.join(UPLOAD_FOLDER, file.filename)
        file.save(path)
        enc_path = logic.encrypt_file(path, password, uuid)
        @after_this_request
        def cleanup(response):
            if os.path.exists(path): os.remove(path)
            return response
        return send_file(enc_path, as_attachment=True, download_name="package.enc")
    return jsonify({"error": "INCOMPLETE_DATA"}), 400

@app.route('/decrypt', methods=['POST'])
def handle_decrypt():
    if logic.is_blacklisted(): return jsonify({"error": "BLACKLISTED"}), 403
    
    file = request.files.get('file')
    password = request.form.get('password')
    
    # SILENT HW SCAN: User doesn't type this; the system grabs it.
    current_hw_id = logic.get_system_uuid() 
    
    if file and password:
        path = os.path.join(UPLOAD_FOLDER, file.filename)
        file.save(path)
        
        dec_path, success = logic.decrypt_file(path, password, current_hw_id)
        
        if success:
            session['attempts'] = 0
            return send_file(dec_path, as_attachment=True, download_name="decrypted.zip")
        else:
            attempts = session.get('attempts', 0) + 1
            session['attempts'] = attempts
            if attempts >= 3: logic.blacklist_machine()
            return jsonify({"error": "PULSE_MISMATCH", "attempts": attempts}), 401
            
    return jsonify({"error": "INCOMPLETE_DATA"}), 400

if __name__ == '__main__':
    app.run(port=5000, debug=False, threaded=True)
