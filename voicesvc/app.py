import basic
import clone
import logging
import basic
import clone

from flask import Flask
from conf import (
    CONFIG_AI_COMPUTING_DEVICE,
    CONFIG_STATIC_RESOURCE_KEY,
    CONFIG_VOICE_MODEL_KEY,
    CONFIG_VOICE_SAMPLE_KEY,
)
from svc import VoiceSvc


def create_app(
    voice_sample_dir: str,
    voice_model_dir: str,
    static_resource_dir: str,
    ai_computing_device: str,
    svc: VoiceSvc,
):
    logging.info("initialising flask url handlers")
    app = Flask(__name__)
    app.config[CONFIG_VOICE_SAMPLE_KEY] = voice_sample_dir
    app.config[CONFIG_VOICE_MODEL_KEY] = voice_model_dir
    app.config[CONFIG_STATIC_RESOURCE_KEY] = static_resource_dir
    app.config[CONFIG_AI_COMPUTING_DEVICE] = ai_computing_device
    basic.readback_handler(app)
    clone.clone_rt_handler(app, svc)
    clone.tts_rt_handler(app, svc)

    logging.info("initialising prerequisites")
    logging.info(f"using {ai_computing_device} for AI computing")
    logging.info(f"using {voice_sample_dir} for voice sample storage")
    logging.info(f"using {voice_model_dir} for voice model storage")
    logging.info(f"using {static_resource_dir} for static resources")
    return app
