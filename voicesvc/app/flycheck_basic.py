from flask import jsonify, request, Flask

def readback(app: Flask):
    @app.route("/readback", methods=["GET", "POST"])
    def handler():
        return jsonify(
            {
                "request-method": request.method,
                "request-host": request.host,
                "request-url": request.url,
            }
        )




