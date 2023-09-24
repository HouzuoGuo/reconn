#!/usr/bin/env python

import argparse
from app import create_app

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="voicesvc", description="Voice clone, inference, and TTS services."
    )
    parser.add_argument(
        "--address", help="web server listener address", default="127.0.0.1"
    )
    parser.add_argument("--port", help="web server port number", type=int, default=8081)
    parser.add_argument(
        "--debug", help="start flask server in debug mode", action="store_true"
    )
    args = parser.parse_args()
    app = create_app()
    app.run(host=args.address, port=args.port, debug=args.debug)
