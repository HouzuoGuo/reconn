from flask import jsonify, request, Flask


# Read back client's request for debugging.
def readback_handler(app: Flask):
    @app.route("/readback", methods=["GET", "POST", "PUT", "DELETE", "HEAD"])
    def readback_handler():
        return jsonify(
            {
                "request-method": request.method,
                "request-host": request.host,
                "request-url": request.url,
            }
        )
