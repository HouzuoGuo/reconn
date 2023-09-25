import urllib
import logging
import os.path
import urllib.request
import huggingface_hub

from flask import jsonify, request, Flask
from bark_voice_clone.bark.generation import load_codec_model, generate_text_semantic
from bark_voice_clone.hubert.hubert_manager import HuBERTManager
from bark_voice_clone.hubert.pre_kmeans_hubert import CustomHubert
from bark_voice_clone.hubert.customtokenizer import CustomTokenizer
from encodec.utils import convert_audio
from .app import VoiceSvc


CONFIG_VOICE_SAMPLE_KEY = "voice_sample_dir"
CONFIG_STATIC_RESOURCE_KEY = "static_resource_dir"


def init_clone(app: Flask, svc: VoiceSvc):
    svc.bark_codec_model = load_codec_model(use_gpu=True)
    svc.hubert_manager = HuBERTManager()

    # Download hubert model and tokeniser for the first time.
    hubert_model_dir = os.path.join(app.config[CONFIG_VOICE_SAMPLE_KEY], "hubert-model")
    hubert_model_file = os.path.join(hubert_model_dir, "hubert-base-model.pt")
    hubert_tokeniser_file = os.path.join(
        hubert_model_dir, "quantifier_hubert_base_ls960_14.pth"
    )

    if not os.path.isdir(hubert_model_dir):
        os.makedirs(hubert_model_dir, exist_ok=True)
    if not os.path.isfile(hubert_model_file):
        logging.info(f"downloading hubert base model to {hubert_model_file}")
        urllib.request.urlretrieve(
            "https://dl.fbaipublicfiles.com/hubert/hubert_base_ls960.pt",
            hubert_model_file,
        )
        logging.info("finished downloading hubert base model")
    if not os.path.isfile(hubert_tokeniser_file):
        logging.info(f"downloading hubert tokeniser to {hubert_tokeniser_file}")
        huggingface_hub.hf_hub_download(
            "GitMylo/bark-voice-cloning",
            "quantifier_hubert_base_ls960_14.pth",
            local_dir=hubert_model_dir,
            local_dir_use_symlinks=False,
        )
        logging.info("finished downloading hubert tokeniser")

    svc.hubert_model = CustomHubert(checkpoint_path=hubert_model_file).to("cuda")
    svc.hubert_tokeniser = CustomTokenizer.load_from_checkpoint(
        "hubert-tokenizer.pth"
    ).to(device)
    pass


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
