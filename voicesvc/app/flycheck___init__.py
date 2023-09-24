from flask import Flask, jsonify, request

from .basic import readback
def create_app():
    app = Flask(__name__)
    readback(app)
    return app
