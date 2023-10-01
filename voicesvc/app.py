import basic
import clone
import logging
import basic
import clone

from flask import Flask, request
from svc import VoiceSvc


def create_app(svc: VoiceSvc):
    app = Flask(__name__)
    app.logger.info("initialising flask url handlers")

    @app.before_request
    def log_before_request():
        app.logger.info(
            f"received request {request.host} {request.method} {request.url}"
        )

    @app.after_request
    def log_after_request(response):
        app.logger.info(
            f"{request.host} {request.method} {request.url} received response status {response.status} and {len(response.data)} bytes"
        )
        return response

    basic.readback_handler(app)
    clone.clone_rt_handler(app, svc)
    clone.tts_rt_handler(app, svc)

    return app
