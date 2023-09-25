#!/usr/bin/env python

import argparse
import logging
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
    parser.add_argument(
        "--voice_sample_dir",
        help="path to the directory of incoming user voice samples",
        default="/tmp/voice_sample_dir",
    )
    args = parser.parse_args()

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.NOTSET,
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    logging.info(
        f"about to start voice web service on {args.address}:{args.port} using {args.voice_sample_dir} for voice sample storage"
    )

    app = create_app(args.voice_sample_dir)
    app.run(host=args.address, port=args.port, debug=args.debug)
