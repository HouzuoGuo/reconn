import huggingface_hub
import logging
import nltk
import numpy
import os
import torch
import torchaudio
import urllib

from bark_voice_clone.bark.generation import (
    load_codec_model,
    preload_models,
    generate_text_semantic,
)
from scipy.io.wavfile import write as write_wav
from bark_voice_clone.bark.api import semantic_to_waveform
from bark_voice_clone.bark.generation import load_codec_model, SAMPLE_RATE
from bark_voice_clone.hubert.customtokenizer import CustomTokenizer
from bark_voice_clone.hubert.hubert_manager import HuBERTManager
from bark_voice_clone.hubert.pre_kmeans_hubert import CustomHubert
from encodec.utils import convert_audio


class VoiceSvc:
    def __init__(
        self,
        ai_computing_device: str,
        static_resource_dir: str,
        voice_sample_dir: str,
        voice_model_dir: str,
        voice_temp_model_dir: str,
        voice_output_dir: str,
    ):
        self.bark_codec_model = None
        self.hubert_manager = None
        self.ai_computing_device = ai_computing_device
        self.static_resource_dir = static_resource_dir
        self.voice_sample_dir = voice_sample_dir
        self.voice_model_dir = voice_model_dir
        self.voice_temp_model_dir = voice_temp_model_dir
        self.voice_output_dir = voice_output_dir
        logging.info(f"using {ai_computing_device} for AI computing")
        logging.info(
            f"using {os.path.abspath(static_resource_dir)} for static resources"
        )
        logging.info(
            f"using {os.path.abspath(voice_sample_dir)} for voice sample storage"
        )
        logging.info(
            f"using {os.path.abspath(voice_model_dir)} for voice model storage"
        )
        logging.info(
            f"using {os.path.abspath(voice_temp_model_dir)} for temporary voice model storage"
        )
        logging.info(
            f"using {os.path.abspath(voice_output_dir)} for TTS output storage"
        )

    def init_clone(self):
        self.bark_codec_model = load_codec_model(
            use_gpu=self.ai_computing_device == "cuda"
        )
        self.hubert_manager = HuBERTManager()

        # Download hubert model once.
        hubert_model_dir = os.path.join(self.static_resource_dir, "hubert-model")
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
            self.ai_computing_device
        )
        map_location = (
            torch.device("cpu") if self.ai_computing_device == "cpu" else None
        )
        self.hubert_tokeniser = CustomTokenizer.load_from_checkpoint(
            os.path.join(hubert_model_dir, "quantifier_hubert_base_ls960_14.pth"),
            map_location=map_location,
        ).to(self.ai_computing_device)
        # TODO: download these files to avoid repetition and wasted bandwidth:
        # 2023-09-29 18:19:28 | INFO | bark_voice_clone.bark.generation | coarse model not found, downloading into `C:\Users\guoho\.cache\serp\bark_v0`.
        # 2023-09-29 18:19:28 | DEBUG | urllib3.connectionpool | https://huggingface.co:443 "HEAD /suno/bark/resolve/main/coarse_2.pt HTTP/1.1" 302 0
        # 2023-09-29 18:19:34 | INFO | bark_voice_clone.bark.generation | model loaded: 314.4M params, 2.901 loss
        # 2023-09-29 18:19:35 | INFO | bark_voice_clone.bark.generation | fine model not found, downloading into `C:\Users\guoho\.cache\serp\bark_v0`.

    def init_tts(self):
        logging.info(f"downloading nltk punctuation data")
        nltk_data_dir = os.path.join(self.static_resource_dir, "nltk-data")
        if not os.path.isdir(nltk_data_dir):
            os.makedirs(nltk_data_dir, exist_ok=True)
        os.environ["NLTK_DATA"] = nltk_data_dir
        nltk.download("punkt")
        logging.info(f"finished downloading nltk punctuation data")
        logging.info(f"preloading bark models")
        logging.info(f"finished preloading bark models")
        # TODO: change *_use_small to True for test cases running on CPU.
        preload_models(
            text_use_gpu=self.ai_computing_device == "cuda",
            text_use_small=False,
            coarse_use_gpu=self.ai_computing_device == "cuda",
            coarse_use_small=False,
            fine_use_gpu=self.ai_computing_device == "cuda",
            fine_use_small=False,
            codec_use_gpu=self.ai_computing_device == "cuda",
            force_reload=False,
            path=self.voice_model_dir,
        )

    def clone(self, user_id, bin_sample_wav) -> str:
        logging.info(
            f"cloning user ID {user_id} based on sample data size {len(bin_sample_wav)}"
        )
        sample_dest_file = os.path.join(self.voice_sample_dir, f"{user_id}.wav")
        with open(sample_dest_file, "wb") as file:
            file.write(bin_sample_wav)

        wav, sr = torchaudio.load(sample_dest_file)
        wav = convert_audio(
            wav, sr, self.bark_codec_model.sample_rate, self.bark_codec_model.channels
        )
        wav = wav.to(self.ai_computing_device)
        semantic_vectors = self.hubert_model.forward(
            wav, input_sample_hz=self.bark_codec_model.sample_rate
        )
        semantic_tokens = self.hubert_tokeniser.get_token(semantic_vectors)
        with torch.no_grad():
            encoded_frames = self.bark_codec_model.encode(wav.unsqueeze(0))
        codes = torch.cat([encoded[0] for encoded in encoded_frames], dim=-1).squeeze()
        codes = codes.cpu().numpy()
        semantic_tokens = semantic_tokens.cpu().numpy()
        base_name = user_id + ".npz"
        model_dest_file = os.path.join(self.voice_model_dir, base_name)
        numpy.savez(
            model_dest_file,
            fine_prompt=codes,
            coarse_prompt=codes[:2, :],
            semantic_prompt=semantic_tokens,
        )
        return base_name

    def tts(
        self,
        user_id: str,
        transaction_id: str,
        text_prompt: str,
        top_k: float,
        top_p: float,
        min_eos_p: float,
        semantic_temp: float,
        waveform_temp: float,
        fine_temp: float,
    ) -> str:
        voice_segments = []
        # Voice model cloned from user's sample.
        original_model = os.path.join(self.voice_model_dir, f"{user_id}.npz")
        # Temporary model created during this TTS invocation.
        temp_model = os.path.join(
            self.voice_temp_model_dir, f"{user_id}-{transaction_id}.npz"
        )
        tts_output_wav = os.path.join(
            self.voice_output_dir, f"{user_id}-{transaction_id}.wav"
        )
        active_model = original_model
        for index, sentence in enumerate(nltk.sent_tokenize(text_prompt)):
            logging.info(
                f'converting sentence "{sentence}" for {user_id} transaction {transaction_id}'
            )
            x_semantic = generate_text_semantic(
                sentence,
                history_prompt=active_model,
                use_kv_caching=True,
                temp=semantic_temp,
                top_k=top_k,
                top_p=top_p,
                min_eos_p=min_eos_p,
                silent=True,
            )
            full_out, audio_array = semantic_to_waveform(
                x_semantic,
                history_prompt=active_model,
                temp=waveform_temp,
                output_full=True,
                fine_temp=fine_temp,
                silent=True,
            )
            voice_segments += [audio_array]

            if index < 1:
                # Save the model from the first sentence and use it in subsequent sentences.
                numpy.savez_compressed(
                    temp_model,
                    semantic_prompt=full_out["semantic_prompt"],
                    coarse_prompt=full_out["coarse_prompt"],
                    fine_prompt=full_out["fine_prompt"],
                )
                active_model = temp_model
        os.remove(temp_model)
        write_wav(tts_output_wav, SAMPLE_RATE, numpy.concatenate(voice_segments))
        # TODO FIXME: periodically clean up left over temporary models
        return tts_output_wav
