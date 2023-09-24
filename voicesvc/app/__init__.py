from flask import Flask
from .basic import readback


def create_app():
    app = Flask(__name__)
    readback(app)
    return app
