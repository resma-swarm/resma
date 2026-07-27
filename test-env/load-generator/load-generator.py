import math
import os
import random
import threading
import time

import requests

DOTNET_URL = os.getenv("DOTNET_URL", "http://dotnet-api:8080")
NODE_URL = os.getenv("NODE_URL", "http://node-api:3000")
PYTHON_URL = os.getenv("PYTHON_URL", "http://python-api:5000")

SERVICES = {
    "dotnet-api": {
        "url": DOTNET_URL,
        "profile": "business",
        "base_interval": 4,
        "cpu_range": (500, 30000),
        "mem_range": (3, 20),
    },
    "node-api": {
        "url": NODE_URL,
        "profile": "spike",
        "base_interval": 3,
        "cpu_range": (1000, 40000),
        "mem_range": (2, 15),
    },
    "python-api": {
        "url": PYTHON_URL,
        "profile": "batch",
        "base_interval": 5,
        "cpu_range": (800, 20000),
        "mem_range": (5, 25),
    },
}

SIM_DAY_SECONDS = 600


def sim_hour(elapsed):
    return (elapsed % SIM_DAY_SECONDS) / (SIM_DAY_SECONDS / 24)


def business_factor(elapsed):
    h = sim_hour(elapsed)
    if 8 <= h <= 18:
        return 0.5 + 0.5 * math.sin(math.pi * (h - 8) / 10)
    if 6 <= h < 8 or 18 < h <= 20:
        return 0.3
    return 0.1


def profile_intensity(profile, elapsed):
    base = 0.5 + 0.3 * math.sin(2 * math.pi * elapsed / 300)

    if profile == "business":
        bf = business_factor(elapsed)
        return base * 0.2 + bf * 0.8

    if profile == "spike":
        spike = 0.9 if random.random() < 0.12 else 0.2
        return base * 0.3 + spike * 0.7

    if profile == "batch":
        burst = 0.9 if (elapsed % 60) < 15 else 0.15
        return base * 0.25 + burst * 0.75

    return base


def generate_load(name, config):
    start = time.time()
    burst_mode = False
    burst_until = 0

    while True:
        elapsed = time.time() - start
        intensity = profile_intensity(config["profile"], elapsed)
        intensity *= random.uniform(0.7, 1.3)
        intensity = max(0.05, min(1.0, intensity))

        if time.time() > burst_until and random.random() < 0.02:
            burst_mode = True
            burst_until = time.time() + random.uniform(10, 25)
            print(f"[burst] {name} entering burst mode for {burst_until - time.time():.0f}s")

        if time.time() < burst_until:
            intensity = min(1.0, intensity * 2.5)

        try:
            cpu_loops = int(
                random.uniform(*config["cpu_range"]) * intensity
            )
            cpu_loops = max(100, cpu_loops)
            requests.get(
                f"{config['url']}/cpu?loops={cpu_loops}",
                timeout=10,
            )

            mb = 0
            if random.random() < 0.6:
                mb = int(
                    random.uniform(*config["mem_range"]) * intensity
                )
                mb = max(1, mb)
                requests.get(
                    f"{config['url']}/memory?mb={mb}",
                    timeout=10,
                )

            mem_label = f"{mb}MB" if mb > 0 else "skip"
            print(
                f"[load] {name} - i={intensity:.2f} "
                f"cpu={cpu_loops} mem={mem_label}"
                f"{' [BURST]' if burst_mode else ''}"
            )
        except Exception as e:
            print(f"[err] {name} - {e}")

        burst_mode = time.time() < burst_until
        interval = config["base_interval"] * (1.5 - intensity * 0.8)
        interval += random.uniform(-0.5, 0.5)
        time.sleep(max(0.5, interval))


def main():
    print("=== RESMA Load Generator v2 ===")
    print(f"Simulated day: {SIM_DAY_SECONDS}s = 24h")
    print()

    threads = []
    for name, config in SERVICES.items():
        print(f"  {name}: profile={config['profile']}")
        t = threading.Thread(
            target=generate_load, args=(name, config), daemon=True
        )
        t.start()
        threads.append(t)

    print("\nLoad generator running. Press Ctrl+C to stop.\n")
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nStopping...")


if __name__ == "__main__":
    main()
