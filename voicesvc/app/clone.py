from flask import jsonify, request, Flask

CONFIG_VOICE_SAMPLE_KEY = "voice_sample_dir"


# Clone the user's voice using the input wave sample. This is a synchronous handler.
def clone_sync_handler(app: Flask):
    @app.route("/clone-sync/<user_id>", methods=["POST"])
    def clone_sync_handler(user_id: str):
        if request.content_type not in ["audio/x-wav", "audio/wav", "audio/wave"]:
            return "", 406
        req_data = request.get_data()
        print(
            f"@@@@@ req data length {len(req_data)} config {app.config[CONFIG_VOICE_SAMPLE_KEY]}"
        )
        return jsonify(
            {
                "request-method": request.method,
                "request-host": request.host,
                "request-url": request.url,
            }
        )
