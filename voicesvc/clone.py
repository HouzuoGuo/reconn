import urllib
import logging
import urllib.request

from flask import jsonify, request, Flask
from bark_voice_clone.bark.generation import load_codec_model, generate_text_semantic
from bark_voice_clone.hubert.hubert_manager import HuBERTManager
from bark_voice_clone.hubert.pre_kmeans_hubert import CustomHubert
from bark_voice_clone.hubert.customtokenizer import CustomTokenizer
from encodec.utils import convert_audio


# Clone the user's voice using the input wave sample. This is a synchronous handler.
def clone_sync_handler(app: Flask):
    @app.route("/clone-sync/<user_id>", methods=["POST"])
    def clone_sync_handler(user_id: str):
        if request.content_type not in ["audio/x-wav", "audio/wav", "audio/wave"]:
            return "", 406
        req_data = request.get_data()
        logging.info(f"clone request data length: {len(req_data)}")
        return jsonify(
            {
                "request-method": request.method,
                "request-host": request.host,
                "request-url": request.url,
            }
        )
