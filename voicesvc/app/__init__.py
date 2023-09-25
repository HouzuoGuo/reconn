import logging
from flask import Flask
from .app import VoiceSvc
from .basic import readback_handler
from .clone import CONFIG_VOICE_SAMPLE_KEY, clone_sync_handler, init_clone

def create_app(voice_sample_dir: str):
    logging.info("initialising flask url handlers")
    app = Flask(__name__)
    app.config[CONFIG_VOICE_SAMPLE_KEY] = voice_sample_dir
    readback_handler(app)
    clone_sync_handler(app)

    logging.info("initialising prerequisites")
    svc = VoiceSvc()
    init_clone(app, svc)
    return app
