import huggingface_hub
import logging
import os
import torch
import urllib

from bark_voice_clone.bark.generation import load_codec_model
from bark_voice_clone.hubert.customtokenizer import CustomTokenizer
from bark_voice_clone.hubert.hubert_manager import HuBERTManager
from bark_voice_clone.hubert.pre_kmeans_hubert import CustomHubert


class VoiceSvc:
    def __init__(self):
        self.voice_sample_dir = ""
        self.bark_codec_model = None
        self.hubert_manager = None

    def init_clone(self, ai_computing_device: str, static_resource_dir: str):
        self.bark_codec_model = load_codec_model(use_gpu=ai_computing_device == "cuda")
        self.hubert_manager = HuBERTManager()

        # Download hubert model once.
        hubert_model_dir = os.path.join(static_resource_dir, "hubert-model")
        if not os.path.isdir(hubert_model_dir):
            os.makedirs(hubert_model_dir, exist_ok=True)
        hubert_model_file = os.path.join(hubert_model_dir, "hubert-base-model.pt")
        if not os.path.isfile(hubert_model_file):
            logging.info(f"downloading hubert base model to {hubert_model_file}")
            urllib.request.urlretrieve(
                "https://dl.fbaipublicfiles.com/hubert/hubert_base_ls960.pt",
                hubert_model_file,
            )
            logging.info("finished downloading hubert base model")
        # Download hubert model once.
        hubert_tokeniser_file = os.path.join(
            hubert_model_dir, "quantifier_hubert_base_ls960_14.pth"
        )
        if not os.path.isfile(hubert_tokeniser_file):
            logging.info(f"downloading hubert tokeniser to {hubert_tokeniser_file}")
            huggingface_hub.hf_hub_download(
                "GitMylo/bark-voice-cloning",
                "quantifier_hubert_base_ls960_14.pth",
                local_dir=hubert_model_dir,
                local_dir_use_symlinks=False,
            )
            logging.info("finished downloading hubert tokeniser")

        self.hubert_model = CustomHubert(checkpoint_path=hubert_model_file).to(
            ai_computing_device
        )
        map_location = torch.device("cpu") if ai_computing_device == "cpu" else None
        self.hubert_tokeniser = CustomTokenizer.load_from_checkpoint(
            os.path.join(hubert_model_dir, "quantifier_hubert_base_ls960_14.pth"),
            map_location=map_location,
        ).to(ai_computing_device)

    def init_tts(self, ai_computing_device: str, static_resource_dir: str):
        # TODO
        pass
