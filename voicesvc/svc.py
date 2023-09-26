import logging
import os
import svc
import basic
import clone
from bark_voice_clone.bark.generation import load_codec_model, generate_text_semantic
from flask import Flask
from bark_voice_clone.hubert.hubert_manager import HuBERTManager
from bark_voice_clone.hubert.pre_kmeans_hubert import CustomHubert
from bark_voice_clone.hubert.customtokenizer import CustomTokenizer
from encodec.utils import convert_audio

CONFIG_VOICE_SAMPLE_KEY = "voice_sample_dir"
CONFIG_STATIC_RESOURCE_KEY = "static_resource_dir"
CONFIG_AI_COMPUTING_DEVICE = "ai_computing_device"


class VoiceSvc:
    def __init__(self):
        self.voice_sample_dir = ""
        self.bark_codec_model = None
        self.hubert_manager = None


def init_clone(app: Flask, svc: VoiceSvc):
    svc.bark_codec_model = load_codec_model(
        use_gpu=app.config[CONFIG_AI_COMPUTING_DEVICE] == "cuda"
    )
    svc.hubert_manager = HuBERTManager()

    # Download hubert model and tokeniser for the first time.
    hubert_model_dir = os.path.join(app.config[CONFIG_VOICE_SAMPLE_KEY], "hubert-model")
    if not os.path.isdir(hubert_model_dir):
        os.makedirs(hubert_model_dir, exist_ok=True)
    hubert_model_file = os.path.join(hubert_model_dir, "hubert-base-model.pt")
    hubert_tokeniser_file = os.path.join(
        hubert_model_dir, "quantifier_hubert_base_ls960_14.pth"
    )
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

    svc.hubert_model = CustomHubert(checkpoint_path=hubert_model_file).to(
        app.config[CONFIG_AI_COMPUTING_DEVICE]
    )
    svc.hubert_tokeniser = CustomTokenizer.load_from_checkpoint(
        "hubert-tokenizer.pth"
    ).to(app.config[CONFIG_AI_COMPUTING_DEVICE])
    pass


def create_app(voice_sample_dir: str, ai_computing_device: str):
    logging.info("initialising flask url handlers")
    app = Flask(__name__)
    app.config[CONFIG_VOICE_SAMPLE_KEY] = voice_sample_dir
    app.config[CONFIG_AI_COMPUTING_DEVICE] = ai_computing_device
    basic.readback_handler(app)
    clone.clone_sync_handler(app)

    logging.info("initialising prerequisites")
    svc = VoiceSvc()
    init_clone(app, svc)
    return app
