import logging
import torchaudio
import os

from encodec.utils import convert_audio
from svc import VoiceSvc
from conf import CONFIG_VOICE_SAMPLE_KEY
from flask import jsonify, request, Flask


# Clone the user's voice using the input wave sample. This is a synchronous handler.
def clone_rt_handler(app: Flask, svc: VoiceSvc):
    @app.route("/clone-sync/<user_id>", methods=["POST"])
    def clone_rt_handler(user_id: str):
        if request.content_type not in ["audio/x-wav", "audio/wav", "audio/wave"]:
            return "", 406
        req_data = request.get_data()
        dest_file = os.path.join(app.config[CONFIG_VOICE_SAMPLE_KEY], user_id + ".wav")
        logging.info(
            f"clone requested for user {user_id}, request body length: {len(req_data)}, file destination: {dest_file}"
        )
        with open(dest_file, "w") as file:
            file.write(req_data)

        wav, sr = torchaudio.load(dest_file)
        wav = convert_audio(
            wav, sr, svc.bark_codec_model.sample_rate, svc.bark_codec_model.channels
        )
        wav = wav.to(app.config[CONFIG_AI_COMPUTING_DEVICE])
        return jsonify(
            {
                "request-method": request.method,
                "request-host": request.host,
                "request-url": request.url,
            }
        )
