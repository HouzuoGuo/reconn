import logging
import numpy
import torchaudio
import torch
import os

from encodec.utils import convert_audio
from svc import VoiceSvc
from conf import (
    CONFIG_VOICE_SAMPLE_KEY,
    CONFIG_AI_COMPUTING_DEVICE,
    CONFIG_VOICE_MODEL_KEY,
)
from flask import jsonify, request, Flask


# Clone the user's voice using the input wave sample. This is a synchronous handler.
def clone_rt_handler(app: Flask, svc: VoiceSvc):
    @app.route("/clone-rt/<user_id>", methods=["POST"])
    def clone_rt_handler(user_id: str):
        if request.content_type not in ["audio/x-wav", "audio/wav", "audio/wave"]:
            return "", 406
        req_data = request.get_data()
        sample_dest_file = os.path.join(
            app.config[CONFIG_VOICE_SAMPLE_KEY], user_id + ".wav"
        )
        logging.info(
            f"clone requested for user {user_id}, request body length: {len(req_data)}, file destination: {sample_dest_file}"
        )
        with open(sample_dest_file, "wb") as file:
            file.write(req_data)

        wav, sr = torchaudio.load(sample_dest_file)
        wav = convert_audio(
            wav, sr, svc.bark_codec_model.sample_rate, svc.bark_codec_model.channels
        )
        wav = wav.to(app.config[CONFIG_AI_COMPUTING_DEVICE])
        semantic_vectors = svc.hubert_model.forward(
            wav, input_sample_hz=svc.bark_codec_model.sample_rate
        )
        semantic_tokens = svc.hubert_tokeniser.get_token(semantic_vectors)
        with torch.no_grad():
            encoded_frames = svc.bark_codec_model.encode(wav.unsqueeze(0))
        codes = torch.cat([encoded[0] for encoded in encoded_frames], dim=-1).squeeze()
        codes = codes.cpu().numpy()
        semantic_tokens = semantic_tokens.cpu().numpy()
        model_dest_file = os.path.join(
            app.config[CONFIG_VOICE_MODEL_KEY], user_id + ".npz"
        )
        numpy.savez(
            model_dest_file,
            fine_prompt=codes,
            coarse_prompt=codes[:2, :],
            semantic_prompt=semantic_tokens,
        )
        return jsonify({"sample": sample_dest_file, "model": model_dest_file})


def tts_rt_handler(app: Flask, svc: VoiceSvc):
    @app.route("/tts-rt/<user_id>", methods=["POST"])
    def tts_rt_handler(user_id: str):
        if request.content_type not in ["application/json"]:
            return "", 406
        req_data = request.get_data()
        user_voice_model = os.path.join(
            app.config[CONFIG_VOICE_MODEL_KEY], user_id + ".npz"
        )
        logging.info(
            f"tts requested for user {user_id}, model path: {user_voice_model}"
        )
