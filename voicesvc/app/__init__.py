from flask import Flask
from .basic import readback_handler
from .clone import CONFIG_VOICE_SAMPLE_KEY, clone_sync_handler


def create_app(voice_sample_dir:str):
    app = Flask(__name__)
    app.config[CONFIG_VOICE_SAMPLE_KEY] = voice_sample_dir
    readback_handler(app)
    clone_sync_handler(app)
    return app
