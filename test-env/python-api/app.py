import math
import time
from flask import Flask, jsonify, request
from workload_sim import WorkloadSimulator

app = Flask(__name__)

_workload = WorkloadSimulator(
    max_cpu_ms=60,
    max_mem_mb=10,
    personality="batch",
)
_workload.start()


@app.route("/")
def index():
    return jsonify({"service": "python-api", "status": "running", "time": time.time()})


@app.route("/cpu")
def cpu():
    loops = int(request.args.get("loops", 1000))
    result = sum(math.sqrt(i) * math.sin(i) for i in range(loops))
    return jsonify({"service": "python-api", "result": result, "loops": loops})


@app.route("/memory")
def memory():
    mb = int(request.args.get("mb", 10))
    data = [bytearray(1024 * 1024) for _ in range(mb)]
    return jsonify({"service": "python-api", "allocatedMB": mb})


@app.route("/health")
def health():
    return jsonify({"status": "healthy", "ts": time.time()})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
