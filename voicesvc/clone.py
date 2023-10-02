import codecs
import time
from flask import jsonify, request, Flask, Response, make_response
from svc import VoiceSvc


# Clone the user's voice using the input wave sample. This is a synchronous handler.
def clone_rt_handler(app: Flask, svc: VoiceSvc):
    @app.route("/clone-rt/<user_id>", methods=["POST"])
    def clone_rt_handler(user_id: str):
        if request.content_type not in ["audio/x-wav", "audio/wav", "audio/wave"]:
            return "", 406
        req_data = request.get_data()
        app.logger.info(
            f"clone requested for user {user_id}, request body length: {len(req_data)}"
        )
        model_dest_file = svc.clone(user_id, req_data)
        return jsonify({"model": model_dest_file}), 200


# Convert text to speech using the user's voice model.
def tts_rt_handler(app: Flask, svc: VoiceSvc):
    @app.route("/tts-rt/<user_id>", methods=["POST"])
    def tts_rt_handler(user_id: str):
        if request.content_type not in ["application/json"]:
            return "", 406
        text = request.json["text"]
        transaction_id = str(time.time())
        app.logger.info(
            f"tts requested for user {user_id} and transaction {transaction_id}, request text: {text}"
        )
        tts_output_wav = svc.tts(
            # user_id, transaction_id, text, 80, 0.9, 0.001, 0.6, 0.7, 0.5
            # Kudus to Yonatan for identifying this parameter set:
            user_id, transaction_id, text, 99, 0.8, 0.01, 0.7, 0.6, 0.5
        )
        response = make_response()
        response.headers["content-type"] = "audio/wav"
        response.data = codecs.open(tts_output_wav, "rb").read()
        return response, 200
