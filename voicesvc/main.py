#!/usr/bin/env python

from logging.config import dictConfig
import argparse
import logging
import app
import waitress
from svc import VoiceSvc

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
        "--ai_computing_device",
        help="computing device for AI workload - cpu or cuda",
        default="cuda",
    )
    parser.add_argument(
        "--static_resource_dir",
        help="path to the directory of static resources (e.g. base hubert model & tokeniser)",
        default="/tmp/voice_static_resource_dir",
    )
    parser.add_argument(
        "--voice_sample_dir",
        help="path to the directory of incoming user voice samples",
        default="/tmp/voice_sample_dir",
    )
    parser.add_argument(
        "--voice_model_dir",
        help="path to the directory of constructed user voice models",
        default="/tmp/voice_model_dir",
    )
    parser.add_argument(
        "--voice_temp_model_dir",
        help="path to the directory of temporary user voice models used during TTS",
        default="/tmp/voice_temp_model_dir",
    )
    parser.add_argument(
        "--voice_output_dir",
        help="path to the directory of TTS output files",
        default="/tmp/voice_output_dir",
    )
    args = parser.parse_args()

    logging.basicConfig(
        format="%(asctime)s %(levelname)-8s %(message)s",
        level=logging.NOTSET,
        datefmt="%Y-%m-%d %H:%M:%S",
    )
    logging.root.setLevel(logging.NOTSET)
    logging.info(f"initialising voice service")

    voice_svc = VoiceSvc(
        args.ai_computing_device,
        args.static_resource_dir,
        args.voice_sample_dir,
        args.voice_model_dir,
        args.voice_temp_model_dir,
        args.voice_output_dir,
    )
    voice_svc.init_clone()
    voice_svc.init_tts()

    flask_app = app.create_app(voice_svc)
    if args.debug:
        logging.info(
            f"starting flask built-in development web server on {args.address}:{args.port}"
        )
        flask_app.run(host=args.address, port=args.port, debug=args.debug)
    else:
        logging.info(f"starting waitress wsgi web server on {args.address}:{args.port}")
        waitress.server.create_server(
            flask_app, host=args.address, port=args.port
        ).run()
