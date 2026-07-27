from flask import Flask, jsonify, request
import gc

app = Flask(__name__)
_held = []


@app.route("/health")
def health():
    return jsonify(status="ok")


@app.route("/oom")
def oom():
    mb = int(request.args.get("mb", 50))
    chunk = bytearray(mb * 1024 * 1024)
    _held.append(chunk)
    return jsonify(allocated=mb, total_mb=len(_held))


@app.route("/reset")
def reset():
    _held.clear()
    gc.collect()
    return jsonify(cleared=True)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
