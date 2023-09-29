import basic
import clone
import logging
import basic
import clone

from flask import Flask
from svc import VoiceSvc


def create_app(svc: VoiceSvc):
    logging.info("initialising flask url handlers")
    app = Flask(__name__)
    basic.readback_handler(app)
    clone.clone_rt_handler(app, svc)
    clone.tts_rt_handler(app, svc)

    return app
